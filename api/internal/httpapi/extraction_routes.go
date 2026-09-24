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
	var saved *extraction.ReviewDraft
	var returned *extraction.ReturnNotice
	var history []extraction.ReviewDraft
	if s.config.ReviewEnabled {
		item, draftErr := s.config.Extraction.CurrentDraft(r.Context(), session.UserID, membership.OrganizationID, r.PathValue("document"))
		if draftErr != nil {
			writeError(w, r, 503, "review_unavailable", "Saved review is unavailable")
			return
		}
		if item.Revision > 0 && item.OCRJobID == ocrJob {
			saved = &item
		}
		history, err = s.config.Extraction.ListDraftHistory(r.Context(), session.UserID, membership.OrganizationID, r.PathValue("document"), 20)
		if err != nil {
			writeError(w, r, 503, "review_unavailable", "Review history is unavailable")
			return
		}
		notice, noticeErr := s.config.Extraction.CurrentReturn(r.Context(), session.UserID, membership.OrganizationID, r.PathValue("document"))
		if noticeErr != nil {
			writeError(w, r, 503, "review_unavailable", "Return status is unavailable")
			return
		}
		if notice.ReasonCode != "" {
			returned = &notice
		}
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, 200, map[string]any{"draft": draft, "ocr_job_id": ocrJob, "revision": current.Revision, "confirmed": confirmed,
		"review_enabled": s.config.ReviewEnabled,
		"saved_review":   saved, "review_history": history, "returned_review": returned,
		"source_superseded": current.ID != "" && current.OCRJobID != ocrJob})
	s.config.Logger.Info("extraction_generated", "duration_ms", time.Since(started).Milliseconds(), "fields", len(draft.Fields), "provider_cost_usd", 0,
		"compute_cost_status", "not_metered")
}

func (s *server) saveExtractionDraft(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxReviewBytes)
	if err := r.ParseForm(); err != nil {
		writeError(w, r, 400, "invalid_review", "Review draft is invalid")
		return
	}
	expected, err := strconv.Atoi(r.PostForm.Get("expected_draft_revision"))
	if err != nil || expected < 0 {
		writeError(w, r, 400, "invalid_review", "Review draft revision is invalid")
		return
	}
	proposal, ocrJob, err := s.extractionDraft(r.Context(), session.UserID, membership.OrganizationID, r.PathValue("document"))
	if err != nil || ocrJob != r.PostForm.Get("ocr_job_id") {
		writeError(w, r, 409, "ocr_changed", "OCR result changed; review again")
		return
	}
	draft := extraction.ReviewDraft{OrganizationID: membership.OrganizationID, DocumentID: r.PathValue("document"),
		OCRJobID: ocrJob, Values: make(map[string]string, len(extraction.Keys)), Decisions: make(map[string]string, len(extraction.Keys)),
		UpdatedBy: session.UserID, UpdatedAt: s.config.Now().UTC()}
	for _, key := range extraction.Keys {
		value := strings.TrimSpace(r.PostForm.Get(key))
		decision := r.PostForm.Get(key + "_decision")
		if len([]rune(value)) > 240 || decision != "" && decision != "accepted" && decision != "corrected" && decision != "unknown" ||
			decision == "unknown" && value != "" || decision == "accepted" && value != proposal.Fields[key].Normalized {
			writeError(w, r, 422, "invalid_review", "Review field is invalid")
			return
		}
		draft.Values[key], draft.Decisions[key] = value, decision
	}
	draft, err = s.config.Extraction.SaveDraft(r.Context(), draft, expected, requestID(r))
	if errors.Is(err, extraction.ErrConflict) {
		writeError(w, r, 409, "review_changed", "Review changed; reload before saving")
		return
	}
	if err != nil {
		writeError(w, r, 503, "review_unavailable", "Review draft could not be saved")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, 200, draft)
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
	if s.config.ReviewEnabled {
		saved, draftErr := s.config.Extraction.CurrentDraft(r.Context(), session.UserID, membership.OrganizationID, r.PathValue("document"))
		draftRevision, parseErr := strconv.Atoi(r.PostForm.Get("expected_draft_revision"))
		if draftErr != nil || parseErr != nil || saved.Revision == 0 || saved.Revision != draftRevision || saved.OCRJobID != ocrJob {
			writeError(w, r, 409, "review_changed", "Saved review changed; reload before confirming")
			return
		}
		for _, key := range extraction.Keys {
			if saved.Decisions[key] != "accepted" && saved.Decisions[key] != "corrected" && saved.Decisions[key] != "unknown" {
				writeError(w, r, 422, "review_required", "Decide every field before confirming")
				return
			}
			if strings.TrimSpace(r.PostForm.Get(key)) != saved.Values[key] || r.PostForm.Get(key+"_decision") != saved.Decisions[key] {
				writeError(w, r, 409, "unsaved_changes", "Save changes before confirming")
				return
			}
			values[key] = saved.Values[key]
		}
		if !s.originalAvailable(r.Context(), session.UserID, membership.OrganizationID, r.PathValue("document")) {
			writeError(w, r, 409, "original_unavailable", "Open the original before confirming")
			return
		}
	} else {
		for _, key := range extraction.Keys {
			values[key] = strings.TrimSpace(r.PostForm.Get(key))
		}
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
	if s.config.ReviewEnabled {
		review.DraftRevision, _ = strconv.Atoi(r.PostForm.Get("expected_draft_revision"))
	}
	review.RequestID = requestID(r)
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

func (s *server) originalAvailable(ctx context.Context, user, org, doc string) bool {
	if s.config.Documents == nil {
		return false
	}
	_, original, err := s.config.Documents.Open(ctx, user, org, doc)
	if err != nil {
		return false
	}
	defer original.Close()
	var first [1]byte
	_, err = io.ReadFull(original, first[:])
	return err == nil
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
	var metas []extraction.Review
	if s.config.ReviewEnabled {
		metas, err = s.config.Extraction.ListCurrentReviewsFiltered(r.Context(), session.UserID, membership.OrganizationID, 100, "Available")
	} else {
		metas, err = s.config.Extraction.ListCurrentReviews(r.Context(), session.UserID, membership.OrganizationID, 100)
	}
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
		rows = append(rows, structuredExportValues(*review))
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
