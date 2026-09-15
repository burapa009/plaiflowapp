package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"plaiflow/api/internal/auth"
	"plaiflow/api/internal/config"
	"plaiflow/api/internal/httpapi"
	"plaiflow/api/internal/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	settings, err := config.LoadServer()
	if err != nil {
		logger.Error("configuration_invalid")
		os.Exit(1)
	}
	callback := auth.CallbackConfig{WebBaseURL: settings.WebBaseURL, AllowedOrigins: settings.AllowedWebOrigins}
	lineCallback, err := callback.CallbackURL("line")
	if err != nil {
		logger.Error("callback_configuration_invalid")
		os.Exit(1)
	}
	googleCallback, err := callback.CallbackURL("google")
	if err != nil {
		logger.Error("callback_configuration_invalid")
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
	authService, err := auth.NewService(auth.ServiceConfig{WebOrigin: settings.WebBaseURL, Providers: []auth.Provider{
		auth.NewLINEProvider(auth.ProviderConfig{ClientID: settings.LineLoginChannel, ClientSecret: settings.LineLoginSecret, RedirectURI: lineCallback}),
		auth.NewGoogleProvider(auth.ProviderConfig{ClientID: settings.GoogleClientID, ClientSecret: settings.GoogleClientSecret, RedirectURI: googleCallback}),
	}}, store)
	if err != nil {
		logger.Error("authentication_initialization_failed")
		os.Exit(1)
	}
	server := &http.Server{
		Addr:              ":" + settings.Port,
		Handler:           httpapi.New(httpapi.Config{LineSecret: settings.LineSecret, LineChannel: settings.LineChannel, DashboardTokens: settings.DashboardTokens, Logger: logger, Auth: authService, Tenants: store, Work: store}, store),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second,
	}
	go func() {
		logger.Info("server_started", "port", settings.Port, "environment", settings.Environment)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server_failed")
			stop()
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdown)
}
