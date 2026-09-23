package config

import (
	"encoding/base64"
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Server struct {
	Environment           string
	Port                  string
	DatabaseURL           string
	LineChannel           string
	LineSecret            string
	LineLoginChannel      string
	LineLoginSecret       string
	GoogleClientID        string
	GoogleClientSecret    string
	GoogleDriveClientID   string
	GoogleDriveSecret     string
	GoogleDriveTokenKey   []byte
	DocumentBucket        string
	DocumentRegion        string
	DocumentEndpoint      string
	DocumentAccessKeyID   string
	DocumentSecretKey     string
	DocumentEncryptionKey []byte
	DocumentPathStyle     bool
	ClamDAddress          string
	SkipDocumentScan      bool
	JobWorkerAuthKey      []byte
	OCRWorkerAuthKey      []byte
	ExportDownloadKey     []byte
	ExportBucket          string
	ExportRegion          string
	ExportEndpoint        string
	ExportAccessKeyID     string
	ExportSecretKey       string
	ExportEncryptionKey   []byte
	ExportPathStyle       bool
	DashboardTokens       []string
	WebBaseURL            string
	AllowedWebOrigins     []string
	PoolMax               int32
}

type JobWorker struct {
	Environment, APIURL, WorkerID string
	AuthKey                       []byte
}

type Worker struct {
	Environment            string
	DatabaseURL            string
	LineChannelAccessToken string
	DocumentBucket         string
	DocumentRegion         string
	DocumentEndpoint       string
	DocumentAccessKeyID    string
	DocumentSecretKey      string
	DocumentEncryptionKey  []byte
	DocumentPathStyle      bool
	ClamDAddress           string
	SkipDocumentScan       bool
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
	if strings.Trim(config.LineLoginChannel, "0123456789") != "" {
		return Server{}, errors.New("invalid LINE Login configuration")
	}
	driveKey := os.Getenv("GOOGLE_DRIVE_TOKEN_KEY")
	driveConfigured := config.GoogleDriveClientID != "" || config.GoogleDriveSecret != "" || driveKey != ""
	if driveConfigured {
		decoded, err := base64.StdEncoding.DecodeString(driveKey)
		if config.GoogleDriveClientID == "" || config.GoogleDriveSecret == "" || config.GoogleDriveClientID == config.GoogleClientID || err != nil || len(decoded) != 32 {
			return Server{}, errors.New("invalid Google Drive configuration")
		}
		config.GoogleDriveTokenKey = decoded
	}
	config.DocumentBucket = os.Getenv("DOCUMENT_BUCKET")
	config.DocumentRegion = os.Getenv("DOCUMENT_REGION")
	config.DocumentEndpoint = os.Getenv("DOCUMENT_ENDPOINT")
	config.DocumentAccessKeyID = os.Getenv("DOCUMENT_ACCESS_KEY_ID")
	config.DocumentSecretKey = os.Getenv("DOCUMENT_SECRET_ACCESS_KEY")
	config.DocumentPathStyle = os.Getenv("DOCUMENT_PATH_STYLE") == "true"
	config.ClamDAddress = os.Getenv("CLAMD_ADDR")
	config.SkipDocumentScan = os.Getenv("SKIP_DOCUMENT_SCAN") == "true"
	if config.SkipDocumentScan && config.Environment != "staging" {
		return Server{}, errors.New("document scan bypass is restricted to staging")
	}
	documentKey := os.Getenv("DOCUMENT_ENCRYPTION_KEY")
	documentConfigured := config.DocumentBucket != "" || config.DocumentRegion != "" || config.DocumentEndpoint != "" || config.DocumentAccessKeyID != "" || config.DocumentSecretKey != "" || documentKey != "" || config.ClamDAddress != "" || config.SkipDocumentScan
	if (config.Environment == "staging" || config.Environment == "production") && !documentConfigured {
		return Server{}, errors.New("document storage and scanner are required")
	}
	if documentConfigured {
		decoded, err := base64.StdEncoding.DecodeString(documentKey)
		if config.DocumentBucket == "" || config.DocumentRegion == "" || config.DocumentEndpoint == "" || config.DocumentAccessKeyID == "" || config.DocumentSecretKey == "" || (!config.SkipDocumentScan && config.ClamDAddress == "") || err != nil || len(decoded) != 32 {
			return Server{}, errors.New("invalid document configuration")
		}
		config.DocumentEncryptionKey = decoded
	}
	jobKey := os.Getenv("JOB_WORKER_AUTH_KEY")
	if key := os.Getenv("OCR_WORKER_AUTH_KEY"); key != "" {
		decoded, err := base64.StdEncoding.DecodeString(key)
		if err != nil || len(decoded) != 32 || key == os.Getenv("JOB_WORKER_AUTH_KEY") {
			return Server{}, errors.New("invalid separate OCR worker key")
		}
		config.OCRWorkerAuthKey = decoded
	}
	if decoded, err := base64.StdEncoding.DecodeString(jobKey); err == nil && len(decoded) == 32 {
		config.JobWorkerAuthKey = decoded
	} else if config.Environment == "staging" || config.Environment == "production" {
		return Server{}, errors.New("invalid job worker configuration")
	}
	downloadKey := os.Getenv("EXPORT_DOWNLOAD_SIGNING_KEY")
	if decoded, err := base64.StdEncoding.DecodeString(downloadKey); err == nil && len(decoded) == 32 {
		config.ExportDownloadKey = decoded
	} else if config.Environment == "staging" || config.Environment == "production" {
		return Server{}, errors.New("invalid export download configuration")
	}
	config.ExportBucket = os.Getenv("EXPORT_BUCKET")
	config.ExportRegion = os.Getenv("EXPORT_REGION")
	config.ExportEndpoint = os.Getenv("EXPORT_ENDPOINT")
	config.ExportAccessKeyID = os.Getenv("EXPORT_ACCESS_KEY_ID")
	config.ExportSecretKey = os.Getenv("EXPORT_SECRET_ACCESS_KEY")
	config.ExportPathStyle = os.Getenv("EXPORT_PATH_STYLE") == "true"
	exportKey := os.Getenv("EXPORT_ENCRYPTION_KEY")
	exportConfigured := config.ExportBucket != "" || config.ExportRegion != "" || config.ExportEndpoint != "" || config.ExportAccessKeyID != "" || config.ExportSecretKey != "" || exportKey != ""
	if len(config.JobWorkerAuthKey) > 0 && (config.Environment == "staging" || config.Environment == "production") && !exportConfigured {
		return Server{}, errors.New("export storage is required")
	}
	if exportConfigured {
		decoded, err := base64.StdEncoding.DecodeString(exportKey)
		if config.ExportBucket == "" || config.ExportRegion == "" || config.ExportEndpoint == "" || config.ExportAccessKeyID == "" || config.ExportSecretKey == "" || err != nil || len(decoded) != 32 {
			return Server{}, errors.New("invalid export storage configuration")
		}
		config.ExportEncryptionKey = decoded
	}
	return config, nil
}

func LoadJobWorker() (JobWorker, error) {
	config := JobWorker{Environment: os.Getenv("APP_ENV"), APIURL: strings.TrimRight(os.Getenv("JOB_API_URL"), "/"), WorkerID: os.Getenv("JOB_WORKER_ID")}
	decoded, err := base64.StdEncoding.DecodeString(os.Getenv("JOB_WORKER_AUTH_KEY"))
	if config.Environment == "" || config.APIURL == "" || config.WorkerID == "" || err != nil || len(decoded) != 32 {
		return JobWorker{}, errors.New("invalid job worker configuration")
	}
	parsed, err := url.Parse(config.APIURL)
	if err != nil || parsed.Host == "" || (config.Environment == "staging" || config.Environment == "production") && parsed.Scheme != "https" {
		return JobWorker{}, errors.New("invalid job worker API URL")
	}
	config.AuthKey = decoded
	return config, nil
}

func LoadWorker() (Worker, error) {
	config := Worker{Environment: os.Getenv("APP_ENV"), DatabaseURL: os.Getenv("DATABASE_URL"),
		LineChannelAccessToken: os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"), WebBaseURL: os.Getenv("WEB_BASE_URL"),
		PoolMax: int32(number("WORKER_DB_POOL_MAX", 5)), DocumentBucket: os.Getenv("DOCUMENT_BUCKET"), DocumentRegion: os.Getenv("DOCUMENT_REGION"),
		DocumentEndpoint: os.Getenv("DOCUMENT_ENDPOINT"), DocumentAccessKeyID: os.Getenv("DOCUMENT_ACCESS_KEY_ID"), DocumentSecretKey: os.Getenv("DOCUMENT_SECRET_ACCESS_KEY"),
		DocumentPathStyle: os.Getenv("DOCUMENT_PATH_STYLE") == "true", ClamDAddress: os.Getenv("CLAMD_ADDR"), SkipDocumentScan: os.Getenv("SKIP_DOCUMENT_SCAN") == "true"}
	if config.SkipDocumentScan && config.Environment != "staging" {
		return Worker{}, errors.New("document scan bypass is restricted to staging")
	}
	if config.Environment == "" || config.DatabaseURL == "" || config.LineChannelAccessToken == "" || config.WebBaseURL == "" {
		return Worker{}, errors.New("missing required worker configuration")
	}
	documentKey := os.Getenv("DOCUMENT_ENCRYPTION_KEY")
	documentConfigured := config.DocumentBucket != "" || config.DocumentRegion != "" || config.DocumentEndpoint != "" || config.DocumentAccessKeyID != "" || config.DocumentSecretKey != "" || documentKey != "" || config.ClamDAddress != "" || config.SkipDocumentScan
	if (config.Environment == "staging" || config.Environment == "production") && !documentConfigured {
		return Worker{}, errors.New("document storage and scanner are required")
	}
	if documentConfigured {
		decoded, err := base64.StdEncoding.DecodeString(documentKey)
		if config.DocumentBucket == "" || config.DocumentRegion == "" || config.DocumentEndpoint == "" || config.DocumentAccessKeyID == "" || config.DocumentSecretKey == "" || (!config.SkipDocumentScan && config.ClamDAddress == "") || err != nil || len(decoded) != 32 {
			return Worker{}, errors.New("invalid document configuration")
		}
		config.DocumentEncryptionKey = decoded
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
