package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLINEWebhookRejectsUnsignedTamperedMalformedAndOversized(t *testing.T) {
	store := &fakeStore{}
	handler := New(Config{LineSecret: "secret", LineChannel: "channel"}, store)
	valid := []byte(`{"events":[{"webhookEventId":"event-1","type":"message","timestamp":1700000000000,"source":{"type":"group","groupId":"group-1","userId":"line-user"}}]}`)
	call := func(body []byte, signature string) int {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, "/webhooks/line", bytes.NewReader(body))
		request.Header.Set("X-Line-Signature", signature)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response.Code
	}
	if got := call(valid, ""); got != http.StatusUnauthorized {
		t.Fatalf("unsigned=%d", got)
	}
	if got := call(bytes.Replace(valid, []byte("line-user"), []byte("attacker!"), 1), sign(valid, "secret")); got != http.StatusUnauthorized {
		t.Fatalf("source impersonation=%d", got)
	}
	malformed := []byte(`{"events":[{"type":"message"}]}`)
	if got := call(malformed, sign(malformed, "secret")); got != http.StatusBadRequest {
		t.Fatalf("signed malformed=%d", got)
	}
	large := bytes.Repeat([]byte("x"), maxBody+1)
	if got := call(large, sign(large, "secret")); got != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized=%d", got)
	}
	if len(store.inserted) != 0 {
		t.Fatalf("invalid events stored=%d", len(store.inserted))
	}
	if got := call(valid, sign(valid, "secret")); got != http.StatusOK || len(store.inserted) != 1 {
		t.Fatalf("valid=%d stored=%d", got, len(store.inserted))
	}
}
