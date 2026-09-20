package line

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContentClientUsesServerTokenAndRejectsInvalidID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" || r.URL.Path != "/v2/bot/message/msg-1/content" {
			t.Fatalf("request=%s auth=%q", r.URL, r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte("%PDF-1.7"))
	}))
	defer server.Close()
	client, err := NewContentClient(server.Client(), server.URL, "token")
	if err != nil {
		t.Fatal(err)
	}
	body, err := client.Download(context.Background(), "msg-1")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(body)
	body.Close()
	if string(data) != "%PDF-1.7" {
		t.Fatalf("body=%q", data)
	}
	if _, err := client.Download(context.Background(), "../bad"); err == nil {
		t.Fatal("invalid content id accepted")
	}
}

func TestContentClientDoesNotRetryExpiredContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "expired", http.StatusNotFound)
	}))
	defer server.Close()
	client, err := NewContentClient(server.Client(), server.URL, "token")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Download(context.Background(), "msg-1")
	if !errors.Is(err, ErrContentUnavailable) {
		t.Fatalf("expired content err=%v", err)
	}
}
