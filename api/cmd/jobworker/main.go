package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"plaiflow/api/internal/config"
	"plaiflow/api/internal/document"
	"plaiflow/api/internal/job"
	"plaiflow/api/internal/work"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	settings, err := config.LoadJobWorker()
	if err != nil {
		logger.Error("configuration_invalid")
		os.Exit(1)
	}
	auth, err := job.NewWorkerAuth(settings.AuthKey, settings.Environment, []string{"jobs:claim", "jobs:heartbeat", "jobs:read", "jobs:artifact", "jobs:fail"})
	if err != nil {
		logger.Error("worker_auth_invalid")
		os.Exit(1)
	}
	worker := apiWorker{baseURL: settings.APIURL, workerID: settings.WorkerID, auth: auth, client: &http.Client{Timeout: 90 * time.Second}, logger: logger}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	for backoff := time.Second; ctx.Err() == nil; {
		claimed, err := worker.claim(ctx)
		if err != nil {
			logger.Warn("job_claim_failed", "error", err)
			wait(ctx, backoff)
			backoff = min(backoff*2, 15*time.Second)
			continue
		}
		if len(claimed) == 0 {
			wait(ctx, backoff)
			backoff = min(backoff*2, 15*time.Second)
			continue
		}
		backoff = time.Second
		for _, item := range claimed {
			if err := worker.process(ctx, item); err != nil {
				logger.Warn("job_processing_failed", "job_id", item.Job.ID, "attempt_id", item.Job.AttemptID, "error", err)
			}
		}
	}
}

type apiWorker struct {
	baseURL, workerID string
	auth              *job.WorkerAuth
	client            *http.Client
	logger            *slog.Logger
}

func (w apiWorker) token(scopes ...string) (string, error) {
	return w.auth.Sign(w.workerID, scopes, time.Now().UTC(), 2*time.Minute)
}
func (w apiWorker) request(ctx context.Context, method, path, scope string, body io.Reader, item *job.Claimed) (*http.Response, error) {
	token, err := w.token(scope)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, w.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if item != nil {
		req.Header.Set("X-Job-Attempt", item.Job.AttemptID)
		req.Header.Set("X-Job-Lease", item.LeaseToken)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return w.client.Do(req)
}
func (w apiWorker) claim(ctx context.Context) ([]job.Claimed, error) {
	body := bytes.NewBufferString(`{"kinds":["export"],"limit":1}`)
	response, err := w.request(ctx, http.MethodPost, "/internal/v1/jobs/claim", "jobs:claim", body, nil)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("claim status %d", response.StatusCode)
	}
	var result struct {
		Jobs []job.Claimed `json:"jobs"`
	}
	err = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result)
	return result.Jobs, err
}
func (w apiWorker) process(ctx context.Context, item job.Claimed) error {
	leaseCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	heartbeatErrors := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-leaseCtx.Done():
				return
			case <-ticker.C:
				if err := w.heartbeat(leaseCtx, item); err != nil {
					heartbeatErrors <- err
					cancel()
					return
				}
			}
		}
	}()
	err := w.processExport(leaseCtx, item)
	select {
	case heartbeatErr := <-heartbeatErrors:
		return heartbeatErr
	default:
		return err
	}
}

