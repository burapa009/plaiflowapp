package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"plaiflow/api/internal/config"
	"plaiflow/api/internal/document"
	lineadapter "plaiflow/api/internal/line"
	"plaiflow/api/internal/postgres"
	"plaiflow/api/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	settings, err := config.LoadWorker()
	if err != nil {
		logger.Error("configuration_invalid")
		os.Exit(1)
	}
	if settings.SkipDocumentScan {
		logger.Warn("document_scan_bypassed")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	store, err := postgres.New(ctx, settings.DatabaseURL, settings.PoolMax, logger)
	if err != nil {
		logger.Error("database_initialization_failed")
		os.Exit(1)
	}
	defer store.Close()
	sender := lineadapter.NewSender(&http.Client{Timeout: 5 * time.Second}, "https://api.line.me", settings.LineChannelAccessToken, settings.WebBaseURL)
	process := worker.RecordOnly
	if settings.DocumentBucket != "" {
		blob, blobErr := document.NewS3Blob(document.S3Config{Bucket: settings.DocumentBucket, Region: settings.DocumentRegion, Endpoint: settings.DocumentEndpoint,
			AccessKeyID: settings.DocumentAccessKeyID, SecretAccessKey: settings.DocumentSecretKey, PathStyle: settings.DocumentPathStyle})
		if blobErr != nil {
			logger.Error("document_storage_initialization_failed")
			os.Exit(1)
		}
		encrypted, encryptErr := document.NewEncryptedStore(blob, settings.DocumentEncryptionKey)
		if encryptErr != nil {
			logger.Error("document_encryption_initialization_failed")
			os.Exit(1)
		}
		downloader, downloadErr := lineadapter.NewContentClient(&http.Client{Timeout: 30 * time.Second}, "https://api-data.line.me", settings.LineChannelAccessToken)
		if downloadErr != nil {
			logger.Error("line_content_initialization_failed")
			os.Exit(1)
		}
		documents := &document.Service{Intake: document.Intake{Temporary: encrypted, Scanner: document.ClamAV{Address: settings.ClamDAddress}, SkipScan: settings.SkipDocumentScan}, Committer: store}
		process = worker.NewLINEDocumentProcessor(documents, store, downloader)
	}
	poll := time.NewTicker(time.Second)
	heartbeat := time.NewTicker(30 * time.Second)
	cleanup := time.NewTicker(24 * time.Hour)
	defer poll.Stop()
	defer heartbeat.Stop()
	defer cleanup.Stop()
	_ = store.Heartbeat(ctx, "worker-1")
	for {
		select {
		case <-ctx.Done():
			return
		case <-poll.C:
			if err := worker.RunOnce(ctx, store, logger, process); err != nil {
				logger.Error("worker_cycle_failed")
			}
			if _, err := worker.RunWorkOnce(ctx, store, sender, time.Now().UTC()); err != nil {
				logger.Error("work_cycle_failed", "error", err)
			}
		case <-heartbeat.C:
			if err := store.Heartbeat(ctx, "worker-1"); err != nil {
				logger.Error("heartbeat_failed")
			}
		case <-cleanup.C:
			if err := store.Cleanup(ctx); err != nil {
				logger.Error("cleanup_failed")
			}
		}
	}
}
