package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	SessionCookieName = "__Host-plaiflow-session"
	CSRFCookieName    = "__Host-plaiflow-csrf"
	OAuthCookieName   = "__Host-plaiflow-oauth"
)

var (
	ErrAuthTransaction  = errors.New("auth transaction is unavailable")
	ErrSession          = errors.New("session is unavailable")
	ErrIdentityConflict = errors.New("identity belongs to another user")
	ErrLastIdentity     = errors.New("last identity cannot be unlinked")
	ErrRecentAuth       = errors.New("recent authentication is required")
)

type Purpose string

const (
	PurposeLogin  Purpose = "login"
	PurposeLink   Purpose = "link"
	PurposeReauth Purpose = "reauth"
)

type Transaction struct {
	StateHash           []byte
	BrowserHash         []byte
	Provider            string
	Issuer              string
	Purpose             Purpose
	Nonce               string
	CodeVerifier        string
	ReturnTo            string
	InitiatingUserID    string
	InitiatingSessionID string
	CreatedAt           time.Time
	ExpiresAt           time.Time
}

type LoginResult struct {
	UserID  string
	Created bool
}

type SessionRecord struct {
	ID                    string
	UserID                string
	TokenHash             []byte
	CSRFHash              []byte
	CreatedAt             time.Time
	AuthenticatedAt       time.Time
	LastSeenAt            time.Time
	IdleExpiresAt         time.Time
	AbsoluteExpiresAt     time.Time
	AuthenticatedProvider string
}

type Session struct {
	ID                    string
	UserID                string
	CSRFHash              []byte
	AuthenticatedAt       time.Time
	AuthenticatedProvider string
	IdleExpiresAt         time.Time
	AbsoluteExpiresAt     time.Time
}

type LinkRequest struct {
	UserID, SessionID string
	Identity          ExternalIdentity
	NewSession        SessionRecord
	Now               time.Time
}

type ReauthRequest struct {
	UserID, SessionID string
	Identity          ExternalIdentity
	NewSession        SessionRecord
	Now               time.Time
}

type UnlinkRequest struct {
	UserID, SessionID, Provider string
	NewSession                  SessionRecord
	Now, CooldownUntil          time.Time
}

type IdentityState struct {
	Provider string `json:"provider"`
	Linked   bool   `json:"linked"`
}

type Store interface {
	CreateAuthTransaction(context.Context, Transaction) error
	ConsumeAuthTransaction(context.Context, []byte, time.Time) (Transaction, error)
	Login(context.Context, ExternalIdentity, string, string, time.Time) (LoginResult, error)
	CreateSession(context.Context, SessionRecord) error
	ResolveSession(context.Context, []byte, time.Time) (Session, error)
	RevokeSession(context.Context, string, time.Time) error
	RevokeUserSessions(context.Context, string, time.Time) error
	LinkIdentityAndRotate(context.Context, LinkRequest) error
	ReauthenticateAndRotate(context.Context, ReauthRequest) error
	UnlinkIdentityAndRotate(context.Context, UnlinkRequest) error
	ListIdentities(context.Context, string) ([]IdentityState, error)
}

type ServiceConfig struct {
	WebOrigin string
	Providers []Provider
	Now       func() time.Time
}

type Service struct {
	webOrigin string
	providers map[string]Provider
	store     Store
	now       func() time.Time
	mux       *http.ServeMux
}

func NewService(config ServiceConfig, store Store) (*Service, error) {
	origin, err := url.Parse(config.WebOrigin)
	if err != nil || origin.Host == "" || origin.User != nil || origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" || (origin.Scheme != "https" && !(origin.Scheme == "http" && isLocalhost(origin.Hostname()))) {
		return nil, errors.New("invalid web origin")
	}
	if store == nil {
		return nil, errors.New("auth store is required")
	}
	service := &Service{webOrigin: origin.String(), providers: map[string]Provider{}, store: store, now: config.Now, mux: http.NewServeMux()}
	if service.now == nil {
		service.now = time.Now
	}
	for _, provider := range config.Providers {
		if provider != nil {
			service.providers[provider.Name()] = provider
		}
	}
	service.mux.HandleFunc("GET /v1/auth/{provider}/start", service.startLogin)
	service.mux.HandleFunc("POST /v1/auth/{provider}/link", service.startLink)
	service.mux.HandleFunc("POST /v1/auth/{provider}/reauth", service.startReauth)
	service.mux.HandleFunc("POST /v1/auth/{provider}/unlink", service.unlink)
	service.mux.HandleFunc("GET /v1/auth/{provider}/callback", service.callback)
	service.mux.HandleFunc("GET /v1/session", service.currentSession)
	service.mux.HandleFunc("POST /v1/logout", service.logout)
	service.mux.HandleFunc("POST /v1/logout-all", service.logoutAll)
	return service, nil
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

