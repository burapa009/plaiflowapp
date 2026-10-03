package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"plaiflow/api/internal/job"
	"plaiflow/api/internal/secretary"
)

type secretaryStoreDouble struct {
	user, organization string
	briefing           secretary.Briefing
}

func (s *secretaryStoreDouble) Open(_ context.Context, user, organization string, _ time.Time) (secretary.Briefing, error) {
	s.user, s.organization = user, organization
	return s.briefing, nil
}
func (s *secretaryStoreDouble) Refresh(_ context.Context, user, organization string, _ time.Time) (secretary.Briefing, error) {
	return s.Open(context.Background(), user, organization, time.Time{})
}
func (*secretaryStoreDouble) Generate(context.Context, job.LeaseCommand) error { return nil }
func (*secretaryStoreDouble) Intent(context.Context, string, string, string, time.Time) (secretary.IntentAnswer, error) {
	return secretary.IntentAnswer{}, nil
}
func (*secretaryStoreDouble) GetTaskSource(context.Context, string, string, string) (secretary.TaskSource, error) {
	return secretary.TaskSource{}, nil
}
func (*secretaryStoreDouble) SetTaskSource(_ context.Context, command secretary.SourceCommand) (secretary.TaskSource, error) {
	return command.TaskSource, nil
}
func (*secretaryStoreDouble) GetPreferences(context.Context, string, string) (secretary.Preferences, error) {
	return secretary.Preferences{}, nil
}
func (*secretaryStoreDouble) SetPreferences(_ context.Context, _, _ string, p secretary.Preferences, _ time.Time) (secretary.Preferences, error) {
	return p, nil
}
func (*secretaryStoreDouble) ListRoutineTemplates(context.Context, string, string) ([]secretary.RoutineTemplate, error) {
	return nil, nil
}
func (*secretaryStoreDouble) SaveRoutineTemplate(_ context.Context, command secretary.TemplateCommand) (secretary.RoutineTemplate, error) {
	return command.RoutineTemplate, nil
}
func (*secretaryStoreDouble) ListRoutineSuggestions(context.Context, string, string, time.Time) ([]secretary.RoutineSuggestion, error) {
	return nil, nil
}
func (*secretaryStoreDouble) ResolveRoutineSuggestion(context.Context, string, string, string, string, time.Time) (secretary.RoutineSuggestion, error) {
	return secretary.RoutineSuggestion{}, nil
}

func TestSecretaryBriefingIsScopedToAuthenticatedMembershipAndFlag(t *testing.T) {
	now := time.Date(2026, 9, 28, 3, 0, 0, 0, time.UTC)
	authService, err := newTestAuth(now)
	if err != nil {
		t.Fatal(err)
	}
	store := &secretaryStoreDouble{briefing: secretary.Briefing{Status: "Ready", LocalDay: "2026-09-28", Items: []secretary.Item{{SourceID: "task-1", Title: "Call supplier"}}}}
	build := func(enabled bool) http.Handler {
		return New(Config{Auth: authService, Tenants: &tenantStore{allowed: "org-1"}, Secretary: store, SecretaryEnabled: enabled,
			SecretaryPilotOrganizations: map[string]bool{"org-1": true}, Now: func() time.Time { return now }}, &fakeStore{})
	}
	request := func(path string) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "https://app.example"+path, nil)
		r.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
		return r
	}
	for _, tc := range []struct {
		enabled  bool
		path     string
		status   int
		fragment string
	}{
		{true, "/v1/o/org-1/secretary-briefing", http.StatusOK, "Call supplier"},
		{true, "/v1/o/org-2/secretary-briefing", http.StatusNotFound, "not_found"},
		{false, "/v1/o/org-1/secretary-briefing", http.StatusNotFound, "404"},
	} {
		response := httptest.NewRecorder()
		build(tc.enabled).ServeHTTP(response, request(tc.path))
		if response.Code != tc.status || !strings.Contains(response.Body.String(), tc.fragment) {
			t.Fatalf("enabled=%v path=%s status=%d body=%s", tc.enabled, tc.path, response.Code, response.Body.String())
		}
	}
	if store.user != "user-1" || store.organization != "org-1" {
		t.Fatalf("scope=%s/%s", store.user, store.organization)
	}
	withoutPilot := New(Config{Auth: authService, Tenants: &tenantStore{allowed: "org-1"}, Secretary: store,
		SecretaryEnabled: true, Now: func() time.Time { return now }}, &fakeStore{})
	response := httptest.NewRecorder()
	withoutPilot.ServeHTTP(response, request("/v1/o/org-1/secretary-briefing"))
	if response.Code != http.StatusNotFound {
		t.Fatalf("no pilot organization: %d", response.Code)
	}
}

func TestSecretaryGenerationRequiresWriteScope(t *testing.T) {
	now := time.Date(2026, 9, 28, 3, 0, 0, 0, time.UTC)
	auth, err := job.NewWorkerAuth([]byte("01234567890123456789012345678901"), "staging", []string{"jobs:read", "jobs:generate"})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Config{SecretaryEnabled: true, Secretary: &secretaryStoreDouble{}, Jobs: &jobStoreDouble{}, JobWorkerAuth: auth,
		Now: func() time.Time { return now }}, &fakeStore{})
	for _, tc := range []struct {
		scope string
		want  int
	}{{"jobs:read", http.StatusUnauthorized}, {"jobs:generate", http.StatusOK}} {
		token, err := auth.Sign("worker-1", []string{tc.scope}, now, time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodPost, "https://api.example/internal/v1/jobs/job-1/secretary-generate", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != tc.want {
			t.Fatalf("scope %s: status %d, body %s", tc.scope, response.Code, response.Body.String())
		}
	}
}
