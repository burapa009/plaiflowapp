package httpapi

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"plaiflow/api/internal/auth"
	"plaiflow/api/internal/inbound"
	"plaiflow/api/internal/postgres"
	"plaiflow/api/internal/tenant"
	"plaiflow/api/internal/worker"
)

type fakeStore struct{ inserted []inbound.Event }

func (f *fakeStore) InsertEvents(_ context.Context, events []inbound.Event) error {
	f.inserted = append(f.inserted, events...)
	return nil
}
func (f *fakeStore) Ready(context.Context) error { return nil }
func (f *fakeStore) Snapshot(context.Context) (inbound.Snapshot, error) {
	return inbound.Snapshot{}, nil
}

func TestWebhookBoundary(t *testing.T) {
	store := &fakeStore{}
	handler := New(Config{LineSecret: "secret", LineChannel: "channel", DashboardTokens: []string{"dashboard-token"}}, store)
	body := []byte(`{"events":[{"webhookEventId":"evt-1","type":"message","timestamp":1700000000000}]}`)

	request := httptest.NewRequest(http.MethodPost, "/webhooks/line", bytes.NewReader(body))
	request.Header.Set("x-line-signature", sign(body, "wrong"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || len(store.inserted) != 0 {
		t.Fatalf("forged request: status=%d inserted=%d", response.Code, len(store.inserted))
	}

	request = httptest.NewRequest(http.MethodPost, "/webhooks/line", bytes.NewReader(body))
	request.Header.Set("x-line-signature", sign(body, "secret"))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || len(store.inserted) != 1 {
		t.Fatalf("valid request: status=%d inserted=%d", response.Code, len(store.inserted))
	}
}

func TestDashboardPropagatesRequestIDOnlyForAuthorizedCalls(t *testing.T) {
	handler := New(Config{LineSecret: "secret", LineChannel: "channel", DashboardTokens: []string{"dashboard-token"}}, &fakeStore{})

	request := httptest.NewRequest(http.MethodGet, "/v1/dashboard", nil)
	request.Header.Set("Authorization", "Bearer dashboard-token")
	request.Header.Set("X-Request-ID", "next-request-123")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("X-Request-ID") != "next-request-123" {
		t.Fatalf("authorized status=%d request_id=%q", response.Code, response.Header().Get("X-Request-ID"))
	}

	request = httptest.NewRequest(http.MethodGet, "/v1/dashboard", nil)
	request.Header.Set("Authorization", "Bearer wrong")
	request.Header.Set("X-Request-ID", "attacker-request")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || response.Header().Get("X-Request-ID") == "attacker-request" {
		t.Fatalf("unauthorized status=%d request_id=%q", response.Code, response.Header().Get("X-Request-ID"))
	}
}

func TestWebhookToPostgresToWorker(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Exec(ctx, "TRUNCATE inbound_events, worker_heartbeats"); err != nil {
		t.Fatal(err)
	}
	connection.Close(ctx)
	store, err := postgres.New(ctx, databaseURL, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	handler := New(Config{LineSecret: "secret", LineChannel: "test", DashboardTokens: []string{"dashboard-token"}}, store)
	body := []byte(fmt.Sprintf(`{"events":[{"webhookEventId":"e2e-1","type":"message","timestamp":%d}]}`, time.Now().UnixMilli()))
	request := httptest.NewRequest(http.MethodPost, "/webhooks/line", bytes.NewReader(body))
	request.Header.Set("x-line-signature", sign(body, "secret"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("webhook status=%d", response.Code)
	}
	if err := worker.RunOnce(ctx, store, nil, worker.RecordOnly); err != nil {
		t.Fatal(err)
	}
	request = httptest.NewRequest(http.MethodGet, "/v1/dashboard", nil)
	request.Header.Set("Authorization", "Bearer dashboard-token")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var snapshot inbound.Snapshot
	if err := json.NewDecoder(response.Body).Decode(&snapshot); err != nil || snapshot.Counts.Processed != 1 {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
}

type lineProviderDouble struct{ attempt auth.Attempt }

func (*lineProviderDouble) Name() string   { return "line" }
func (*lineProviderDouble) Issuer() string { return "https://access.line.me" }
func (p *lineProviderDouble) AuthorizationURL(attempt auth.Attempt) (string, error) {
	p.attempt = attempt
	return "https://line.example/authorize", nil
}
func (*lineProviderDouble) Exchange(context.Context, string, string, string) (auth.ExternalIdentity, error) {
	return auth.ExternalIdentity{Provider: "line", Issuer: "https://access.line.me", Subject: "line-user", DisplayName: "ผู้ใช้ทดสอบ"}, nil
}

func TestLineLoginCreatesOrganizationAndResolvesMembership(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Exec(ctx, "TRUNCATE users, organizations CASCADE"); err != nil {
		t.Fatal(err)
	}
	connection.Close(ctx)

	store, err := postgres.New(ctx, databaseURL, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	provider := &lineProviderDouble{}
	now := time.Now().UTC()
	authService, err := auth.NewService(auth.ServiceConfig{
		WebOrigin: "https://app.example", Providers: []auth.Provider{provider}, Now: func() time.Time { return now },
	}, store)
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Config{
		LineSecret: "secret", LineChannel: "channel", DashboardTokens: []string{"dashboard"}, Auth: authService, Tenants: store,
	}, store)

	start := httptest.NewRequest(http.MethodGet, "https://app.example/v1/auth/line/start?return_to=/organizations", nil)
	startResponse := httptest.NewRecorder()
	handler.ServeHTTP(startResponse, start)
	browserCookie := namedCookie(startResponse.Result().Cookies(), auth.OAuthCookieName)
	if startResponse.Code != http.StatusSeeOther || browserCookie == nil || provider.attempt.State == "" {
		t.Fatalf("start status=%d cookie=%v attempt=%+v", startResponse.Code, browserCookie, provider.attempt)
	}

	callback := httptest.NewRequest(http.MethodGet, "https://app.example/v1/auth/line/callback?code=test-code&state="+url.QueryEscape(provider.attempt.State), nil)
	callback.AddCookie(browserCookie)
	callbackResponse := httptest.NewRecorder()
	handler.ServeHTTP(callbackResponse, callback)
	sessionCookie := namedCookie(callbackResponse.Result().Cookies(), auth.SessionCookieName)
	csrfCookie := namedCookie(callbackResponse.Result().Cookies(), auth.CSRFCookieName)
	if callbackResponse.Code != http.StatusSeeOther || callbackResponse.Header().Get("Location") != "/organizations" || sessionCookie == nil || csrfCookie == nil {
		t.Fatalf("callback status=%d location=%q session=%v csrf=%v", callbackResponse.Code, callbackResponse.Header().Get("Location"), sessionCookie, csrfCookie)
	}

	form := url.Values{"name": {"บริษัท ปลายโฟลว์"}}
	create := httptest.NewRequest(http.MethodPost, "https://app.example/v1/organizations", strings.NewReader(form.Encode()))
	create.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	create.Header.Set("Origin", "https://app.example")
	create.Header.Set("X-CSRF-Token", csrfCookie.Value)
	create.AddCookie(sessionCookie)
	create.AddCookie(csrfCookie)
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, create)
	var organization tenant.Organization
	if err := json.NewDecoder(createResponse.Body).Decode(&organization); err != nil || createResponse.Code != http.StatusCreated || organization.Role != tenant.Owner {
		t.Fatalf("organization=%+v status=%d err=%v", organization, createResponse.Code, err)
	}

	read := httptest.NewRequest(http.MethodGet, "https://app.example/v1/o/"+organization.ID, nil)
	read.AddCookie(sessionCookie)
	readResponse := httptest.NewRecorder()
	handler.ServeHTTP(readResponse, read)
	var details struct {
		Membership tenant.Membership `json:"membership"`
	}
	if err := json.NewDecoder(readResponse.Body).Decode(&details); err != nil || readResponse.Code != http.StatusOK || details.Membership.OrganizationID != organization.ID || details.Membership.Role != tenant.Owner {
		t.Fatalf("membership=%+v status=%d err=%v", details.Membership, readResponse.Code, err)
	}
}

func namedCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