func (s *Service) startLogin(w http.ResponseWriter, r *http.Request) {
	s.beginAuthorization(w, r, PurposeLogin, nil)
}

func (s *Service) startLink(w http.ResponseWriter, r *http.Request) {
	s.startProtected(w, r, PurposeLink)
}

func (s *Service) startReauth(w http.ResponseWriter, r *http.Request) {
	s.startProtected(w, r, PurposeReauth)
}

func (s *Service) startProtected(w http.ResponseWriter, r *http.Request, purpose Purpose) {
	session, err := s.Authenticate(r)
	if err != nil {
		writeAuthError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !s.ValidMutation(r, session) {
		writeAuthError(w, http.StatusForbidden, "csrf_rejected")
		return
	}
	if s.now().UTC().Sub(session.AuthenticatedAt) > 10*time.Minute {
		writeAuthError(w, http.StatusForbidden, "recent_auth_required")
		return
	}
	s.beginAuthorization(w, r, purpose, &session)
}

func (s *Service) beginAuthorization(w http.ResponseWriter, r *http.Request, purpose Purpose, session *Session) {
	provider := s.providers[r.PathValue("provider")]
	if provider == nil {
		s.errorRedirect(w, r, "provider_unavailable")
		return
	}
	returnTo, ok := safeReturnPath(r.URL.Query().Get("return_to"))
	if !ok {
		s.errorRedirect(w, r, "invalid_return")
		return
	}
	attempt, err := NewAttempt()
	if err != nil {
		s.errorRedirect(w, r, "provider_unavailable")
		return
	}
	browserSecret, err := randomToken()
	if err != nil {
		s.errorRedirect(w, r, "provider_unavailable")
		return
	}
	now := s.now().UTC()
	transaction := Transaction{
		StateHash: hashSecret(attempt.State), BrowserHash: hashSecret(browserSecret), Provider: provider.Name(), Issuer: provider.Issuer(),
		Purpose: purpose, Nonce: attempt.Nonce, CodeVerifier: attempt.CodeVerifier, ReturnTo: returnTo,
		CreatedAt: now, ExpiresAt: now.Add(10 * time.Minute),
	}
	if session != nil {
		transaction.InitiatingUserID = session.UserID
		transaction.InitiatingSessionID = session.ID
	}
	if err := s.store.CreateAuthTransaction(r.Context(), transaction); err != nil {
		s.errorRedirect(w, r, "provider_unavailable")
		return
	}
	location, err := provider.AuthorizationURL(attempt)
	if err != nil {
		s.errorRedirect(w, r, "provider_unavailable")
		return
	}
	http.SetCookie(w, secureCookie(OAuthCookieName, browserSecret, 10*time.Minute, true))
	http.Redirect(w, r, location, http.StatusSeeOther)
}

func (s *Service) callback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	transaction, err := s.store.ConsumeAuthTransaction(r.Context(), hashSecret(state), s.now().UTC())
	if err != nil || state == "" {
		s.errorRedirect(w, r, "expired_attempt")
		return
	}
	provider := s.providers[r.PathValue("provider")]
	if provider == nil || transaction.Provider != provider.Name() || transaction.Issuer != provider.Issuer() {
		s.errorRedirect(w, r, "identity_conflict")
		return
	}
	browser, err := r.Cookie(OAuthCookieName)
	if err != nil || !hmac.Equal(hashSecret(browser.Value), transaction.BrowserHash) {
		s.errorRedirect(w, r, "expired_attempt")
		return
	}
	if r.URL.Query().Get("error") != "" {
		s.errorRedirect(w, r, "access_denied")
		return
	}
	identity, err := provider.Exchange(r.Context(), r.URL.Query().Get("code"), transaction.CodeVerifier, transaction.Nonce)
	if err != nil || identity.Provider != provider.Name() || identity.Issuer != provider.Issuer() || identity.Subject == "" {
		s.errorRedirect(w, r, "provider_unavailable")
		return
	}
	now := s.now().UTC()
	switch transaction.Purpose {
	case PurposeLogin:
		result, loginErr := s.store.Login(r.Context(), identity, newID(), newID(), now)
		if loginErr != nil {
			s.errorRedirect(w, r, "identity_conflict")
			return
		}
		if err := s.issueSession(w, r.Context(), result.UserID, identity.Provider, now); err != nil {
			s.errorRedirect(w, r, "provider_unavailable")
			return
		}
	case PurposeLink, PurposeReauth:
		session, authErr := s.Authenticate(r)
		if authErr != nil || session.UserID != transaction.InitiatingUserID || session.ID != transaction.InitiatingSessionID {
			s.errorRedirect(w, r, "expired_attempt")
			return
		}
		credentials, credentialErr := newSessionCredentials(session.UserID, identity.Provider, now)
		if credentialErr != nil {
			s.errorRedirect(w, r, "provider_unavailable")
			return
		}
		if transaction.Purpose == PurposeLink {
			err = s.store.LinkIdentityAndRotate(r.Context(), LinkRequest{UserID: session.UserID, SessionID: session.ID, Identity: identity, NewSession: credentials.Record, Now: now})
		} else {
			err = s.store.ReauthenticateAndRotate(r.Context(), ReauthRequest{UserID: session.UserID, SessionID: session.ID, Identity: identity, NewSession: credentials.Record, Now: now})
		}
		if err != nil {
			s.errorRedirect(w, r, errorCode(err))
			return
		}
		setSessionCookies(w, credentials)
	default:
		s.errorRedirect(w, r, "identity_conflict")
		return
	}
	clearCookie(w, OAuthCookieName, true)
	http.Redirect(w, r, transaction.ReturnTo, http.StatusSeeOther)
}

