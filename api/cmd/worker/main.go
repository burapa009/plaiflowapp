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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	store, err := postgres.New(ctx, settings.DatabaseURL, settings.PoolMax, logger)
	if err != nil {
		logger.Error("database_initialization_failed")
		os.Exit(1)
	}
	defer store.Close()
	sender := lineadapter.NewSender(&http.Client{Timeout: 5 * time.Second}, "https://api.line.me", settings.LineChannelAccessToken, settings.WebBaseURL)
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
			if err := worker.RunOnce(ctx, store, logger, worker.RecordOnly); err != nil {
				logger.Error("worker_cycle_failed")
			}
			if _, err := worker.RunWorkOnce(ctx, store, sender, time.Now().UTC()); err != nil {
				logger.Error("work_cycle_failed")
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
