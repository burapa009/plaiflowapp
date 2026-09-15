package auth

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type fakeProvider struct {
	attempt  Attempt
	identity ExternalIdentity
}

func (p *fakeProvider) Name() string   { return p.identity.Provider }
func (p *fakeProvider) Issuer() string { return p.identity.Issuer }
func (p *fakeProvider) AuthorizationURL(attempt Attempt) (string, error) {
	p.attempt = attempt
	return "https://provider.example/authorize?state=" + url.QueryEscape(attempt.State), nil
}
func (p *fakeProvider) Exchange(context.Context, string, string, string) (ExternalIdentity, error) {
	return p.identity, nil
}

type memoryAuthStore struct {
	transaction *Transaction
	identity    ExternalIdentity
	session     SessionRecord
	resolved    Session
	consumed    bool
}

func (s *memoryAuthStore) CreateAuthTransaction(_ context.Context, transaction Transaction) error {
	s.transaction = &transaction
	return nil
}
func (s *memoryAuthStore) ConsumeAuthTransaction(_ context.Context, stateHash []byte, now time.Time) (Transaction, error) {
	if s.transaction == nil || s.consumed || !bytes.Equal(s.transaction.StateHash, stateHash) || !s.transaction.ExpiresAt.After(now) {
		return Transaction{}, ErrAuthTransaction
	}
	s.consumed = true
	return *s.transaction, nil
}
func (s *memoryAuthStore) Login(_ context.Context, identity ExternalIdentity, _, _ string, _ time.Time) (LoginResult, error) {
	s.identity = identity
	return LoginResult{UserID: "user-1", Created: true}, nil
}
func (s *memoryAuthStore) CreateSession(_ context.Context, session SessionRecord) error {
	s.session = session
	return nil
}
func (s *memoryAuthStore) ResolveSession(context.Context, []byte, time.Time) (Session, error) {
	if s.resolved.ID == "" {
		return Session{}, ErrSession
	}
	return s.resolved, nil
}

func TestProviderLinkRequiresSameOriginCSRFAndRecentSession(t *testing.T) {
	now := time.Date(2026, 9, 15, 11, 0, 0, 0, time.UTC)
	provider := &fakeProvider{identity: ExternalIdentity{Provider: "google", Issuer: googleIssuer, Subject: "google-user"}}
	store := &memoryAuthStore{resolved: Session{
		ID: "session-1", UserID: "user-1", CSRFHash: hashSecret("csrf-token"), AuthenticatedAt: now.Add(-time.Minute),
		AuthenticatedProvider: "line", IdleExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(24 * time.Hour),
	}}
	service, err := NewService(ServiceConfig{WebOrigin: "https://app.example", Providers: []Provider{provider}, Now: func() time.Time { return now }}, store)
	if err != nil {
		t.Fatal(err)
	}

	denied := httptest.NewRequest(http.MethodPost, "https://app.example/v1/auth/google/link", nil)
	denied.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "session-token"})
	denied.Header.Set("X-CSRF-Token", "csrf-token")
	deniedResponse := httptest.NewRecorder()
	service.ServeHTTP(deniedResponse, denied)
	if deniedResponse.Code != http.StatusForbidden || store.transaction != nil {
		t.Fatalf("cross-origin mutation status=%d transaction=%+v", deniedResponse.Code, store.transaction)
	}

	allowed := httptest.NewRequest(http.MethodPost, "https://app.example/v1/auth/google/link?return_to=/account", nil)
	allowed.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "session-token"})
	allowed.Header.Set("Origin", "https://app.example")
	allowed.Header.Set("X-CSRF-Token", "csrf-token")
	allowedResponse := httptest.NewRecorder()
	service.ServeHTTP(allowedResponse, allowed)
	if allowedResponse.Code != http.StatusSeeOther || store.transaction == nil {
		t.Fatalf("link status=%d transaction=%+v", allowedResponse.Code, store.transaction)
	}
	if store.transaction.Purpose != PurposeLink || store.transaction.InitiatingUserID != "user-1" || store.transaction.InitiatingSessionID != "session-1" {
		t.Fatalf("link transaction = %+v", store.transaction)
	}
}
func (s *memoryAuthStore) RevokeSession(context.Context, string, time.Time) error { return nil }
func (s *memoryAuthStore) RevokeUserSessions(context.Context, string, time.Time) error {
	return nil
}
func (s *memoryAuthStore) LinkIdentityAndRotate(context.Context, LinkRequest) error { return nil }
func (s *memoryAuthStore) ReauthenticateAndRotate(context.Context, ReauthRequest) error {
	return nil
}
func (s *memoryAuthStore) UnlinkIdentityAndRotate(context.Context, UnlinkRequest) error {
	return nil
}
func (s *memoryAuthStore) ListIdentities(context.Context, string) ([]IdentityState, error) {
	return nil, nil
}