func (s *Service) issueSession(w http.ResponseWriter, ctx context.Context, userID, provider string, now time.Time) error {
	credentials, err := newSessionCredentials(userID, provider, now)
	if err != nil {
		return err
	}
	if err := s.store.CreateSession(ctx, credentials.Record); err != nil {
		return err
	}
	setSessionCookies(w, credentials)
	return nil
}

type sessionCredentials struct {
	Record SessionRecord
	Token  string
	CSRF   string
}

func newSessionCredentials(userID, provider string, now time.Time) (sessionCredentials, error) {
	token, err := randomToken()
	if err != nil {
		return sessionCredentials{}, err
	}
	csrf, err := randomToken()
	if err != nil {
		return sessionCredentials{}, err
	}
	return sessionCredentials{Record: SessionRecord{
		ID: newID(), UserID: userID, TokenHash: hashSecret(token), CSRFHash: hashSecret(csrf), CreatedAt: now,
		AuthenticatedAt: now, LastSeenAt: now, IdleExpiresAt: now.Add(14 * 24 * time.Hour),
		AbsoluteExpiresAt: now.Add(30 * 24 * time.Hour), AuthenticatedProvider: provider,
	}, Token: token, CSRF: csrf}, nil
}

func setSessionCookies(w http.ResponseWriter, credentials sessionCredentials) {
	http.SetCookie(w, secureCookie(SessionCookieName, credentials.Token, 30*24*time.Hour, true))
	http.SetCookie(w, secureCookie(CSRFCookieName, credentials.CSRF, 30*24*time.Hour, false))
}

func (s *Service) Authenticate(r *http.Request) (Session, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie.Value == "" {
		return Session{}, ErrSession
	}
	return s.store.ResolveSession(r.Context(), hashSecret(cookie.Value), s.now().UTC())
}

func (s *Service) ValidMutation(r *http.Request, session Session) bool {
	if r.Header.Get("Origin") != s.webOrigin {
		return false
	}
	csrf := r.Header.Get("X-CSRF-Token")
	if csrf == "" {
		csrf = r.FormValue("csrf_token")
	}
	return csrf != "" && hmac.Equal(hashSecret(csrf), session.CSRFHash)
}

