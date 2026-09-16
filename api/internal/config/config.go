package config

import (
	"encoding/base64"
	"errors"
	"os"
	"strconv"
	"strings"
)

type Server struct {
	Environment         string
	Port                string
	DatabaseURL         string
	LineChannel         string
	LineSecret          string
	LineLoginChannel    string
	LineLoginSecret     string
	GoogleClientID      string
	GoogleClientSecret  string
	GoogleDriveClientID string
	GoogleDriveSecret   string
	GoogleDriveTokenKey []byte
	DashboardTokens     []string
	WebBaseURL          string
	AllowedWebOrigins   []string
	PoolMax             int32
}

type Worker struct {
	Environment            string
	DatabaseURL            string
	LineChannelAccessToken string
	WebBaseURL             string
	PoolMax                int32
}

func LoadServer() (Server, error) {
	config := Server{
		Environment: os.Getenv("APP_ENV"), Port: value("PORT", "8080"), DatabaseURL: os.Getenv("DATABASE_URL"),
		LineChannel: os.Getenv("LINE_CHANNEL_ID"), LineSecret: os.Getenv("LINE_CHANNEL_SECRET"),
		LineLoginChannel: os.Getenv("LINE_LOGIN_CHANNEL_ID"), LineLoginSecret: os.Getenv("LINE_LOGIN_CHANNEL_SECRET"),
		GoogleClientID: os.Getenv("GOOGLE_LOGIN_CLIENT_ID"), GoogleClientSecret: os.Getenv("GOOGLE_LOGIN_CLIENT_SECRET"),
		GoogleDriveClientID: os.Getenv("GOOGLE_DRIVE_CLIENT_ID"), GoogleDriveSecret: os.Getenv("GOOGLE_DRIVE_CLIENT_SECRET"),
		DashboardTokens: split(os.Getenv("DASHBOARD_API_TOKEN")), WebBaseURL: os.Getenv("WEB_BASE_URL"),
		AllowedWebOrigins: split(os.Getenv("ALLOWED_WEB_ORIGINS")), PoolMax: int32(number("API_DB_POOL_MAX", 10)),
	}
	if config.Environment == "" || config.DatabaseURL == "" || config.LineChannel == "" || config.LineSecret == "" || config.LineLoginChannel == "" || config.LineLoginSecret == "" || config.GoogleClientID == "" || config.GoogleClientSecret == "" || len(config.DashboardTokens) == 0 || config.WebBaseURL == "" || len(config.AllowedWebOrigins) == 0 {
		return Server{}, errors.New("missing required server configuration")
	}
	driveKey := os.Getenv("GOOGLE_DRIVE_TOKEN_KEY")
	driveConfigured := config.GoogleDriveClientID != "" || config.GoogleDriveSecret != "" || driveKey != ""
	if driveConfigured {
		decoded, err := base64.StdEncoding.DecodeString(driveKey)
		if config.GoogleDriveClientID == "" || config.GoogleDriveSecret == "" || err != nil || len(decoded) != 32 {
			return Server{}, errors.New("invalid Google Drive configuration")
		}
		config.GoogleDriveTokenKey = decoded
	}
	return config, nil
}

func LoadWorker() (Worker, error) {
	config := Worker{Environment: os.Getenv("APP_ENV"), DatabaseURL: os.Getenv("DATABASE_URL"),
		LineChannelAccessToken: os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"), WebBaseURL: os.Getenv("WEB_BASE_URL"),
		PoolMax: int32(number("WORKER_DB_POOL_MAX", 5))}
	if config.Environment == "" || config.DatabaseURL == "" || config.LineChannelAccessToken == "" || config.WebBaseURL == "" {
		return Worker{}, errors.New("missing required worker configuration")
	}
	return config, nil
}

func value(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func split(value string) []string {
	var values []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, item)
		}
	}
	return values
}

func number(key string, fallback int) int {
	parsed, err := strconv.Atoi(os.Getenv(key))
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}