func TestLoginCallbackCreatesSecureSessionAndRejectsReplay(t *testing.T) {
	now := time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)
	provider := &fakeProvider{identity: ExternalIdentity{Provider: "line", Issuer: lineIssuer, Subject: "line-user"}}
	store := &memoryAuthStore{}
	service, err := NewService(ServiceConfig{
		WebOrigin: "https://app.example", Providers: []Provider{provider}, Now: func() time.Time { return now },
	}, store)
	if err != nil {
		t.Fatal(err)
	}

	start := httptest.NewRequest(http.MethodGet, "https://app.example/v1/auth/line/start?return_to=/setup", nil)
	startResponse := httptest.NewRecorder()
	service.ServeHTTP(startResponse, start)
	if startResponse.Code != http.StatusSeeOther || store.transaction == nil {
		t.Fatalf("start status=%d transaction=%+v", startResponse.Code, store.transaction)
	}
	if store.transaction.Provider != "line" || store.transaction.Issuer != lineIssuer || store.transaction.Purpose != PurposeLogin || store.transaction.ReturnTo != "/setup" {
		t.Fatalf("transaction = %+v", store.transaction)
	}

	var browserCookie *http.Cookie
	for _, cookie := range startResponse.Result().Cookies() {
		if cookie.Name == OAuthCookieName {
			browserCookie = cookie
		}
	}
	if browserCookie == nil || !browserCookie.HttpOnly || !browserCookie.Secure || browserCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("oauth cookie = %+v", browserCookie)
	}

	callbackURL := "https://app.example/v1/auth/line/callback?code=code&state=" + url.QueryEscape(provider.attempt.State)
	callback := httptest.NewRequest(http.MethodGet, callbackURL, nil)
	callback.AddCookie(browserCookie)
	callbackResponse := httptest.NewRecorder()
	service.ServeHTTP(callbackResponse, callback)
	if callbackResponse.Code != http.StatusSeeOther || callbackResponse.Header().Get("Location") != "/setup" {
		t.Fatalf("callback status=%d location=%q", callbackResponse.Code, callbackResponse.Header().Get("Location"))
	}
	if store.identity.Subject != "line-user" || store.session.UserID != "user-1" {
		t.Fatalf("identity=%+v session=%+v", store.identity, store.session)
	}
	cookies := callbackResponse.Result().Cookies()
	var sessionCookie, csrfCookie *http.Cookie
	for _, cookie := range cookies {
		switch cookie.Name {
		case SessionCookieName:
			sessionCookie = cookie
		case CSRFCookieName:
			csrfCookie = cookie
		}
	}
	if sessionCookie == nil || !sessionCookie.HttpOnly || !sessionCookie.Secure || sessionCookie.Domain != "" || sessionCookie.Path != "/" || sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("session cookie = %+v", sessionCookie)
	}
	if csrfCookie == nil || csrfCookie.HttpOnly || !csrfCookie.Secure || len(store.session.CSRFHash) == 0 {
		t.Fatalf("csrf cookie=%+v session=%+v", csrfCookie, store.session)
	}

	replay := httptest.NewRequest(http.MethodGet, callbackURL, nil)
	replay.AddCookie(browserCookie)
	replayResponse := httptest.NewRecorder()
	service.ServeHTTP(replayResponse, replay)
	if replayResponse.Code != http.StatusSeeOther || !strings.HasPrefix(replayResponse.Header().Get("Location"), "/auth/error?code=expired_attempt") {
		t.Fatalf("replay status=%d location=%q", replayResponse.Code, replayResponse.Header().Get("Location"))
	}
}
