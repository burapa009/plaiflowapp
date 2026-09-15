package config

import "testing"

func TestLoadServerRejectsMissingSecrets(t *testing.T) {
	t.Setenv("APP_ENV", "staging")
	t.Setenv("PORT", "8080")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("LINE_CHANNEL_ID", "channel")
	t.Setenv("LINE_CHANNEL_SECRET", "")
	t.Setenv("DASHBOARD_API_TOKEN", "token")
	t.Setenv("WEB_BASE_URL", "https://staging.example")
	t.Setenv("ALLOWED_WEB_ORIGINS", "https://staging.example")
	if _, err := LoadServer(); err == nil {
		t.Fatal("missing LINE secret accepted")
	}
}

func TestLoadWorkerRequiresLineDeliveryConfiguration(t *testing.T) {
	t.Setenv("APP_ENV", "staging")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("WEB_BASE_URL", "https://staging.example")
	t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "")
	if _, err := LoadWorker(); err == nil {
		t.Fatal("missing LINE access token accepted")
	}
	t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "token")
	config, err := LoadWorker()
	if err != nil || config.WebBaseURL != "https://staging.example" || config.LineChannelAccessToken != "token" {
		t.Fatalf("config=%+v err=%v", config, err)
	}
}
