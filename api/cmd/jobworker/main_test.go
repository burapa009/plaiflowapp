package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"plaiflow/api/internal/job"
)

func TestExportStreamsFormulaSafeCSVToArtifactEndpoint(t *testing.T) {
	var artifact string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /internal/v1/jobs/job-1/heartbeat", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /internal/v1/jobs/job-1/export-rows", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"rows":[{"OrganizationID":"org-1","OrganizationName":"=IMPORTXML()","TaskID":"task-1","Title":"+SUM(1,1)","Status":"Open","Priority":"Normal","CreatedAt":"2026-09-21T00:00:00Z","UpdatedAt":"2026-09-21T00:00:00Z"}],"done":true}`)
	})
	mux.HandleFunc("PUT /internal/v1/jobs/job-1/artifact", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		artifact = string(body)
		_, _ = io.WriteString(w, `{}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	auth, _ := job.NewWorkerAuth([]byte("01234567890123456789012345678901"), "test", []string{"jobs:heartbeat", "jobs:read", "jobs:artifact", "jobs:fail"})
	worker := apiWorker{baseURL: server.URL, workerID: "worker-1", auth: auth, client: server.Client(), logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	claimed := job.Claimed{Job: job.Job{ID: "job-1", Kind: job.Export, AttemptID: "attempt-1", Payload: []byte(`{"format":"csv"}`)}, LeaseToken: "lease-1", LeaseExpiresAt: time.Now().Add(time.Minute)}
	if err := worker.process(t.Context(), claimed); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(artifact, "\ufefforganization_id") || !strings.Contains(artifact, "'=IMPORTXML()") || !strings.Contains(artifact, "'+SUM(1,1)") {
		t.Fatalf("unsafe artifact: %q", artifact)
	}
}

func TestApprovedExportWritesVersionedFormulaSafeCSV(t *testing.T) {
	var artifact string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /internal/v1/jobs/job-1/heartbeat", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	mux.HandleFunc("GET /internal/v1/jobs/job-1/export-rows", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"product":"approved_suggestions","header":["row_id","reviewed_seller_name","template_version"],"values":[["approval-1","=IMPORTXML(1)","accounting_suggestions_v1"]],"done":true}`)
	})
	mux.HandleFunc("PUT /internal/v1/jobs/job-1/artifact", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		artifact = string(body)
		_, _ = io.WriteString(w, `{}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	auth, _ := job.NewWorkerAuth([]byte("01234567890123456789012345678901"), "test", []string{"jobs:heartbeat", "jobs:read", "jobs:artifact", "jobs:fail"})
	worker := apiWorker{baseURL: server.URL, workerID: "worker-1", auth: auth, client: server.Client(), logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	claimed := job.Claimed{Job: job.Job{ID: "job-1", Kind: job.Export, AttemptID: "attempt-1",
		Payload: []byte(`{"export_type":"approved_suggestions","format":"csv","row_count":1}`)}, LeaseToken: "lease-1", LeaseExpiresAt: time.Now().Add(time.Minute)}
	if err := worker.process(t.Context(), claimed); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(artifact, "\ufeffrow_id,reviewed_seller_name,template_version\r\n") ||
		!strings.Contains(artifact, "'=IMPORTXML(1)") || !strings.Contains(artifact, "accounting_suggestions_v1") {
		t.Fatalf("wrong approved artifact: %q", artifact)
	}
}
