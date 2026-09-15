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