func (w apiWorker) processExport(ctx context.Context, item job.Claimed) error {
	if item.Job.Kind != job.Export {
		return w.fail(ctx, item, "unsupported_format")
	}
	var payload struct {
		Format     string `json:"format"`
		ExportType string `json:"export_type"`
		RowCount   int64  `json:"row_count"`
	}
	if json.Unmarshal(item.Job.Payload, &payload) != nil || payload.Format != "csv" && payload.Format != "xlsx" {
		return w.fail(ctx, item, "unsupported_format")
	}
	file, err := os.CreateTemp("", "plaiflow-export-*."+payload.Format)
	if err != nil {
		return w.fail(ctx, item, "artifact_upload_failed")
	}
	defer os.Remove(file.Name())
	defer file.Close()
	documentExport := payload.ExportType == "raw_documents" || payload.ExportType == "confirmed_values" || payload.ExportType == "approved_suggestions"
	var exportWriter interface {
		Write(work.ExportRow) error
		Close() error
	}
	if !documentExport {
		if payload.Format == "xlsx" {
			exportWriter, err = work.NewXLSXWriter(file)
		} else {
			exportWriter, err = work.NewCSVWriter(file)
		}
		if err != nil {
			return w.fail(ctx, item, "artifact_upload_failed")
		}
	}
	cursor := ""
	var count int64
	var header []string
	var values [][]string
	for {
		if err = w.heartbeat(ctx, item); err != nil {
			return err
		}
		page, readErr := w.readPage(ctx, item, cursor)
		if readErr != nil {
			return w.fail(ctx, item, "temporary_upstream")
		}
		if documentExport {
			if page.Product != payload.ExportType || len(page.Header) == 0 || header != nil && !sameHeader(header, page.Header) || count+int64(len(page.Values)) > payload.RowCount {
				return w.fail(ctx, item, "invalid_input")
			}
			header = page.Header
			values = append(values, page.Values...)
			count += int64(len(page.Values))
		} else {
			for _, row := range page.Rows {
				if err = exportWriter.Write(row); err != nil {
					return w.fail(ctx, item, "artifact_upload_failed")
				}
				count++
			}
		}
		if page.Done {
			break
		}
		cursor = page.NextCursor
	}
	if documentExport {
		if count != payload.RowCount || count > 20000 || header == nil {
			return w.fail(ctx, item, "invalid_input")
		}
		if err = document.WriteTable(file, payload.Format, header, values); err != nil {
			return w.fail(ctx, item, "artifact_upload_failed")
		}
	} else if err = exportWriter.Close(); err != nil {
		return w.fail(ctx, item, "artifact_upload_failed")
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return w.fail(ctx, item, "artifact_upload_failed")
	}
	path := "/internal/v1/jobs/" + url.PathEscape(item.Job.ID) + "/artifact?format=" + payload.Format + "&rows=" + fmt.Sprint(count)
	response, err := w.request(ctx, http.MethodPut, path, "jobs:artifact", file, &item)
	if err != nil {
		return w.fail(ctx, item, "artifact_upload_failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_ = w.fail(ctx, item, "artifact_upload_failed")
		return fmt.Errorf("artifact status %d", response.StatusCode)
	}
	w.logger.Info("job_completed", "job_id", item.Job.ID, "attempt_id", item.Job.AttemptID, "rows", count)
	return nil
}

func sameHeader(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func (w apiWorker) heartbeat(ctx context.Context, item job.Claimed) error {
	response, err := w.request(ctx, http.MethodPost, "/internal/v1/jobs/"+url.PathEscape(item.Job.ID)+"/heartbeat", "jobs:heartbeat", nil, &item)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return errors.New("job lease lost")
	}
	return nil
}
func (w apiWorker) readPage(ctx context.Context, item job.Claimed, cursor string) (job.ExportPage, error) {
	path := "/internal/v1/jobs/" + url.PathEscape(item.Job.ID) + "/export-rows"
	if cursor != "" {
		path += "?cursor=" + url.QueryEscape(cursor)
	}
	response, err := w.request(ctx, http.MethodGet, path, "jobs:read", nil, &item)
	if err != nil {
		return job.ExportPage{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return job.ExportPage{}, fmt.Errorf("read status %d", response.StatusCode)
	}
	var page job.ExportPage
	err = json.NewDecoder(io.LimitReader(response.Body, 16<<20)).Decode(&page)
	return page, err
}
func (w apiWorker) fail(ctx context.Context, item job.Claimed, code string) error {
	body, _ := json.Marshal(map[string]any{"code": code})
	response, err := w.request(ctx, http.MethodPost, "/internal/v1/jobs/"+url.PathEscape(item.Job.ID)+"/fail", "jobs:fail", bytes.NewReader(body), &item)
	if err == nil {
		defer response.Body.Close()
		if response.StatusCode != http.StatusNoContent {
			return fmt.Errorf("fail status %d", response.StatusCode)
		}
	}
	return err
}
func wait(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
