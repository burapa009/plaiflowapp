package httpapi

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"plaiflow/api/internal/inbound"
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

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
