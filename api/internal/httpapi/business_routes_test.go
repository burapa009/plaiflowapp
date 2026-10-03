package httpapi

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"plaiflow/api/internal/business"
	"plaiflow/api/internal/plan"
	"plaiflow/api/internal/work"
)

type businessStore struct {
	created business.Contact
	items   []business.Contact
	preview business.Preview
}

func (s *businessStore) CreateVendor(_ context.Context, _, _ string, contact business.Contact, _ time.Time) (business.Contact, error) {
	s.created = contact
	contact.ID = "vendor-1"
	return contact, nil
}
func (s *businessStore) ListVendors(context.Context, string, string, string, string, int) (business.Page, error) {
	return business.Page{Vendors: s.items}, nil
}
func (s *businessStore) ExportVendors(context.Context, string, string, int) ([]business.Contact, error) {
	return s.items, nil
}
func (s *businessStore) CreateImportPreview(_ context.Context, _, _, id string, contacts []business.Contact, now time.Time) (business.Preview, error) {
	s.preview = business.Preview{ID: id, Rows: []business.PreviewRow{{Row: 2, Status: "ready", Contact: contacts[0]}}, Ready: 1, ExpiresAt: now.Add(24 * time.Hour)}
	return s.preview, nil
}
func (s *businessStore) GetImportPreview(context.Context, string, string, string, time.Time) (business.Preview, error) {
	return s.preview, nil
}
func (*businessStore) CommitImport(context.Context, string, string, string, time.Time) (business.ImportResult, error) {
	return business.ImportResult{Created: 1}, nil
}

type fixedHTTPPlanStore plan.Key

func (s fixedHTTPPlanStore) EffectivePlan(context.Context, string) (plan.Key, error) {
	return plan.Key(s), nil
}

func TestPlansAndVendorRoutesUseServerAuthority(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	store := &businessStore{items: []business.Contact{{ID: "vendor-1", DisplayName: "Vendor", Vendor: true}}}
	handler := businessHandler(t, now, store, fixedHTTPPlanStore(plan.Starter))

	request := httptest.NewRequest(http.MethodGet, "https://app.example/v1/plans", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "six_months") || !strings.Contains(response.Body.String(), "85500") {
		t.Fatalf("catalog status=%d body=%s", response.Code, response.Body.String())
	}

	request = authenticatedForm(http.MethodPost, "https://app.example/v1/o/org-1/vendors", "display_name=Vendor&tax_id=0105552117718&csrf_token=csrf")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || store.created.TaxID != "0105552117718" {
		t.Fatalf("create status=%d contact=%+v body=%s", response.Code, store.created, response.Body.String())
	}

	denied := businessHandler(t, now, store, fixedHTTPPlanStore(plan.Free))
	request = httptest.NewRequest(http.MethodGet, "https://app.example/v1/o/org-1/vendors/export.xlsx?plan=Business", nil)
	request.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
	response = httptest.NewRecorder()
	denied.ServeHTTP(response, request)
	if response.Code != http.StatusPaymentRequired {
		t.Fatalf("tampered export status=%d body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "https://app.example/v1/o/org-1/vendors/export.csv", nil)
	request.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
	response = httptest.NewRecorder()
	denied.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.HasPrefix(response.Header().Get("Content-Type"), "text/csv") || !strings.Contains(response.Body.String(), "Vendor") {
		t.Fatalf("csv export status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestVendorImportPreviewAcceptsBoundedCSVWithoutWritingContacts(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	store := &businessStore{}
	handler := businessHandler(t, now, store, fixedHTTPPlanStore(plan.Starter))
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": {`form-data; name="file"; filename="vendors.csv"`},
		"Content-Type":        {"text/csv"},
	})
	_, _ = part.Write([]byte("display_name,tax_id\r\nVendor,0105552117718\r\n"))
	_ = writer.WriteField("csrf_token", "csrf")
	_ = writer.Close()
	request := httptest.NewRequest(http.MethodPost, "https://app.example/v1/o/org-1/vendor-imports/preview", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Origin", "https://app.example")
	request.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || store.preview.Ready != 1 || store.created.ID != "" {
		t.Fatalf("preview status=%d preview=%+v created=%+v body=%s", response.Code, store.preview, store.created, response.Body.String())
	}
}

func businessHandler(t *testing.T, now time.Time, store business.Store, planStore plan.Store) http.Handler {
	t.Helper()
	authService, err := newTestAuth(now)
	if err != nil {
		t.Fatal(err)
	}
	gate := plan.Gate{Store: planStore}
	return New(Config{LineSecret: "secret", LineChannel: "channel", DashboardTokens: []string{"dashboard"}, Auth: authService,
		Tenants: &tenantStore{allowed: "org-1"}, Business: store, PlanStore: planStore, Gate: gate, Now: func() time.Time { return now }}, &fakeStore{})
}

func authenticatedForm(method, target, body string) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://app.example")
	request.AddCookie(&http.Cookie{Name: "__Host-plaiflow-session", Value: "session"})
	return request
}

var _ work.Gate = plan.Gate{}
