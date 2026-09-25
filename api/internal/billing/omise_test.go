package billing

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
)

func TestWebhookRequiresValidSignatureAndRecentTimestamp(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	secret := []byte("0123456789abcdef0123456789abcdef")
	body := []byte(`{"id":"ev_1","key":"charge.complete"}`)
	timestamp := fmt.Sprint(now.Unix())
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(timestamp + "."))
	mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))
	encoded := base64.StdEncoding.EncodeToString(secret)
	if err := VerifyWebhook(body, timestamp, "bad,"+signature, encoded, now); err != nil {
		t.Fatal(err)
	}
	if err := VerifyWebhook([]byte(`{"id":"ev_2"}`), timestamp, signature, encoded, now); err == nil {
		t.Fatal("forged body accepted")
	}
	if err := VerifyWebhook(body, timestamp, signature, encoded, now.Add(6*time.Minute)); err == nil {
		t.Fatal("stale event accepted")
	}
}

func TestCreateAndRetrievePromptPayCharge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _, ok := r.BasicAuth()
		if !ok || user != "skey_test_safe" {
			t.Error("missing secret auth")
		}
		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.Form.Get("amount") != "15000" || r.Form.Get("source[type]") != "promptpay" || r.Form.Get("metadata[intent_id]") != "intent-1" {
				t.Errorf("untrusted charge form: %v", r.Form)
			}
		} else if r.URL.Path != "/charges/chrg_test_1" {
			t.Errorf("path=%s", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":"chrg_test_1","status":"successful","paid":true,"currency":"THB","amount":15000,"livemode":false,"source":{"type":"promptpay"},"metadata":{"intent_id":"intent-1"}}`)
	}))
	defer server.Close()
	provider := Omise{SecretKey: "skey_test_safe", Endpoint: server.URL, Client: server.Client()}
	charge, err := provider.CreateCharge(context.Background(), "intent-1", 15000, time.Now().Add(30*time.Minute))
	if err != nil || !charge.Successful("intent-1", 15000, false) {
		t.Fatalf("charge=%+v err=%v", charge, err)
	}
	charge, err = provider.RetrieveCharge(context.Background(), "chrg_test_1")
	if err != nil || !charge.Successful("intent-1", 15000, false) {
		t.Fatalf("retrieved=%+v err=%v", charge, err)
	}
	if charge.Successful("intent-1", 15100, false) || charge.Successful("other", 15000, false) || charge.Successful("intent-1", 15000, true) {
		t.Fatal("tampered charge accepted")
	}
	if _, err := provider.RetrieveCharge(context.Background(), strings.Repeat("x", 30)); err == nil {
		t.Fatal("invalid ID accepted")
	}
}