func (s *Service) currentSession(w http.ResponseWriter, r *http.Request) {
	session, err := s.Authenticate(r)
	if err != nil {
		writeAuthError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	identities, err := s.store.ListIdentities(r.Context(), session.UserID)
	if err != nil {
		writeAuthError(w, http.StatusServiceUnavailable, "session_unavailable")
		return
	}
	writeAuthJSON(w, http.StatusOK, map[string]any{"user_id": session.UserID, "identities": identities, "recent_auth": s.now().UTC().Sub(session.AuthenticatedAt) <= 10*time.Minute})
}

func (s *Service) logout(w http.ResponseWriter, r *http.Request) {
	session, err := s.Authenticate(r)
	if err != nil {
		writeAuthError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !s.ValidMutation(r, session) {
		writeAuthError(w, http.StatusForbidden, "csrf_rejected")
		return
	}
	if err := s.store.RevokeSession(r.Context(), session.ID, s.now().UTC()); err != nil {
		writeAuthError(w, http.StatusServiceUnavailable, "session_unavailable")
		return
	}
	clearSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) logoutAll(w http.ResponseWriter, r *http.Request) {
	session, err := s.Authenticate(r)
	if err != nil {
		writeAuthError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !s.ValidMutation(r, session) {
		writeAuthError(w, http.StatusForbidden, "csrf_rejected")
		return
	}
	if err := s.store.RevokeUserSessions(r.Context(), session.UserID, s.now().UTC()); err != nil {
		writeAuthError(w, http.StatusServiceUnavailable, "session_unavailable")
		return
	}
	clearSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) unlink(w http.ResponseWriter, r *http.Request) {
	session, err := s.Authenticate(r)
	if err != nil {
		writeAuthError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !s.ValidMutation(r, session) {
		writeAuthError(w, http.StatusForbidden, "csrf_rejected")
		return
	}
	provider := r.PathValue("provider")
	if s.providers[provider] == nil {
		writeAuthError(w, http.StatusNotFound, "provider_unavailable")
		return
	}
	credentials, err := newSessionCredentials(session.UserID, session.AuthenticatedProvider, s.now().UTC())
	if err != nil {
		writeAuthError(w, http.StatusServiceUnavailable, "session_unavailable")
		return
	}
	err = s.store.UnlinkIdentityAndRotate(r.Context(), UnlinkRequest{
		UserID: session.UserID, SessionID: session.ID, Provider: provider, NewSession: credentials.Record,
		Now: s.now().UTC(), CooldownUntil: s.now().UTC().Add(30 * 24 * time.Hour),
	})
	if err != nil {
		writeAuthError(w, http.StatusConflict, errorCode(err))
		return
	}
	setSessionCookies(w, credentials)
	w.WriteHeader(http.StatusNoContent)
}

func clearSessionCookies(w http.ResponseWriter) {
	clearCookie(w, SessionCookieName, true)
	clearCookie(w, CSRFCookieName, false)
}

func errorCode(err error) string {
	switch {
	case errors.Is(err, ErrIdentityConflict):
		return "identity_conflict"
	case errors.Is(err, ErrLastIdentity):
		return "last_identity"
	case errors.Is(err, ErrRecentAuth):
		return "recent_auth_required"
	default:
		return "provider_unavailable"
	}
}

func writeAuthError(w http.ResponseWriter, status int, code string) {
	writeAuthJSON(w, status, map[string]string{"code": code, "message": "Authentication request could not be completed"})
}

func writeAuthJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (s *Service) errorRedirect(w http.ResponseWriter, r *http.Request, code string) {
	requestID, _ := randomToken()
	location := "/auth/error?" + url.Values{"code": {code}, "request_id": {requestID}}.Encode()
	http.Redirect(w, r, location, http.StatusSeeOther)
}

func secureCookie(name, value string, lifetime time.Duration, httpOnly bool) *http.Cookie {
	return &http.Cookie{Name: name, Value: value, Path: "/", MaxAge: int(lifetime.Seconds()), Secure: true, HttpOnly: httpOnly, SameSite: http.SameSiteLaxMode}
}

func clearCookie(w http.ResponseWriter, name string, httpOnly bool) {
	cookie := secureCookie(name, "", -time.Second, httpOnly)
	cookie.MaxAge = -1
	http.SetCookie(w, cookie)
}

func hashSecret(value string) []byte {
	digest := sha256.Sum256([]byte(value))
	return digest[:]
}

func newID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return ""
	}
	value[6] = value[6]&0x0f | 0x40
	value[8] = value[8]&0x3f | 0x80
	encoded := hex.EncodeToString(value)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32]
}

func safeReturnPath(value string) (string, bool) {
	if value == "" {
		return "/organizations", true
	}
	if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.Contains(value, "\\") || strings.Contains(strings.ToLower(value), "%2f") || strings.Contains(strings.ToLower(value), "%5c") {
		return "", false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.User != nil || parsed.Fragment != "" || strings.Contains(parsed.Path, "..") {
		return "", false
	}
	allowed := []string{"/setup", "/organizations", "/account", "/invite/", "/o/"}
	for _, prefix := range allowed {
		if value == prefix || strings.HasPrefix(value, prefix) {
			return value, true
		}
	}
	return "", false
}

func isLocalhost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
