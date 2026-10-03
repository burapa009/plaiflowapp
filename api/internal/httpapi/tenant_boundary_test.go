package httpapi

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"plaiflow/api/internal/auth"
	"plaiflow/api/internal/tenant"
)

type boundarySessionStore struct {
	sessionStore
	revoked bool
}

func (s *boundarySessionStore) ResolveSession(_ context.Context, _ []byte, now time.Time) (auth.Session, error) {
	if s.revoked || !now.Before(s.session.IdleExpiresAt) || !now.Before(s.session.AbsoluteExpiresAt) {
		return auth.Session{}, auth.ErrSession
	}
	return s.session, nil
}

func (s *boundarySessionStore) RevokeSession(context.Context, string, time.Time) error {
	s.revoked = true
	return nil
}

func TestTenantRequestsUseCurrentSessionAndRole(t *testing.T) {
	now := time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)
	sessions := &boundarySessionStore{sessionStore: sessionStore{session: auth.Session{ID: "session", UserID: "user",
		CSRFHash: hash("csrf"), AuthenticatedAt: now, IdleExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(2 * time.Hour)}}}
	authService, err := auth.NewService(auth.ServiceConfig{WebOrigin: "https://app.example", Now: func() time.Time { return now }}, sessions)
	if err != nil {
		t.Fatal(err)
	}
	tenants := &tenantStore{allowed: "org-1", role: tenant.Owner}
	handler := New(Config{Auth: authService, Tenants: tenants, Now: func() time.Time { return now }}, &fakeStore{})
	request := func(method, path, csrf, origin string) int {
		t.Helper()
		var body bytes.Buffer
		if method == http.MethodPost {
			writer := multipart.NewWriter(&body)
			for key, value := range map[string]string{
				"business_type": "บริษัทมหาชนจำกัด", "vat_status": "registered", "branch_type": "head",
				"name_th": "Boundary", "phone": "0635167015",
			} {
				if err := writer.WriteField(key, value); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest(method, "https://app.example"+path, &body)
			r.Header.Set("Content-Type", writer.FormDataContentType())
			r.Header.Set("Origin", origin)
			r.Header.Set("X-CSRF-Token", csrf)
			r.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, r)
			return response.Code
		}
		r := httptest.NewRequest(method, "https://app.example"+path, nil)
		r.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, r)
		return response.Code
	}
	if got := request(http.MethodGet, "/v1/o/org-1", "", ""); got != http.StatusOK {
		t.Fatalf("owner read=%d", got)
	}
	if got := request(http.MethodGet, "/v1/o/org-2", "", ""); got != http.StatusNotFound {
		t.Fatalf("cross-tenant read=%d", got)
	}
	for _, role := range []tenant.Role{tenant.Owner, tenant.Admin, tenant.Member} {
		tenants.role = role
		want := http.StatusNoContent
		if role == tenant.Member {
			want = http.StatusForbidden
		}
		if got := request(http.MethodPost, "/v1/o/org-1/business", "csrf", "https://app.example"); got != want {
			t.Fatalf("role=%s update=%d want=%d", role, got, want)
		}
	}
	if got := request(http.MethodPost, "/v1/o/org-2/business", "csrf", "https://app.example"); got != http.StatusNotFound {
		t.Fatalf("cross-tenant write=%d", got)
	}
	if got := request(http.MethodPost, "/v1/o/org-1/business", "wrong", "https://app.example"); got != http.StatusForbidden {
		t.Fatalf("invalid CSRF=%d", got)
	}
	if got := request(http.MethodPost, "/v1/o/org-1/business", "csrf", "https://evil.example"); got != http.StatusForbidden {
		t.Fatalf("invalid origin=%d", got)
	}
	tenants.allowed = "" // invited-only and revoked members have no active membership
	if got := request(http.MethodGet, "/v1/o/org-1", "", ""); got != http.StatusNotFound {
		t.Fatalf("inactive membership=%d", got)
	}
	tenants.allowed = "org-1"
	logout := httptest.NewRequest(http.MethodPost, "https://app.example/v1/logout", nil)
	logout.Header.Set("Origin", "https://app.example")
	logout.Header.Set("X-CSRF-Token", "csrf")
	logout.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, logout)
	if response.Code != http.StatusNoContent || !sessions.revoked {
		t.Fatalf("logout=%d revoked=%v", response.Code, sessions.revoked)
	}
	if got := request(http.MethodGet, "/v1/o/org-1", "", ""); got != http.StatusUnauthorized {
		t.Fatalf("logged-out read=%d", got)
	}
	sessions.revoked = false
	now = now.Add(2 * time.Hour)
	if got := request(http.MethodGet, "/v1/o/org-1", "", ""); got != http.StatusUnauthorized {
		t.Fatalf("expired read=%d", got)
	}
}
