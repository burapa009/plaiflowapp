package httpapi

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"plaiflow/api/internal/job"
)

func (s *server) registerJobRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /internal/v1/jobs/claim", s.claimJobs)
	mux.HandleFunc("POST /internal/v1/jobs/{job}/heartbeat", s.heartbeatJob)
	mux.HandleFunc("POST /internal/v1/jobs/{job}/fail", s.failJob)
	mux.HandleFunc("GET /internal/v1/jobs/{job}/export-rows", s.exportRows)
	mux.HandleFunc("PUT /internal/v1/jobs/{job}/artifact", s.uploadArtifact)
	mux.HandleFunc("GET /v1/export-artifacts/{job}", s.retrieveArtifact)
}

func (s *server) retrieveArtifact(w http.ResponseWriter, r *http.Request) {
	if s.config.ArtifactTokens == nil || s.config.JobArtifacts == nil {
		http.NotFound(w, r)
		return
	}
	jobID, organizationID, userID, err := s.config.ArtifactTokens.Verify(r.URL.Query().Get("token"), s.config.Now().UTC())
	if err != nil || jobID != r.PathValue("job") {
		http.NotFound(w, r)
		return
	}
	artifact, err := s.config.Jobs.AuthorizeExportArtifact(r.Context(), userID, organizationID, jobID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := s.config.Jobs.RecordArtifactDownload(r.Context(), userID, organizationID, jobID, s.config.Now().UTC()); err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "artifact_unavailable", "Artifact is unavailable")
		return
	}
	body, err := s.config.JobArtifacts.Open(r.Context(), artifact.ObjectKey)
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "artifact_unavailable", "Artifact is unavailable")
		return
	}
	defer body.Close()
	contentType := "text/csv; charset=utf-8"
	if artifact.Format == "xlsx" {
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", `attachment; filename="plaiflow-export.`+artifact.Format+`"`)
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Length", strconv.FormatInt(artifact.ByteCount, 10))
	_, _ = io.Copy(w, body)
}

