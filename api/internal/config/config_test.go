package config

import (
	"encoding/base64"
	"strings"
	"testing"
)

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

func TestLoadServerRequiresCompleteSeparateDriveConfiguration(t *testing.T) {
	setValidServerEnvironment(t)
	t.Setenv("GOOGLE_DRIVE_CLIENT_ID", "drive-client")
	if _, err := LoadServer(); err == nil {
		t.Fatal("partial Drive configuration accepted")
	}
	t.Setenv("GOOGLE_DRIVE_CLIENT_SECRET", "drive-secret")
	t.Setenv("GOOGLE_DRIVE_TOKEN_KEY", "MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDA=")
	t.Setenv("GOOGLE_DRIVE_CLIENT_ID", "google-login")
	if _, err := LoadServer(); err == nil {
		t.Fatal("Google Login client reused for Drive")
	}
	t.Setenv("GOOGLE_DRIVE_CLIENT_ID", "drive-client")
	config, err := LoadServer()
	if err != nil || config.GoogleDriveClientID != "drive-client" || len(config.GoogleDriveTokenKey) != 32 {
		t.Fatalf("config=%+v err=%v", config, err)
	}
}

func TestLoadServerRejectsNonNumericLineLoginChannelID(t *testing.T) {
	setValidServerEnvironment(t)
	t.Setenv("LINE_LOGIN_CHANNEL_ID", "staging-not-configured")
	if _, err := LoadServer(); err == nil {
		t.Fatal("non-numeric LINE Login channel ID accepted")
	}
}

func TestLoadServerRequiresCompleteDocumentStorageAndScanner(t *testing.T) {
	setValidServerEnvironment(t)
	t.Setenv("DOCUMENT_BUCKET", "staging-documents")
	if _, err := LoadServer(); err == nil {
		t.Fatal("partial document configuration accepted")
	}
	t.Setenv("DOCUMENT_REGION", "sin1")
	t.Setenv("DOCUMENT_ENDPOINT", "https://storage.example")
	t.Setenv("DOCUMENT_ACCESS_KEY_ID", "access")
	t.Setenv("DOCUMENT_SECRET_ACCESS_KEY", "secret")
	t.Setenv("DOCUMENT_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))))
	t.Setenv("CLAMD_ADDR", "clamd.internal:3310")
	config, err := LoadServer()
	if err != nil || config.DocumentBucket != "staging-documents" || len(config.DocumentEncryptionKey) != 32 {
		t.Fatalf("config=%+v err=%v", config, err)
	}
}

func setValidServerEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "test")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("LINE_CHANNEL_ID", "line-channel")
	t.Setenv("LINE_CHANNEL_SECRET", "line-secret")
	t.Setenv("LINE_LOGIN_CHANNEL_ID", "1234567890")
	t.Setenv("LINE_LOGIN_CHANNEL_SECRET", "line-login-secret")
	t.Setenv("GOOGLE_LOGIN_CLIENT_ID", "google-login")
	t.Setenv("GOOGLE_LOGIN_CLIENT_SECRET", "google-login-secret")
	t.Setenv("DASHBOARD_API_TOKEN", "dashboard")
	t.Setenv("WEB_BASE_URL", "https://app.example")
	t.Setenv("ALLOWED_WEB_ORIGINS", "https://app.example")
	t.Setenv("GOOGLE_DRIVE_CLIENT_ID", "")
	t.Setenv("GOOGLE_DRIVE_CLIENT_SECRET", "")
	t.Setenv("GOOGLE_DRIVE_TOKEN_KEY", "")
}
