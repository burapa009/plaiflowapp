package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"plaiflow/api/internal/classification"
	"plaiflow/api/internal/job"
	"plaiflow/api/internal/ocr"
	"plaiflow/api/internal/tenant"
)

func (s *server) registerOCRRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /internal/v1/ocr/jobs/claim", s.claimOCR)
	m.HandleFunc("POST /internal/v1/ocr/jobs/{job}/heartbeat", s.heartbeatOCR)
	m.HandleFunc("POST /internal/v1/ocr/jobs/{job}/provider", s.providerOCR)
	m.HandleFunc("POST /internal/v1/ocr/jobs/{job}/fail", s.failOCR)
	m.HandleFunc("GET /internal/v1/ocr/jobs/{job}/input", s.inputOCR)
	m.HandleFunc("GET /internal/v1/ocr/jobs/{job}/original", s.originalOCR)
	m.HandleFunc("PUT /internal/v1/ocr/jobs/{job}/result", s.submitOCR)
	m.HandleFunc("GET /v1/o/{organization}/documents/{document}/ocr", s.stateOCR)
	m.HandleFunc("POST /v1/o/{organization}/documents/{document}/ocr/retry", s.retryOCR)
	m.HandleFunc("GET /v1/ocr-results/{document}", s.retrieveOCR)
}
func (s *server) providerOCR(w http.ResponseWriter, r *http.Request) {
	_, id, ok := s.ocrInputFor(w, r, "heartbeat")
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	var input struct {
		ID          string `json:"provider_job_id"`
		ExecutionMS int64  `json:"execution_ms"`
		QueueMS     int64  `json:"queue_ms"`
		GPUClass    string `json:"gpu_class"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if dec.Decode(&input) != nil || input.ID == "" || len(input.ID) > 128 || input.ExecutionMS < 0 || input.QueueMS < 0 || len(input.GPUClass) > 100 {
		writeError(w, r, 400, "invalid_provider_job", "Invalid provider job")
		return
	}
	if err := s.config.OCR.RecordProviderJob(r.Context(), s.leaseCommand(r), id.WorkerID,
		ocr.ProviderJob{ID: input.ID, ExecutionMS: input.ExecutionMS, QueueMS: input.QueueMS, GPUClass: input.GPUClass}); err != nil {
		s.writeJobError(w, r, err)
		return
	}
	w.WriteHeader(204)
}
func (s *server) authorizeOCR(w http.ResponseWriter, r *http.Request, scope string) (job.WorkerIdentity, bool) {
	id, err := s.config.OCRAuth.Verify(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), "ocr:"+scope, s.config.Now())
	if err != nil {
		writeError(w, r, 401, "unauthorized", "Worker access denied")
		return id, false
	}
	return id, true
}
func (s *server) claimOCR(w http.ResponseWriter, r *http.Request) {
	id, ok := s.authorizeOCR(w, r, "claim")
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	var input struct {
		Provider string `json:"provider"`
		JobID    string `json:"job_id"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if dec.Decode(&input) != nil || input.Provider != "" && input.Provider != "railway" && input.Provider != "runpod" {
		writeError(w, r, 400, "invalid_claim", "Invalid OCR provider")
		return
	}
	if input.Provider == "" {
		input.Provider = "railway"
	}
	if input.JobID != "" {
		var id pgtype.UUID
		if len(input.JobID) != 36 || id.Scan(input.JobID) != nil {
			writeError(w, r, 400, "invalid_claim", "Invalid OCR job ID")
			return
		}
	}
	orgs := make([]string, 0, len(s.config.OCRRunPodOrganizations))
	for organizationID := range s.config.OCRRunPodOrganizations {
		orgs = append(orgs, organizationID)
	}
	jobs, err := s.config.OCRJobs.ClaimJobs(r.Context(), job.ClaimCommand{WorkerID: id.WorkerID, Environment: id.Environment, Kinds: []job.Kind{job.OCR}, Limit: 1, Lease: 2 * time.Minute, Now: s.config.Now(), OCRProvider: input.Provider, OCRDefaultProvider: s.config.OCRDefaultProvider, OCRRunPodOrganizations: orgs, OCRJobID: input.JobID})
	if err != nil {
		s.writeJobError(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"jobs": jobs})
}
func (s *server) ocrInputFor(w http.ResponseWriter, r *http.Request, scope string) (ocr.Input, job.WorkerIdentity, bool) {
	id, ok := s.authorizeOCR(w, r, scope)
	if !ok {
		return ocr.Input{}, id, false
	}
	in, err := s.config.OCR.OCRInput(r.Context(), s.leaseCommand(r), id.WorkerID)
	if err != nil {
		s.writeJobError(w, r, err)
		return in, id, false
	}
	return in, id, true
}
func (s *server) heartbeatOCR(w http.ResponseWriter, r *http.Request) {
	_, _, ok := s.ocrInputFor(w, r, "heartbeat")
	if !ok {
		return
	}
	expiry, err := s.config.OCRJobs.HeartbeatJob(r.Context(), s.leaseCommand(r), 2*time.Minute)
	if err != nil {
		s.writeJobError(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"lease_expires_at": expiry})
}
func (s *server) failOCR(w http.ResponseWriter, r *http.Request) {
	_, _, ok := s.ocrInputFor(w, r, "fail")
	if !ok {
		return
	}
	var input struct {
		Code string `json:"code"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if dec.Decode(&input) != nil || !validFailureCode(input.Code) {
		writeError(w, r, 400, "invalid_failure", "Invalid error code")
		return
	}
	if err := s.config.OCRJobs.FailJob(r.Context(), job.FailureCommand{LeaseCommand: s.leaseCommand(r), Code: input.Code}); err != nil {
		s.writeJobError(w, r, err)
		return
	}
	w.WriteHeader(204)
}
func (s *server) inputOCR(w http.ResponseWriter, r *http.Request) {
	in, id, ok := s.ocrInputFor(w, r, "input")
	if !ok {
		return
	}
	token, err := s.config.OCRTokens.SignTTL(r.PathValue("job"), in.OrganizationID, id.WorkerID, s.config.Now(), 5*time.Minute)
	if err != nil {
		s.writeJobError(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"sha256": in.SHA256, "mime": in.MIME, "size": in.Size, "download_url": "/internal/v1/ocr/jobs/" + r.PathValue("job") + "/original?token=" + token})
}
func (s *server) originalOCR(w http.ResponseWriter, r *http.Request) {
	in, id, ok := s.ocrInputFor(w, r, "input")
	if !ok {
		return
	}
	jid, org, worker, err := s.config.OCRTokens.Verify(r.URL.Query().Get("token"), s.config.Now())
	if err != nil || jid != r.PathValue("job") || org != in.OrganizationID || worker != id.WorkerID {
		writeError(w, r, 401, "unauthorized", "Input access denied")
		return
	}
	body, err := s.config.OCRStorage.Open(r.Context(), in.StorageKey)
	if err != nil {
		s.writeJobError(w, r, err)
		return
	}
	defer body.Close()
	w.Header().Set("Content-Type", in.MIME)
	w.Header().Set("Cache-Control", "private, no-store")
	_, _ = io.Copy(w, io.LimitReader(body, in.Size))
}
func (s *server) submitOCR(w http.ResponseWriter, r *http.Request) {
	in, id, ok := s.ocrInputFor(w, r, "submit")
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, ocr.MaxResultBytes)
	body, err := io.ReadAll(r.Body)
	var result ocr.Result
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err != nil || dec.Decode(&result) != nil || result.Validate(in) != nil {
		writeError(w, r, 422, "invalid_result", "Invalid OCR result")
		return
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		writeError(w, r, 422, "invalid_result", "Invalid OCR result")
		return
	}
	var suffix [16]byte
	if _, err = rand.Read(suffix[:]); err != nil {
		s.writeJobError(w, r, err)
		return
	}
	key := "ocr/" + in.OrganizationID + "/" + in.DocumentID + "/" + hex.EncodeToString(suffix[:]) + ".json"
	keep := false
	defer func() {
		if !keep {
			ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 10*time.Second)
			defer cancel()
			_ = s.config.OCRStorage.Delete(ctx, key)
		}
	}()
	if err = s.config.OCRStorage.Put(r.Context(), key, bytes.NewReader(body)); err != nil {
		s.writeJobError(w, r, err)
		return
	}
	hash := sha256.Sum256(body)
	saved, err := s.config.OCR.CompleteOCR(r.Context(), s.leaseCommand(r), id.WorkerID, result, key, hex.EncodeToString(hash[:]), int64(len(body)))
	if err != nil {
		s.writeJobError(w, r, err)
		return
	}
	keep = saved == key
	if keep && s.classificationAllowed(in.OrganizationID) && s.config.Classification != nil {
		started := time.Now()
		llmCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		prediction, llmUsed, classifyErr := classification.Hybrid(llmCtx, classification.Normalize(in.DocumentID, result),
			s.config.Classifier, s.config.ClassificationRuleThreshold, s.config.ClassificationReviewThreshold)
		cancel()
		if classifyErr != nil {
			s.config.Logger.Warn("classification_fallback", "document_id", in.DocumentID, "error", classifyErr)
		}
		if err = s.config.Classification.SaveClassification(r.Context(), in.OrganizationID, in.DocumentID, r.PathValue("job"), prediction); err != nil {
			s.config.Logger.Error("classification_save_failed", "document_id", in.DocumentID, "error", err)
		}
		s.config.Logger.Info("document_classified", "document_id", in.DocumentID, "ocr_duration_ms", result.DurationMS,
			"classification_duration_ms", time.Since(started).Milliseconds(), "classification_method", prediction.Method,
			"document_type", prediction.DocumentType, "confidence", prediction.Confidence, "llm_used", llmUsed,
			"classifier_model", prediction.ClassifierVersion, "requires_review", prediction.RequiresReview)
	}
	writeJSON(w, 200, map[string]string{"status": "Completed"})
}
func (s *server) stateOCR(w http.ResponseWriter, r *http.Request) {
	session, m, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	state, err := s.config.OCR.OCRState(r.Context(), session.UserID, m.OrganizationID, r.PathValue("document"))
	if err != nil {
		writeError(w, r, 404, "not_found", "Document unavailable")
		return
	}
	url := ""
	processingStatus := map[string]string{"NotScheduled": "uploaded", "Queued": "uploaded", "Running": "ocr_processing", "Completed": "ocr_completed", "Failed": "failed", "Cancelled": "failed"}[state.Status]
	if state.Status == "Completed" && s.classificationAllowed(m.OrganizationID) && s.config.Classification != nil {
		processingStatus = "classifying"
		if record, classErr := s.config.Classification.GetClassification(r.Context(), session.UserID, m.OrganizationID, r.PathValue("document")); classErr == nil && record.EffectiveType != "" {
			processingStatus = "ready"
			if record.RequiresReview {
				processingStatus = "review_required"
			}
		}
	}
	if state.Status == "Completed" && state.ObjectKey != "" {
		token, e := s.config.OCRTokens.SignTTL(r.PathValue("document"), m.OrganizationID, session.UserID, s.config.Now(), 5*time.Minute)
		if e != nil {
			s.writeJobError(w, r, e)
			return
		}
		url = "/v1/ocr-results/" + r.PathValue("document") + "?token=" + token
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, 200, map[string]any{"ocr": state, "download_url": url, "processing_status": processingStatus})
}
func (s *server) retryOCR(w http.ResponseWriter, r *http.Request) {
	session, m, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if m.Role != tenant.Owner && m.Role != tenant.Admin {
		writeError(w, r, 403, "forbidden", "Owner or Admin required")
		return
	}
	state, err := s.config.OCR.RetryOCR(r.Context(), session.UserID, m.OrganizationID, r.PathValue("document"), s.config.Now())
	if err != nil {
		writeError(w, r, 409, "ocr_unavailable", "OCR cannot be scheduled")
		return
	}
	writeJSON(w, 200, map[string]any{"ocr": state})
}
func (s *server) retrieveOCR(w http.ResponseWriter, r *http.Request) {
	doc, org, user, err := s.config.OCRTokens.Verify(r.URL.Query().Get("token"), s.config.Now())
	if err != nil || doc != r.PathValue("document") {
		http.NotFound(w, r)
		return
	}
	state, err := s.config.OCR.OCRState(r.Context(), user, org, doc)
	if err != nil || state.Status != "Completed" || state.ObjectKey == "" {
		http.NotFound(w, r)
		return
	}
	body, err := s.config.OCRStorage.Open(r.Context(), state.ObjectKey)
	if err != nil {
		s.writeJobError(w, r, err)
		return
	}
	defer body.Close()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = io.Copy(w, io.LimitReader(body, ocr.MaxResultBytes))
}
