package auth

import "testing"

func TestCallbackURLAllowlist(t *testing.T) {
	config := CallbackConfig{WebBaseURL: "https://staging.plaiflow.example", AllowedOrigins: []string{"https://staging.plaiflow.example"}}
	got, err := config.CallbackURL("line")
	if err != nil || got != "https://staging.plaiflow.example/api/auth/line/callback" {
		t.Fatalf("callback = %q, %v", got, err)
	}
	config.WebBaseURL = "https://evil.example"
	if _, err := config.CallbackURL("line"); err == nil {
		t.Fatal("unlisted origin accepted")
	}
}
