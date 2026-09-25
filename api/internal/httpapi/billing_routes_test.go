package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"plaiflow/api/internal/billing"
	"plaiflow/api/internal/tenant"
)

type billingHTTPStore struct {
	billing.Store
	created int
	queued  int
	amount  int64
}

func (s *billingHTTPStore) CreateIntent(_ context.Context, i billing.Intent, _ time.Time) (billing.Intent, error) {
	s.created++
	s.amount = i.AmountSatang
	return i, nil
}
func (s *billingHTTPStore) AttachCharge(_ context.Context, i billing.Intent, c billing.Charge) (billing.Intent, error) {
	i.Status = "awaiting_payment"
	i.QRImageURL = c.Source.ScannableCode.Image.DownloadURI
	return i, nil
}
func (s *billingHTTPStore) QueueEvent(context.Context, billing.Event) error { s.queued++; return nil }

type billingHTTPProvider struct{ billing.Provider }

func (billingHTTPProvider) CreateCharge(_ context.Context, id string, amount int64, expiresAt time.Time) (billing.Charge, error) {
	c := billing.Charge{ID: "chrg_test_1", Status: "pending", Currency: "THB", Amount: amount, ExpiresAt: expiresAt, Metadata: map[string]string{"intent_id": id}}
	c.Source.Type = "promptpay"
	c.Source.ScannableCode.Image.DownloadURI = "https://api.omise.co/charges/chrg_test_1/documents/qr/downloads/safe"
	return c, nil
}

func billingHandler(t *testing.T, role tenant.Role, store *billingHTTPStore, now time.Time) http.Handler {
	t.Helper()
	authService, err := newTestAuth(now)
	if err != nil {
		t.Fatal(err)
	}
	service := &billing.Service{Store: store, Provider: billingHTTPProvider{}, Now: func() time.Time { return now }}
	return New(Config{Auth: authService, Tenants: &tenantStore{allowed: "org-1", role: role}, Billing: service, BillingEnabled: true,
		OmiseWebhookSecret: base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")), Now: func() time.Time { return now }}, &fakeStore{})
}

func TestBillingCheckoutRequiresOwnerAndTrustedCatalog(t *testing.T) {
	now := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	store := &billingHTTPStore{}
	handler := billingHandler(t, tenant.Owner, store, now)
	request := authenticatedForm(http.MethodPost, "https://app.example/v1/o/org-1/billing/intents", "plan=Starter&interval=six_months&amount_satang=1&csrf_token=csrf")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || store.created != 1 || store.amount != 85500 || !strings.Contains(response.Body.String(), "qr_image_url") {
		t.Fatalf("status=%d created=%d amount=%d body=%s", response.Code, store.created, store.amount, response.Body.String())
	}
	for _, tc := range []struct {
		role       tenant.Role
		path, body string
		want       int
	}{
		{tenant.Member, "/v1/o/org-1/billing/intents", "plan=Starter&interval=monthly&csrf_token=csrf", http.StatusForbidden},
		{tenant.Owner, "/v1/o/org-2/billing/intents", "plan=Starter&interval=monthly&csrf_token=csrf", http.StatusNotFound},
		{tenant.Owner, "/v1/o/org-1/billing/intents", "plan=Free&interval=monthly&csrf_token=csrf", http.StatusUnprocessableEntity},
	} {
		h := billingHandler(t, tc.role, store, now)
		r := authenticatedForm(http.MethodPost, "https://app.example"+tc.path, tc.body)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("role=%s path=%s status=%d body=%s", tc.role, tc.path, w.Code, w.Body.String())
		}
	}
	if store.created != 1 {
		t.Fatalf("unauthorized checkout reached store %d", store.created)
	}
}

func TestOmiseWebhookForgeryIsRejected(t *testing.T) {
	now := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	store := &billingHTTPStore{}
	handler := billingHandler(t, tenant.Owner, store, now)
	body := `{"id":"evnt_test_1","key":"charge.complete","data":{"id":"chrg_test_1"}}`
	request := httptest.NewRequest(http.MethodPost, "https://api.example/webhooks/omise", strings.NewReader(body))
	request.Header.Set("Omise-Signature-Timestamp", fmt.Sprint(now.Unix()))
	request.Header.Set("Omise-Signature", "forged")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || store.queued != 0 {
		t.Fatalf("forgery status=%d queued=%d", response.Code, store.queued)
	}
	mac := hmac.New(sha256.New, []byte("0123456789abcdef0123456789abcdef"))
	mac.Write([]byte(fmt.Sprint(now.Unix()) + "." + body))
	request = httptest.NewRequest(http.MethodPost, "https://api.example/webhooks/omise", strings.NewReader(body))
	request.Header.Set("Omise-Signature-Timestamp", fmt.Sprint(now.Unix()))
	request.Header.Set("Omise-Signature", hex.EncodeToString(mac.Sum(nil)))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || store.queued != 1 {
		t.Fatalf("valid status=%d queued=%d", response.Code, store.queued)
	}
}
