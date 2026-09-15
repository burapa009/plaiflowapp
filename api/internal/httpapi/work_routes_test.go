package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"plaiflow/api/internal/auth"
	"plaiflow/api/internal/tenant"
	"plaiflow/api/internal/work"
)

type workStore struct {
	created    work.CreateTask
	updated    work.UpdateTask
	dashboard  work.Dashboard
	exportRows []work.ExportRow
}

func (s *workStore) CreateTask(_ context.Context, command work.CreateTask) (work.Task, error) {
	s.created = command
	return work.Task{ID: command.ID, OrganizationID: command.OrganizationID, Title: command.Title, Status: work.Open, Priority: command.Priority}, nil
}
func (*workStore) ListTasks(context.Context, string, string, work.TaskFilter) (work.TaskPage, error) {
	return work.TaskPage{}, nil
}
func (*workStore) GetTask(context.Context, string, string, string) (work.Task, error) {
	return work.Task{}, nil
}
func (s *workStore) UpdateTask(_ context.Context, command work.UpdateTask) (work.Task, error) {
	s.updated = command
	return work.Task{ID: command.ID, Title: command.Title, Status: command.Status, Priority: command.Priority}, nil
}
func (*workStore) SetWatcher(context.Context, string, string, string, string, bool, time.Time) error {
	return nil
}
func (*workStore) ListNotifications(context.Context, string, string, string, int) (work.NotificationPage, error) {
	return work.NotificationPage{}, nil
}
func (*workStore) MarkNotificationRead(context.Context, string, string, string, time.Time) error {
	return nil
}
func (*workStore) MarkAllNotificationsRead(context.Context, string, string, time.Time) error {
	return nil
}
func (*workStore) ListPreferences(context.Context, string, string) ([]work.Preference, error) {
	return nil, nil
}
func (*workStore) SetPreference(context.Context, work.Preference) error { return nil }
func (s *workStore) Dashboard(context.Context, string, string) (work.Dashboard, error) {
	return s.dashboard, nil
}
func (s *workStore) CountExport(context.Context, string, string, work.TaskFilter) (int64, error) {
	return int64(len(s.exportRows)), nil
}
func (s *workStore) ExportRows(_ context.Context, _, _ string, _ work.TaskFilter, emit func(work.ExportRow) error) error {
	for _, row := range s.exportRows {
		if err := emit(row); err != nil {
			return err
		}
	}
	return nil
}
func (*workStore) QueueExport(context.Context, work.ExportRequest, int64) (work.ExportJob, error) {
	return work.ExportJob{}, nil
}
func (*workStore) ListExports(context.Context, string, string, string, int) (work.ExportPage, error) {
	return work.ExportPage{}, nil
}
func (*workStore) AuditExport(context.Context, work.ExportRequest, string, int64, int64, string) error {
	return nil
}

type deniedGate struct{}

func (deniedGate) Check(context.Context, string, work.Capability, int64) (work.EntitlementDecision, error) {
	return work.EntitlementDecision{Allowed: false, Reason: "limit_reached"}, nil
}
func (deniedGate) Record(context.Context, string, work.Capability, int64, string) error { return nil }

func TestTaskCreateRequiresServerEntitlementAfterMembership(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	store := &workStore{}
	handler := workHandler(t, now, store, work.UnlimitedGate{})
	form := url.Values{"title": {"  First task  "}, "assignee_user_id": {"user-2"}, "csrf_token": {"csrf"}}
	request := httptest.NewRequest(http.MethodPost, "https://app.example/v1/o/org-1/tasks", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://app.example")
	request.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || store.created.Title != "First task" || store.created.ActorUserID != "user-1" {
		t.Fatalf("status=%d command=%+v body=%s", response.Code, store.created, response.Body.String())
	}

	denied := workHandler(t, now, &workStore{}, deniedGate{})
	request = httptest.NewRequest(http.MethodPost, "https://app.example/v1/o/org-1/tasks", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://app.example")
	request.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
	response = httptest.NewRecorder()
	denied.ServeHTTP(response, request)
	if response.Code != http.StatusPaymentRequired || !strings.Contains(response.Body.String(), "usage_limit_reached") {
		t.Fatalf("denied status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestTaskUpdateDistinguishesExplicitClearsFromOmittedFields(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	store := &workStore{}
	handler := workHandler(t, now, store, work.UnlimitedGate{})
	request := httptest.NewRequest(http.MethodPost, "https://app.example/v1/o/org-1/tasks/task-1", strings.NewReader("assignee_user_id=&csrf_token=csrf"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://app.example")
	request.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !store.updated.AssigneeSet || store.updated.TitleSet || store.updated.DueOnSet {
		t.Fatalf("status=%d command=%+v body=%s", response.Code, store.updated, response.Body.String())
	}
}

func TestDashboardAssistantAndCSVUseAuthorizedOrganizationData(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	store := &workStore{dashboard: work.Dashboard{Scope: "org-1", Role: tenant.Owner, MyFocus: []work.Task{{ID: "task-1"}}, OpenCount: 1, OverdueCount: 1, UnreadCount: 2}, exportRows: []work.ExportRow{{
		OrganizationID: "org-1", OrganizationName: "=bad", TaskID: "task-1", Title: "+formula", Status: work.Open, Priority: work.Normal,
		CreatedAt: now, UpdatedAt: now,
	}}}
	handler := workHandler(t, now, store, work.UnlimitedGate{})
	for path, fragment := range map[string]string{
		"/v1/o/org-1/work-dashboard":    `"open_count":1`,
		"/v1/o/org-1/assistant-summary": `"bullets"`,
	} {
		request := httptest.NewRequest(http.MethodGet, "https://app.example"+path, nil)
		request.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), fragment) {
			t.Fatalf("path=%s status=%d body=%s", path, response.Code, response.Body.String())
		}
	}

	request := httptest.NewRequest(http.MethodPost, "https://app.example/v1/o/org-1/task-exports", strings.NewReader("csrf_token=csrf"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://app.example")
	request.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "'=bad") || !strings.Contains(response.Body.String(), "'+formula") {
		t.Fatalf("csv status=%d body=%q", response.Code, response.Body.String())
	}
}

func workHandler(t *testing.T, now time.Time, workStore work.Store, gate work.Gate) http.Handler {
	t.Helper()
	authService, err := newTestAuth(now)
	if err != nil {
		t.Fatal(err)
	}
	tenants := &tenantStore{allowed: "org-1"}
	return New(Config{LineSecret: "secret", LineChannel: "channel", DashboardTokens: []string{"dashboard"}, Auth: authService, Tenants: tenants, Work: workStore, Gate: gate, Now: func() time.Time { return now }}, &fakeStore{})
}

func newTestAuth(now time.Time) (*auth.Service, error) {
	return auth.NewService(auth.ServiceConfig{WebOrigin: "https://app.example", Now: func() time.Time { return now }}, &sessionStore{session: auth.Session{
		ID: "session-1", UserID: "user-1", CSRFHash: hash("csrf"), AuthenticatedAt: now, IdleExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(time.Hour),
	}})
}
