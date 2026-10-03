package main

import (
	"context"
	"encoding/base64"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"plaiflow/api/internal/auth"
	"plaiflow/api/internal/billing"
	"plaiflow/api/internal/classification"
	"plaiflow/api/internal/config"
	"plaiflow/api/internal/document"
	"plaiflow/api/internal/drive"
	"plaiflow/api/internal/httpapi"
	"plaiflow/api/internal/job"
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
	provider := os.Getenv("OCR_PROVIDER")
	if provider != "" && provider != "railway" && provider != "runpod" {
		logger.Error("ocr_provider_invalid")
		os.Exit(1)
	}
	autoMatch, reviewMatch, err := matchingThresholds()
	if err != nil {
		logger.Error("matching_thresholds_invalid")
		os.Exit(1)
	}
	classifierEnabled := os.Getenv("DOCUMENT_CLASSIFIER_ENABLED") == "true"
	classifierAuto, classifierReview := .9, .7
	if value := os.Getenv("DOCUMENT_CLASSIFIER_RULE_THRESHOLD"); value != "" {
		classifierAuto, err = strconv.ParseFloat(value, 64)
	}
	if err == nil {
		if value := os.Getenv("DOCUMENT_CLASSIFIER_REVIEW_THRESHOLD"); value != "" {
			classifierReview, err = strconv.ParseFloat(value, 64)
		}
	}
	if err != nil || classifierReview <= 0 || classifierReview >= classifierAuto || classifierAuto > 1 {
		logger.Error("document_classifier_thresholds_invalid")
		os.Exit(1)
	}
	var llmClassifier classification.DocumentClassifier
	if classifierEnabled && os.Getenv("DOCUMENT_CLASSIFIER_URL") != "" {
		if os.Getenv("DOCUMENT_CLASSIFIER_PROVIDER") != "runpod" {
			logger.Error("document_classifier_provider_invalid")
			os.Exit(1)
		}
		endpoint, parseErr := url.Parse(os.Getenv("DOCUMENT_CLASSIFIER_URL"))
		if parseErr != nil || endpoint.Host == "" || endpoint.Scheme != "https" && !(endpoint.Scheme == "http" && (endpoint.Hostname() == "localhost" || endpoint.Hostname() == "127.0.0.1")) || os.Getenv("DOCUMENT_CLASSIFIER_MODEL") == "" {
			logger.Error("document_classifier_endpoint_invalid")
			os.Exit(1)
		}
		llmClassifier = classification.HTTPClassifier{URL: os.Getenv("DOCUMENT_CLASSIFIER_URL"), Model: os.Getenv("DOCUMENT_CLASSIFIER_MODEL"), Token: os.Getenv("DOCUMENT_CLASSIFIER_API_KEY"), Client: &http.Client{Timeout: 5 * time.Second}}
	}
	if settings.SkipDocumentScan {
		logger.Warn("document_scan_bypassed")
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
	if os.Getenv("REVIEW_ENABLED") == "true" {
		store.EnableReview()
	}
	if classifierEnabled {
		store.EnableClassification()
	}
	matchingEnabled := os.Getenv("MATCHING_ENABLED") == "true"
	if matchingEnabled {
		store.EnableMatching()
	}
	runPodOrganizations := secretaryPilotOrganizations(os.Getenv("OCR_RUNPOD_ENABLED_ORGS"))
	if provider == "runpod" || len(runPodOrganizations) > 0 {
		store.RequireMigration19()
	}
	billingEnabled := os.Getenv("BILLING_ENABLED") == "true"
	var billingService *billing.Service
	if billingEnabled {
		secretKey, webhookSecret := os.Getenv("OMISE_SECRET_KEY"), os.Getenv("OMISE_WEBHOOK_SECRET")
		previousSecret := os.Getenv("OMISE_WEBHOOK_PREVIOUS_SECRET")
		decodedSecret, decodeErr := base64.StdEncoding.DecodeString(webhookSecret)
		decodedPrevious, previousErr := base64.StdEncoding.DecodeString(previousSecret)
		validKey := (settings.Environment == "production" && strings.HasPrefix(secretKey, "skey_live_")) ||
			(settings.Environment != "production" && strings.HasPrefix(secretKey, "skey_test_"))
		if !validKey || decodeErr != nil || len(decodedSecret) < 16 ||
			(previousSecret != "" && (previousErr != nil || len(decodedPrevious) < 16 || previousSecret == webhookSecret)) {
			logger.Error("billing_configuration_invalid")
			os.Exit(1)
		}
		store.EnableBilling()
		billingService = &billing.Service{Store: store, Provider: billing.Omise{SecretKey: secretKey},
			Live: settings.Environment == "production", Now: time.Now, Logger: logger}
		go runBilling(ctx, *billingService, logger)
	}
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
	var artifactStore job.ArtifactStore
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
			Scanner: document.ClamAV{Address: settings.ClamDAddress}, SkipScan: settings.SkipDocumentScan}
		artifactStore = encrypted
	}
	if settings.ExportBucket != "" {
		blob, blobErr := document.NewS3Blob(document.S3Config{Bucket: settings.ExportBucket, Region: settings.ExportRegion, Endpoint: settings.ExportEndpoint,
			AccessKeyID: settings.ExportAccessKeyID, SecretAccessKey: settings.ExportSecretKey, PathStyle: settings.ExportPathStyle})
		if blobErr != nil {
			logger.Error("export_storage_initialization_failed")
			os.Exit(1)
		}
		artifactStore, err = document.NewEncryptedStore(blob, settings.ExportEncryptionKey)
		if err != nil {
			logger.Error("export_encryption_initialization_failed")
			os.Exit(1)
		}
	}
	var workerAuth *job.WorkerAuth
	var ocrAuth *job.WorkerAuth
	var ocrTokens *job.ArtifactToken
	if len(settings.OCRWorkerAuthKey) > 0 {
		ocrAuth, err = job.NewWorkerAuth(settings.OCRWorkerAuthKey, settings.Environment, []string{"ocr:claim", "ocr:heartbeat", "ocr:input", "ocr:submit", "ocr:fail"})
		if err != nil {
			logger.Error("ocr_auth_initialization_failed")
			os.Exit(1)
		}
		ocrTokens, err = job.NewArtifactToken(settings.ExportDownloadKey)
		if err != nil {
			logger.Error("ocr_token_initialization_failed")
			os.Exit(1)
		}
	}
	var artifactTokens *job.ArtifactToken
	if len(settings.JobWorkerAuthKey) > 0 {
		workerAuth, err = job.NewWorkerAuth(settings.JobWorkerAuthKey, settings.Environment, []string{"jobs:claim", "jobs:heartbeat", "jobs:read", "jobs:generate", "jobs:artifact", "jobs:fail"})
		if err != nil {
			logger.Error("job_worker_auth_initialization_failed")
			os.Exit(1)
		}
		artifactTokens, err = job.NewArtifactToken(settings.ExportDownloadKey)
		if err != nil {
			logger.Error("artifact_token_initialization_failed")
			os.Exit(1)
		}
	}
	server := &http.Server{
		Addr: ":" + settings.Port,
		Handler: httpapi.New(httpapi.Config{
			LineSecret: settings.LineSecret, LineChannel: settings.LineChannel, DashboardTokens: settings.DashboardTokens,
			Logger: logger, Auth: authService, Tenants: store, Work: store, Business: store, PlanStore: store,
			Gate: plan.Gate{Store: store}, Drive: driveService, Documents: documentService,
			Jobs: store, JobWorkerAuth: workerAuth, JobArtifacts: artifactStore, ArtifactTokens: artifactTokens,
			OCR: store, OCRJobs: store, OCRAuth: ocrAuth, OCRTokens: ocrTokens, OCRStorage: documentService.Intake.Temporary,
			Extraction: store, ExtractionEnabled: os.Getenv("EXTRACTION_ENABLED") == "true", OCRPilotOrganizations: secretaryPilotOrganizations(os.Getenv("OCR_PILOT_ORGANIZATION_IDS")), ReviewEnabled: os.Getenv("REVIEW_ENABLED") == "true",
			Classification: store, ClassificationEnabled: classifierEnabled, ClassificationOrganizations: secretaryPilotOrganizations(os.Getenv("DOCUMENT_CLASSIFIER_ORGANIZATION_IDS")), ClassificationRuleThreshold: classifierAuto, ClassificationReviewThreshold: classifierReview, Classifier: llmClassifier,
			OCRDefaultProvider: provider, OCRRunPodOrganizations: runPodOrganizations,
			Matching: store, MatchingEnabled: matchingEnabled, AutoMatchThreshold: autoMatch, ReviewMatchThreshold: reviewMatch,
			Accounting: store, AccountingEnabled: os.Getenv("ACCOUNTING_ENABLED") == "true", ReviewExports: store,
			Firm: store, FirmEnabled: os.Getenv("FIRM_ENABLED") == "true",
			Secretary: store, SecretaryEnabled: os.Getenv("SECRETARY_ENABLED") == "true",
			SecretaryPilotOrganizations: secretaryPilotOrganizations(os.Getenv("SECRETARY_PILOT_ORGANIZATION_IDS")),
			Billing:                     billingService, BillingEnabled: billingEnabled, BillingTestOrganizationID: os.Getenv("BILLING_TEST_ORGANIZATION_ID"),
			OmiseWebhookSecret: os.Getenv("OMISE_WEBHOOK_SECRET"), OmiseWebhookPreviousSecret: os.Getenv("OMISE_WEBHOOK_PREVIOUS_SECRET"),
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

func runBilling(ctx context.Context, service billing.Service, logger *slog.Logger) {
	events := time.NewTicker(5 * time.Second)
	reconcile := time.NewTicker(15 * time.Minute)
	defer events.Stop()
	defer reconcile.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-events.C:
			work, cancel := context.WithTimeout(ctx, 20*time.Second)
			count, err := service.ProcessDue(work)
			cancel()
			if err != nil {
				logger.Warn("billing_event_processing_unavailable")
			} else if count > 0 {
				logger.Info("billing_events_processed", "count", count)
			}
		case <-reconcile.C:
			work, cancel := context.WithTimeout(ctx, 30*time.Second)
			count, err := service.ReconcilePending(work)
			cancel()
			if err != nil {
				logger.Warn("billing_reconciliation_unavailable", "checked", count)
			} else {
				logger.Info("billing_reconciliation", "checked", count)
			}
			if count == 100 {
				logger.Warn("billing_reconciliation_capacity", "checked", count)
			}
		}
	}
}

func secretaryPilotOrganizations(value string) map[string]bool {
	organizations := map[string]bool{}
	for _, id := range strings.Split(value, ",") {
		if id = strings.TrimSpace(id); id != "" {
			organizations[id] = true
		}
	}
	return organizations
}

func matchingThresholds() (float64, float64, error) {
	parse := func(name string, fallback float64) (float64, error) {
		if os.Getenv(name) == "" {
			return fallback, nil
		}
		return strconv.ParseFloat(os.Getenv(name), 64)
	}
	auto, err := parse("OCR_AUTO_MATCH_THRESHOLD", .9)
	if err != nil {
		return 0, 0, err
	}
	review, err := parse("OCR_REVIEW_THRESHOLD", .7)
	if err != nil {
		return 0, 0, err
	}
	if !(review > 0 && review < auto && auto <= 1) {
		return 0, 0, strconv.ErrRange
	}
	return auto, review, nil
}
