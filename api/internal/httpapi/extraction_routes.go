package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"plaiflow/api/internal/document"
	"plaiflow/api/internal/extraction"
	"plaiflow/api/internal/ocr"
	"plaiflow/api/internal/plan"
	"plaiflow/api/internal/tenant"
)

const maxReviewBytes = 32 << 10

var structuredHeader = []string{"document_id", "document_type", "issue_date", "document_number", "seller_name", "seller_tax_id", "buyer_name", "buyer_tax_id", "currency", "subtotal", "vat_amount", "total_amount", "confirmed_at", "confirmed_by", "extraction_schema_version"}

func (s *server) registerExtractionRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /v1/o/{organization}/documents/{document}/extraction", s.getExtraction)
	m.HandleFunc("POST /v1/o/{organization}/documents/{document}/extraction/confirm", s.confirmExtraction)
	m.HandleFunc("GET /v1/o/{organization}/documents/extraction.csv", s.exportExtractions)
	m.HandleFunc("GET /v1/o/{organization}/documents/extraction.xlsx", s.exportExtractions)
}

func (s *server) extractionDraft(ctx context.Context, user, org, doc string) (extraction.Draft, string, error) {
	state, err := s.config.OCR.OCRState(ctx, user, org, doc)
	if err != nil || state.Status != "Completed" || state.ObjectKey == "" {
		return extraction.Draft{}, "", errors.New("completed OCR is unavailable")
	}
	body, err := s.config.OCRStorage.Open(ctx, state.ObjectKey)
	if err != nil {
		return extraction.Draft{}, "", err
	}
	defer body.Close()
	data, err := io.ReadAll(io.LimitReader(body, ocr.MaxResultBytes+1))
	if err != nil || len(data) > ocr.MaxResultBytes {
		return extraction.Draft{}, "", errors.New("OCR result exceeds limit")
	}
	var result ocr.Result
	if err := json.Unmarshal(data, &result); err != nil || result.SchemaVersion != 1 || len(result.Pages) == 0 || len(result.Pages) > 20 {
		return extraction.Draft{}, "", errors.New("OCR result is invalid")
	}
	return extraction.Extract(result), state.JobID, nil
}

func (s *server) getExtraction(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	started := time.Now()
	draft, ocrJob, err := s.extractionDraft(r.Context(), session.UserID, membership.OrganizationID, r.PathValue("document"))
	if err != nil {
		writeError(w, r, 409, "ocr_unavailable", "Complete OCR is required before extraction")
		return
	}
	current, err := s.config.Extraction.CurrentReview(r.Context(), session.UserID, membership.OrganizationID, r.PathValue("document"))
	if err != nil {
		writeError(w, r, 503, "extraction_unavailable", "Review is unavailable")
		return
	}
	var confirmed *extraction.Review
	if current.ID != "" && current.OCRJobID == ocrJob {
		confirmed, err = s.readReview(r.Context(), current)
		if err != nil {
			writeError(w, r, 503, "extraction_unavailable", "Review is unavailable")
			return
		}
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, 200, map[string]any{"draft": draft, "ocr_job_id": ocrJob, "revision": current.Revision, "confirmed": confirmed,
		"source_superseded": current.ID != "" && current.OCRJobID != ocrJob})
	s.config.Logger.Info("extraction_generated", "duration_ms", time.Since(started).Milliseconds(), "fields", len(draft.Fields), "provider_cost_usd", 0,
		"compute_cost_status", "not_metered")
}

