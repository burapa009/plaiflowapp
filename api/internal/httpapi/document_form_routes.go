package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"plaiflow/api/internal/extraction"
	"plaiflow/api/internal/tenant"
	"time"
)

func (s *server) getDocumentForm(w http.ResponseWriter, r *http.Request) {
	session, member, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	store, ok := s.config.Extraction.(extraction.FormStore)
	if !ok {
		writeError(w, r, 503, "form_unavailable", "Form storage unavailable")
		return
	}
	draft, job, err := s.extractionDraft(r.Context(), session.UserID, member.OrganizationID, r.PathValue("document"))
	if err != nil {
		writeError(w, r, 409, "ocr_unavailable", "Completed OCR required")
		return
	}
	record, err := store.GetDocumentForm(r.Context(), session.UserID, member.OrganizationID, r.PathValue("document"))
	if err != nil {
		writeError(w, r, 503, "form_unavailable", "Form schema unavailable")
		return
	}
	current, err := s.config.Extraction.CurrentReview(r.Context(), session.UserID, member.OrganizationID, r.PathValue("document"))
	if err != nil {
		writeError(w, r, 503, "form_unavailable", "Review unavailable")
		return
	}
	legacyDraftRevision := 0
	superseded := record.Revision > 0 && record.OCRJobID != job
	if record.Revision == 0 {
		record.Data = extraction.FormFromDraft(draft)
		record.Status = "Draft"
		if current.ID != "" && current.OCRJobID == job {
			previous, e := s.readReview(r.Context(), current)
			if e != nil {
				writeError(w, r, 503, "form_unavailable", "Review unavailable")
				return
			}
			for key, value := range previous.Values {
				record.Data.Fields[key] = value
			}
			record.Data.Type = previous.DocumentType
			if record.Data.Type == "receipt_tax_invoice" {
				record.Data.Type = "tax_invoice_receipt"
			}
			if _, ok := extraction.DocumentForms.Types[record.Data.Type]; !ok {
				record.Data.Type = "unknown"
			}
		}
		// Preserve historical human work even if member reviewing is now disabled.
		legacy, e := s.config.Extraction.CurrentDraft(r.Context(), session.UserID, member.OrganizationID, r.PathValue("document"))
		if e != nil {
			writeError(w, r, 503, "form_unavailable", "Saved review unavailable")
			return
		}
		if legacy.Revision > 0 && legacy.OCRJobID == job {
			legacyDraftRevision = legacy.Revision
			for key, value := range legacy.Values {
				record.Data.Fields[key] = value
			}
		}
	}
	if record.Data.Source == nil {
		source := extraction.FormFromDraft(draft)
		record.Data.Source = &extraction.FormSource{OCRJobID: job, Fields: source.Fields, Tables: source.Tables}
	}
	// Preserve saved data when OCR changes; expected OCR and both revisions must be explicit on save.
	record.OCRJobID = job
	references, err := store.ListFormReferences(r.Context(), session.UserID, member.OrganizationID)
	if err != nil {
		writeError(w, r, 503, "form_unavailable", "References unavailable")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, 200, map[string]any{"form": record, "references": references, "review_revision": current.Revision, "legacy_draft_revision": legacyDraftRevision, "source_superseded": superseded,
		"can_edit":    member.Role == tenant.Owner || member.Role == tenant.Admin || s.config.ReviewEnabled,
		"can_confirm": member.Role == tenant.Owner || member.Role == tenant.Admin})
}

