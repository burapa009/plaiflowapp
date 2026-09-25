package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"plaiflow/api/internal/auth"
	"plaiflow/api/internal/document"
	"plaiflow/api/internal/firm"
	"plaiflow/api/internal/tenant"
)

type firmRouteStore struct {
	action  string
	scope   string
	request firm.Grant
}

func (*firmRouteStore) ReadyFirm(context.Context) error { return nil }
func (f *firmRouteStore) RequestGrant(_ context.Context, _, firmID, clientID, id string, _ time.Time) (firm.Grant, error) {
	f.request = firm.Grant{ID: id, FirmOrganizationID: firmID, ClientOrganizationID: clientID, Status: "Requested"}
	return f.request, nil
}
func (*firmRouteStore) ListGrants(context.Context, string, string) ([]firm.Grant, error) {
	return nil, nil
}
func (*firmRouteStore) ListPortfolio(context.Context, string, string, string) (firm.PortfolioPage, error) {
	return firm.PortfolioPage{}, nil
}
func (f *firmRouteStore) TransitionGrant(_ context.Context, _, _, _, action string, _ time.Time) error {
	f.action = action
	return nil
}
func (*firmRouteStore) AssignStaff(context.Context, string, string, string, string, time.Time) error {
	return nil
}
func (*firmRouteStore) RemoveStaff(context.Context, string, string, string, string, time.Time) error {
	return nil
}
func (*firmRouteStore) ListStaff(context.Context, string, string, string) ([]firm.Assignment, error) {
	return nil, nil
}
func (f *firmRouteStore) GetFirmDocument(_ context.Context, _, _, _, _, scope string) (document.Document, error) {
	f.scope = scope
	return document.Document{}, tenant.ErrNotFound
}

func firmRouteHandler(t *testing.T, now time.Time, authenticatedAt time.Time, enabled bool, store *firmRouteStore) http.Handler {
	t.Helper()
	authService, err := auth.NewService(auth.ServiceConfig{WebOrigin: "https://app.example", Now: func() time.Time { return now }},
		&sessionStore{session: auth.Session{ID: "session-1", UserID: "user-1", CSRFHash: hash("csrf"),
			AuthenticatedAt: authenticatedAt, IdleExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(time.Hour)}})
	if err != nil {
		t.Fatal(err)
	}
	return New(Config{Auth: authService, Tenants: &tenantStore{allowed: "org-1"}, Firm: store, FirmEnabled: enabled,
		Now: func() time.Time { return now }}, &fakeStore{})
}

func TestFirmRoutesRequireFlagAndRecentClientApproval(t *testing.T) {
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	store := &firmRouteStore{}
	path := "https://app.example/v1/o/org-1/firm/grants/grant-1/approve"
	request := func() *http.Request { return authenticatedForm(http.MethodPost, path, "csrf_token=csrf") }
	response := httptest.NewRecorder()
	firmRouteHandler(t, now, now, false, store).ServeHTTP(response, request())
	if response.Code != http.StatusNotFound {
		t.Fatalf("disabled status=%d", response.Code)
	}
	response = httptest.NewRecorder()
	firmRouteHandler(t, now, now.Add(-11*time.Minute), true, store).ServeHTTP(response, request())
	if response.Code != http.StatusForbidden || store.action != "" {
		t.Fatalf("stale status=%d action=%q", response.Code, store.action)
	}
	response = httptest.NewRecorder()
	firmRouteHandler(t, now, now, true, store).ServeHTTP(response, request())
	if response.Code != http.StatusNoContent || store.action != "approve" {
		t.Fatalf("fresh status=%d action=%q", response.Code, store.action)
	}
}

func TestFirmPreviewAndDownloadUseSeparateScopes(t *testing.T) {
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	store := &firmRouteStore{}
	handler := firmRouteHandler(t, now, now, true, store)
	base := "https://app.example/v1/o/org-1/firm/clients/client-1/documents/document-1/original"
	for _, tc := range []struct{ url, scope string }{{base + "?preview=1", "read"}, {base, "original.download"}} {
		request := httptest.NewRequest(http.MethodGet, tc.url, nil)
		request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound || store.scope != tc.scope {
			t.Fatalf("url=%q status=%d scope=%q", tc.url, response.Code, store.scope)
		}
	}
}
