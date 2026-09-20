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
	"plaiflow/api/internal/document"
	"plaiflow/api/internal/drive"
	"plaiflow/api/internal/httpapi"
	"plaiflow/api/internal/plan"
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
	var driveService *drive.Service
	if settings.GoogleDriveClientID != "" {
		provider, providerErr := drive.NewGoogleProvider(drive.GoogleConfig{
			ClientID: settings.GoogleDriveClientID, ClientSecret: settings.GoogleDriveSecret,
			RedirectURI: settings.WebBaseURL + "/api/drive/callback",
		})
		if providerErr != nil {
			logger.Error("drive_provider_initialization_failed")
			os.Exit(1)
		}
		driveService, err = drive.New(drive.Config{Provider: provider, Store: store, EncryptionKey: settings.GoogleDriveTokenKey})
		if err != nil {
			logger.Error("drive_initialization_failed")
			os.Exit(1)
		}
	}
	documentService := &document.Service{Committer: store, Reader: store}
	if settings.DocumentBucket != "" {
		blob, blobErr := document.NewS3Blob(document.S3Config{
			Bucket: settings.DocumentBucket, Region: settings.DocumentRegion, Endpoint: settings.DocumentEndpoint,
			AccessKeyID: settings.DocumentAccessKeyID, SecretAccessKey: settings.DocumentSecretKey,
			PathStyle: settings.DocumentPathStyle,
		})
		if blobErr != nil {
			logger.Error("document_storage_initialization_failed")
			os.Exit(1)
		}
		encrypted, encryptErr := document.NewEncryptedStore(blob, settings.DocumentEncryptionKey)
		if encryptErr != nil {
			logger.Error("document_encryption_initialization_failed")
			os.Exit(1)
		}
		documentService.Intake = document.Intake{Temporary: encrypted,
			Scanner: document.ClamAV{Address: settings.ClamDAddress}}
	}
	server := &http.Server{
		Addr: ":" + settings.Port,
		Handler: httpapi.New(httpapi.Config{
			LineSecret: settings.LineSecret, LineChannel: settings.LineChannel, DashboardTokens: settings.DashboardTokens,
			Logger: logger, Auth: authService, Tenants: store, Work: store, Business: store, PlanStore: store,
			Gate: plan.Gate{Store: store}, Drive: driveService, Documents: documentService,
		}, store),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
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