func (s *server) claimJobs(w http.ResponseWriter, r *http.Request) {
	identity, ok := s.workerAuthorized(w, r, "jobs:claim")
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	var input struct {
		Kinds []job.Kind `json:"kinds"`
		Limit int        `json:"limit"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil || input.Limit < 1 || input.Limit > 10 || !validJobKinds(input.Kinds) {
		writeError(w, r, http.StatusBadRequest, "invalid_claim", "Job claim is invalid")
		return
	}
	claimed, err := s.config.Jobs.ClaimJobs(r.Context(), job.ClaimCommand{WorkerID: identity.WorkerID, Environment: identity.Environment, Kinds: input.Kinds, Limit: input.Limit, Lease: 2 * time.Minute, Now: s.config.Now().UTC()})
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "claim_unavailable", "Jobs are temporarily unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": claimed})
}

func (s *server) leaseCommand(r *http.Request) job.LeaseCommand {
	return job.LeaseCommand{JobID: r.PathValue("job"), AttemptID: r.Header.Get("X-Job-Attempt"), LeaseToken: r.Header.Get("X-Job-Lease"), Now: s.config.Now().UTC()}
}

func (s *server) heartbeatJob(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.workerAuthorized(w, r, "jobs:heartbeat"); !ok {
		return
	}
	expires, err := s.config.Jobs.HeartbeatJob(r.Context(), s.leaseCommand(r), 2*time.Minute)
	if err != nil {
		s.writeJobError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lease_expires_at": expires})
}

func (s *server) failJob(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.workerAuthorized(w, r, "jobs:fail"); !ok {
		return
	}
	var input struct {
		Code              string `json:"code"`
		RetryAfterSeconds int    `json:"retry_after_seconds"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil || !validFailureCode(input.Code) || input.RetryAfterSeconds < 0 || input.RetryAfterSeconds > 900 {
		writeError(w, r, http.StatusBadRequest, "invalid_failure", "Job failure is invalid")
		return
	}
	err := s.config.Jobs.FailJob(r.Context(), job.FailureCommand{LeaseCommand: s.leaseCommand(r), Code: input.Code, RetryAfter: time.Duration(input.RetryAfterSeconds) * time.Second})
	if err != nil {
		s.writeJobError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) exportRows(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.workerAuthorized(w, r, "jobs:read"); !ok {
		return
	}
	page, err := s.config.Jobs.ReadExportPage(r.Context(), s.leaseCommand(r), r.URL.Query().Get("cursor"), 500)
	if err != nil {
		s.writeJobError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *server) uploadArtifact(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.workerAuthorized(w, r, "jobs:artifact"); !ok {
		return
	}
	if s.config.JobArtifacts == nil {
		writeError(w, r, http.StatusServiceUnavailable, "artifact_unavailable", "Artifact storage is unavailable")
		return
	}
	format := r.URL.Query().Get("format")
	rows, rowErr := strconv.ParseInt(r.URL.Query().Get("rows"), 10, 64)
	if format != "csv" && format != "xlsx" || rowErr != nil || rows < 0 {
		writeError(w, r, http.StatusBadRequest, "invalid_artifact", "Artifact is invalid")
		return
	}
	max := int64(256 << 20)
	if format == "xlsx" {
		max = 128 << 20
	}
	limited := http.MaxBytesReader(w, r.Body, max+1)
	hash := sha256.New()
	counter := &byteCounter{reader: io.TeeReader(limited, hash)}
	lease := s.leaseCommand(r)
	key := fmt.Sprintf("exports/%s/%s/%s.%s", lease.JobID, lease.AttemptID, lease.JobID, format)
	if err := s.config.JobArtifacts.Put(r.Context(), key, counter); err != nil || counter.bytes > max {
		_ = s.config.JobArtifacts.Delete(r.Context(), key)
		writeError(w, r, http.StatusUnprocessableEntity, "invalid_artifact", "Artifact could not be stored")
		return
	}
	artifact, err := s.config.Jobs.CompleteExport(r.Context(), job.CompleteExportCommand{LeaseCommand: lease, Format: format, RowCount: rows, ByteCount: counter.bytes, SHA256: fmt.Sprintf("%x", hash.Sum(nil)), ObjectKey: key, ExpiresAt: s.config.Now().UTC().Add(24 * time.Hour)})
	if err != nil {
		_ = s.config.JobArtifacts.Delete(r.Context(), key)
		s.writeJobError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job_id": artifact.JobID, "status": job.Completed, "expires_at": artifact.ExpiresAt})
}

type byteCounter struct {
	reader io.Reader
	bytes  int64
}

func (c *byteCounter) Read(p []byte) (int, error) {
	n, err := c.reader.Read(p)
	c.bytes += int64(n)
	return n, err
}

func (s *server) writeJobError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, job.ErrLeaseLost) {
		writeError(w, r, http.StatusConflict, "lease_lost", "Job lease is no longer current")
		return
	}
	s.config.Logger.Error("job_request_failed", "job_id", r.PathValue("job"), "error", err)
	writeError(w, r, http.StatusServiceUnavailable, "job_unavailable", "Job is temporarily unavailable")
}

func validFailureCode(code string) bool {
	switch code {
	case "temporary_upstream", "rate_limited", "artifact_upload_failed", "invalid_input", "unauthorized_input", "unsupported_format":
		return true
	}
	return false
}

func (s *server) workerAuthorized(w http.ResponseWriter, r *http.Request, scope string) (job.WorkerIdentity, bool) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	identity, err := s.config.JobWorkerAuth.Verify(token, scope, s.config.Now().UTC())
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "Worker access is unauthorized")
		return job.WorkerIdentity{}, false
	}
	return identity, true
}

func validJobKinds(kinds []job.Kind) bool {
	if len(kinds) == 0 || len(kinds) > 2 {
		return false
	}
	seen := map[job.Kind]bool{}
	for _, kind := range kinds {
		if kind != job.Export && kind != job.OCR || seen[kind] {
			return false
		}
		seen[kind] = true
	}
	return true
}
