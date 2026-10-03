package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"plaiflow/api/internal/drive"
	"plaiflow/api/internal/plan"
	"plaiflow/api/internal/tenant"
)

type httpDriveStore struct {
	attempt    drive.Attempt
	connection drive.Connection
}

func (s *httpDriveStore) SaveAttempt(_ context.Context, attempt drive.Attempt) error {
	s.attempt = attempt
	return nil
}
func (s *httpDriveStore) ConsumeAttempt(context.Context, []byte, time.Time) (drive.Attempt, error) {
	return s.attempt, nil
}
func (s *httpDriveStore) SaveConnection(_ context.Context, connection drive.Connection) error {
	s.connection = connection
	return nil
}
func (s *httpDriveStore) GetConnection(context.Context, string, string) (drive.Connection, error) {
	if s.connection.Status == "" {
		return drive.Connection{Status: drive.StatusNotConnected}, nil
	}
	return s.connection, nil
}
func (s *httpDriveStore) Disconnect(context.Context, string, string, time.Time) (drive.Connection, error) {
	connection := s.connection
	s.connection.Status, s.connection.EncryptedRefreshToken, s.connection.TokenNonce = drive.StatusNotConnected, nil, nil
	return connection, nil
}
func (*httpDriveStore) RequireReconnect(context.Context, string, int64, time.Time) error { return nil }

type httpDriveProvider struct{}

func (*httpDriveProvider) AuthorizationURL(attempt drive.OAuthAttempt) string {
	return "https://accounts.example/authorize?" + url.Values{"state": {attempt.State}}.Encode()
}
func (*httpDriveProvider) Exchange(context.Context, string, string) (drive.Credential, error) {
	return drive.Credential{AccessToken: "access", RefreshToken: "refresh", Subject: "subject", Email: "owner@example.com"}, nil
}
func (*httpDriveProvider) CreateFolder(context.Context, string, string) (string, error) {
	return "folder", nil
}
func (*httpDriveProvider) Refresh(context.Context, string) (string, error) { return "access", nil }
func (*httpDriveProvider) Revoke(context.Context, string) error            { return nil }

func TestOwnerConnectsAndDisconnectsDriveThroughAuthenticatedBoundary(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	store := &httpDriveStore{}
	service, err := drive.New(drive.Config{Provider: &httpDriveProvider{}, Store: store, EncryptionKey: make([]byte, 32), Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	authService, _ := newTestAuth(now)
	planStore := fixedHTTPPlanStore(plan.Business)
	handler := New(Config{LineSecret: "secret", LineChannel: "channel", DashboardTokens: []string{"dashboard"}, Auth: authService,
		Tenants: &tenantStore{allowed: "org-1"}, Drive: service, PlanStore: planStore, Gate: plan.Gate{Store: planStore}, Now: func() time.Time { return now }}, &fakeStore{})

	connect := authenticatedForm(http.MethodPost, "https://app.example/v1/o/org-1/drive/connect", "csrf_token=csrf")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, connect)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("connect status=%d body=%s", response.Code, response.Body.String())
	}
	location, _ := url.Parse(response.Header().Get("Location"))
	state := location.Query().Get("state")
	var browserCookie *http.Cookie
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == DriveCookieName {
			browserCookie = cookie
		}
	}
	if state == "" || browserCookie == nil {
		t.Fatalf("location=%s cookies=%v", location, response.Result().Cookies())
	}

	callback := httptest.NewRequest(http.MethodGet, "https://app.example/v1/drive/callback?state="+url.QueryEscape(state)+"&code=ok", nil)
	callback.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
	callback.AddCookie(browserCookie)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, callback)
	if response.Code != http.StatusSeeOther || !strings.Contains(response.Header().Get("Location"), "/connections") {
		t.Fatalf("callback status=%d location=%s body=%s", response.Code, response.Header().Get("Location"), response.Body.String())
	}
	status := httptest.NewRequest(http.MethodGet, "https://app.example/v1/o/org-1/drive", nil)
	status.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, status)
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "refresh") || strings.Contains(response.Body.String(), "nonce") {
		t.Fatalf("unsafe status=%d body=%s", response.Code, response.Body.String())
	}

	disconnect := authenticatedForm(http.MethodPost, "https://app.example/v1/o/org-1/drive/disconnect", "csrf_token=csrf")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, disconnect)
	if response.Code != http.StatusNoContent || store.connection.Status != drive.StatusNotConnected {
		t.Fatalf("disconnect status=%d connection=%+v", response.Code, store.connection)
	}
}

func TestClientAdminCanStartDriveConnection(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	store := &httpDriveStore{}
	service, _ := drive.New(drive.Config{Provider: &httpDriveProvider{}, Store: store, EncryptionKey: make([]byte, 32), Now: func() time.Time { return now }})
	authService, _ := newTestAuth(now)
	planStore := fixedHTTPPlanStore(plan.Business)
	handler := New(Config{LineSecret: "secret", LineChannel: "channel", DashboardTokens: []string{"dashboard"}, Auth: authService,
		Tenants: &tenantStore{allowed: "org-1", role: tenant.Admin}, Drive: service, PlanStore: planStore, Gate: plan.Gate{Store: planStore}, Now: func() time.Time { return now }}, &fakeStore{})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, authenticatedForm(http.MethodPost, "https://app.example/v1/o/org-1/drive/connect", "csrf_token=csrf"))
	if response.Code != http.StatusSeeOther || store.attempt.StateHash == nil {
		t.Fatalf("status=%d attempt=%+v", response.Code, store.attempt)
	}
}