func (s *server) confirmExtraction(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, 403, "forbidden", "Owner or Admin required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxReviewBytes)
	if err := r.ParseForm(); err != nil {
		writeError(w, r, 400, "invalid_review", "Review is invalid")
		return
	}
	if r.PostForm.Get("review_ack") != "1" {
		writeError(w, r, 422, "review_required", "Compare the values with the original before confirming")
		return
	}
	revision, err := strconv.Atoi(r.PostForm.Get("expected_revision"))
	if err != nil || revision < 0 {
		writeError(w, r, 400, "invalid_review", "Review revision is invalid")
		return
	}
	draft, ocrJob, err := s.extractionDraft(r.Context(), session.UserID, membership.OrganizationID, r.PathValue("document"))
	if err != nil || ocrJob != r.PostForm.Get("ocr_job_id") {
		writeError(w, r, 409, "ocr_changed", "OCR result changed; review again")
		return
	}
	values := make(map[string]string, len(extraction.Keys))
	for _, key := range extraction.Keys {
		values[key] = strings.TrimSpace(r.PostForm.Get(key))
	}
	if err := extraction.ValidateReview(draft, values); err != nil {
		writeError(w, r, 422, "invalid_review", "Review required fields and amounts")
		return
	}
	id := newUUID()
	if id == "" {
		writeError(w, r, 503, "extraction_unavailable", "Review is unavailable")
		return
	}
	review := extraction.Review{ID: id, OrganizationID: membership.OrganizationID, DocumentID: r.PathValue("document"), OCRJobID: ocrJob,
		DocumentType: draft.DocumentType, Values: values, ConfirmedBy: session.UserID, ConfirmedAt: s.config.Now().UTC(),
		ObjectKey: "extraction/" + membership.OrganizationID + "/" + id + ".json"}
	data, err := json.Marshal(review)
	if err != nil || len(data) > maxReviewBytes {
		writeError(w, r, 422, "invalid_review", "Review exceeds limit")
		return
	}
	if err = s.config.OCRStorage.Put(r.Context(), review.ObjectKey, bytes.NewReader(data)); err != nil {
		writeError(w, r, 503, "extraction_unavailable", "Review storage is unavailable")
		return
	}
	keep := false
	defer func() {
		if !keep {
			ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 10*time.Second)
			defer cancel()
			_ = s.config.OCRStorage.Delete(ctx, review.ObjectKey)
		}
	}()
	review, err = s.config.Extraction.SaveReview(r.Context(), review, revision)
	if errors.Is(err, extraction.ErrConflict) {
		writeError(w, r, 409, "review_changed", "Review changed; reload before confirming")
		return
	}
	if err != nil {
		writeError(w, r, 503, "extraction_unavailable", "Review could not be saved")
		return
	}
	keep = true
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, 200, map[string]any{"status": "Confirmed", "revision": review.Revision})
}

func (s *server) readReview(ctx context.Context, meta extraction.Review) (*extraction.Review, error) {
	body, err := s.config.OCRStorage.Open(ctx, meta.ObjectKey)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	data, err := io.ReadAll(io.LimitReader(body, maxReviewBytes+1))
	if err != nil || len(data) > maxReviewBytes {
		return nil, errors.New("review artifact exceeds limit")
	}
	var review extraction.Review
	if err := json.Unmarshal(data, &review); err != nil || review.ID != meta.ID || review.OrganizationID != meta.OrganizationID ||
		review.DocumentID != meta.DocumentID || review.OCRJobID != meta.OCRJobID || len(review.Values) > len(extraction.Keys) {
		return nil, errors.New("review artifact is invalid")
	}
	review.Revision, review.ConfirmedBy, review.ConfirmedAt = meta.Revision, meta.ConfirmedBy, meta.ConfirmedAt
	return &review, nil
}

func (s *server) exportExtractions(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, 403, "forbidden", "Owner or Admin required")
		return
	}
	format := "csv"
	if strings.HasSuffix(r.URL.Path, ".xlsx") {
		format = "xlsx"
	}
	capability := plan.ExportCSV
	if format == "xlsx" {
		capability = plan.ExportXLSX
	}
	decision, err := s.config.Gate.Check(r.Context(), membership.OrganizationID, capability, 0)
	if err != nil || !decision.Allowed {
		writeError(w, r, 403, "feature_unavailable", "Export is unavailable for this plan")
		return
	}
	// ponytail: synchronous private-blob reads are capped at 100 rows; use a durable export job before raising this limit.
	metas, err := s.config.Extraction.ListCurrentReviews(r.Context(), session.UserID, membership.OrganizationID, 100)
	if err != nil {
		writeError(w, r, 503, "extraction_export_unavailable", "Export is unavailable")
		return
	}
	rows := make([][]string, 0, len(metas))
	for _, meta := range metas {
		review, err := s.readReview(r.Context(), meta)
		if err != nil {
			writeError(w, r, 503, "extraction_export_unavailable", "Export is unavailable")
			return
		}
		v := review.Values
		rows = append(rows, []string{review.DocumentID, review.DocumentType, v["issue_date"], v["document_number"], v["seller_name"],
			v["seller_tax_id"], v["buyer_name"], v["buyer_tax_id"], v["currency"], v["subtotal"], v["vat_amount"],
			v["total_amount"], review.ConfirmedAt.UTC().Format(time.RFC3339), review.ConfirmedBy, strconv.Itoa(extraction.SchemaVersion)})
	}
	var output bytes.Buffer
	if err := document.WriteTable(&output, format, structuredHeader, rows); err != nil || output.Len() > 128<<20 {
		writeError(w, r, 503, "extraction_export_unavailable", "Export is unavailable")
		return
	}
	w.Header().Set("Content-Type", map[string]string{"csv": "text/csv; charset=utf-8", "xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"}[format])
	w.Header().Set("Content-Disposition", `attachment; filename="structured-documents-v1.`+format+`"`)
	w.Header().Set("Cache-Control", "private, no-store")
	_, _ = w.Write(output.Bytes())
}