func (s *server) saveDocumentForm(w http.ResponseWriter, r *http.Request) {
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		writeError(w, r, http.StatusUnsupportedMediaType, "invalid_content_type", "Request content type is invalid")
		return
	}
	session, ok := s.authenticatedMutation(w, r)
	if !ok {
		return
	}
	member, err := s.config.Tenants.ResolveMembership(r.Context(), session.UserID, r.PathValue("organization"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Resource was not found")
		return
	}
	store, ok := s.config.Extraction.(extraction.FormStore)
	if !ok {
		writeError(w, r, 503, "form_unavailable", "Form storage unavailable")
		return
	}
	var input struct {
		Data                        extraction.DocumentForm `json:"data"`
		Action                      string                  `json:"action"`
		OCRJobID                    string                  `json:"ocr_job_id"`
		ExpectedRevision            int                     `json:"expected_revision"`
		ExpectedReviewRevision      int                     `json:"expected_review_revision"`
		ExpectedLegacyDraftRevision int                     `json:"expected_legacy_draft_revision"`
		Confirm                     bool                    `json:"confirm"`
		Acknowledged                bool                    `json:"acknowledged"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(w, r, 400, "invalid_form", "Invalid form")
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeError(w, r, 400, "invalid_form", "Invalid form")
		return
	}
	manager := member.Role == tenant.Owner || member.Role == tenant.Admin
	if input.Action != "" && input.Action != "draft" && input.Action != "confirm" {
		writeJSON(w, 422, map[string]any{"code": "unsupported_action", "fields": extraction.ValidateFormAction(input.Data, input.Action)})
		return
	}
	if input.Action != "" {
		input.Confirm = input.Action == "confirm"
	}
	if !manager && (input.Confirm || !s.config.ReviewEnabled) {
		writeError(w, r, 403, "forbidden", "Insufficient role")
		return
	}
	if input.ExpectedRevision < 0 || input.ExpectedReviewRevision < 0 || input.ExpectedLegacyDraftRevision < 0 {
		writeError(w, r, 400, "invalid_form", "Invalid revision")
		return
	}
	if input.Confirm && !input.Acknowledged {
		writeJSON(w, 422, map[string]any{"code": "review_required", "fields": map[string]string{"review_ack": "ยืนยันว่าตรวจข้อมูลเทียบต้นฉบับแล้ว"}})
		return
	}
	previous, err := store.GetDocumentForm(r.Context(), session.UserID, member.OrganizationID, r.PathValue("document"))
	if err != nil {
		writeError(w, r, 503, "form_unavailable", "Form unavailable")
		return
	}
	if previous.Revision != input.ExpectedRevision {
		writeError(w, r, 409, "form_changed", "Reload before saving")
		return
	}
	input.Data = extraction.MergeDocumentForm(previous.Data, input.Data)
	if errs := extraction.ValidateDocumentForm(input.Data, input.Confirm); len(errs) > 0 {
		writeJSON(w, 422, map[string]any{"code": "invalid_form", "fields": errs})
		return
	}
	draft, job, err := s.extractionDraft(r.Context(), session.UserID, member.OrganizationID, r.PathValue("document"))
	if err != nil || job != input.OCRJobID {
		writeError(w, r, 409, "ocr_changed", "Reload OCR")
		return
	}
	// Only server-loaded evidence can populate the immutable source snapshot.
	input.Data.Source = previous.Data.Source
	if input.Data.Source == nil || input.Data.Source.OCRJobID != job {
		source := extraction.FormFromDraft(draft)
		input.Data.Source = &extraction.FormSource{OCRJobID: job, Fields: source.Fields, Tables: source.Tables}
	}
	action := "draft"
	if input.Confirm {
		action = "confirm"
	}
	input.Data.Assessment = extraction.AssessDocumentForm(input.Data, action)
	var record extraction.FormRecord
	if !input.Confirm {
		record, err = store.SaveDocumentForm(r.Context(), session.UserID, member.OrganizationID, r.PathValue("document"), job, input.Data, input.ExpectedRevision, input.ExpectedLegacyDraftRevision, requestID(r), s.config.Now().UTC())
	} else {
		if !s.originalAvailable(r.Context(), session.UserID, member.OrganizationID, r.PathValue("document")) {
			writeError(w, r, 409, "original_unavailable", "Original unavailable")
			return
		}
		id := newUUID()
		if id == "" {
			writeError(w, r, 503, "form_unavailable", "ID unavailable")
			return
		}
		review := extraction.Review{ID: id, OrganizationID: member.OrganizationID, DocumentID: r.PathValue("document"), OCRJobID: job,
			Canonical: &input.Data, ExpectedFormRevision: input.ExpectedRevision, ExpectedLegacyDraftRevision: input.ExpectedLegacyDraftRevision, DocumentType: input.Data.Type, Values: input.Data.LegacyValues(),
			ConfirmedBy: session.UserID, ConfirmedAt: s.config.Now().UTC(), RequestID: requestID(r), ObjectKey: "extraction/" + member.OrganizationID + "/" + id + ".json"}
		raw, e := json.Marshal(review)
		if e != nil || len(raw) > 256<<10 {
			writeError(w, r, 422, "invalid_form", "Form too large")
			return
		}
		if e = s.config.OCRStorage.Put(r.Context(), review.ObjectKey, bytes.NewReader(raw)); e != nil {
			writeError(w, r, 503, "form_unavailable", "Storage unavailable")
			return
		}
		_, err = s.config.Extraction.SaveReview(r.Context(), review, input.ExpectedReviewRevision)
		if err != nil {
			ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 10*time.Second)
			defer cancel()
			_ = s.config.OCRStorage.Delete(ctx, review.ObjectKey)
		}
		record = extraction.FormRecord{Data: input.Data, Revision: input.ExpectedRevision + 1, OCRJobID: job, Status: "Confirmed", UpdatedAt: review.ConfirmedAt}
	}
	if errors.Is(err, extraction.ErrConflict) {
		writeError(w, r, 409, "form_changed", "Reload before saving")
		return
	}
	var fields extraction.FormErrors
	if errors.As(err, &fields) {
		writeJSON(w, 422, map[string]any{"code": "invalid_form", "fields": fields})
		return
	}
	if err != nil {
		writeError(w, r, 503, "form_unavailable", "Form could not be saved")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, 200, record)
}
