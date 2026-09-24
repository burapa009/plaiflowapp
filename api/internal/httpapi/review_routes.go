package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"plaiflow/api/internal/document"
	"plaiflow/api/internal/extraction"
	"plaiflow/api/internal/tenant"
	"plaiflow/api/internal/work"
)

func (s *server) reviewQueue(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	limit, valid := pageLimit(r.URL.Query().Get("limit"))
	if !valid || limit > 50 {
		writeError(w, r, 400, "invalid_queue", "Queue limit is invalid")
		return
	}
	view := r.URL.Query().Get("view")
	if view == "" {
		view = "all"
	}
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "Available"
	}
	page, err := s.config.Extraction.ListReviewQueue(r.Context(), session.UserID, membership.OrganizationID,
		r.URL.Query().Get("cursor"), view, status, limit)
	if errors.Is(err, document.ErrInvalidCursor) {
		writeError(w, r, 400, "invalid_queue", "Queue filter or cursor is invalid")
		return
	}
	if err != nil {
		writeError(w, r, 503, "review_unavailable", "Review queue is unavailable")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, 200, page)
}

func (s *server) returnReview(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, r, 400, "invalid_return", "Return request is invalid")
		return
	}
	reviewRevision, e1 := strconv.Atoi(r.PostForm.Get("review_revision"))
	draftRevision, e2 := strconv.Atoi(r.PostForm.Get("draft_revision"))
	if e1 != nil || e2 != nil || reviewRevision < 0 || draftRevision < 0 {
		writeError(w, r, 400, "invalid_return", "Review revision is invalid")
		return
	}
	input := extraction.ReturnInput{ID: newUUID(), ActorID: session.UserID, OrganizationID: membership.OrganizationID,
		DocumentID: r.PathValue("document"), OCRJobID: r.PostForm.Get("ocr_job_id"),
		ExpectedReviewRevision: reviewRevision, ExpectedDraftRevision: draftRevision,
		ReasonCode: r.PostForm.Get("reason_code"), PrivateNote: strings.TrimSpace(r.PostForm.Get("private_note")),
		RequestID: requestID(r), ReturnedAt: s.config.Now().UTC()}
	revision, err := s.config.Extraction.ReturnReview(r.Context(), input)
	if errors.Is(err, extraction.ErrConflict) {
		writeError(w, r, 409, "review_changed", "Review changed; reload")
		return
	}
	if err != nil {
		writeError(w, r, 503, "review_unavailable", "Review could not be returned")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, 200, map[string]any{"status": "Returned", "draft_revision": revision})
}

func (s *server) reprocessReview(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, r, 400, "invalid_reprocess", "Reprocess request is invalid")
		return
	}
	draftRevision, err := strconv.Atoi(r.PostForm.Get("draft_revision"))
	if err != nil || draftRevision < 0 {
		writeError(w, r, 400, "invalid_reprocess", "Review revision is invalid")
		return
	}
	state, err := s.config.OCR.ReprocessOCR(r.Context(), session.UserID, membership.OrganizationID, r.PathValue("document"),
		r.PostForm.Get("ocr_job_id"), draftRevision,
		r.PostForm.Get("reason_code"), strings.TrimSpace(r.PostForm.Get("private_note")), requestID(r), s.config.Now().UTC())
	if err != nil {
		writeError(w, r, 409, "reprocess_unavailable", "Reprocess is unavailable; reload")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, 202, map[string]any{"ocr": state})
}

func (s *server) assignReviewQueue(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, r, 400, "invalid_assignment", "Assignment is invalid")
		return
	}
	docs, ocrs := r.PostForm["document_id"], r.PostForm["ocr_job_id"]
	reviews, drafts := r.PostForm["review_revision"], r.PostForm["draft_revision"]
	if len(docs) == 0 || len(docs) > 50 || len(ocrs) != len(docs) || len(reviews) != len(docs) || len(drafts) != len(docs) {
		writeError(w, r, 400, "invalid_assignment", "Select 1 to 50 current review items")
		return
	}
	items := make([]extraction.Assignment, len(docs))
	for i := range docs {
		review, e1 := strconv.Atoi(reviews[i])
		draft, e2 := strconv.Atoi(drafts[i])
		if e1 != nil || e2 != nil || review < 0 || draft < 0 {
			writeError(w, r, 400, "invalid_assignment", "Assignment revision is invalid")
			return
		}
		items[i] = extraction.Assignment{DocumentID: docs[i], OCRJobID: ocrs[i], ReviewRevision: review,
			DraftRevision: draft, TaskID: newUUID()}
	}
	decision, err := s.config.Gate.Check(r.Context(), membership.OrganizationID, work.CreateTasks, int64(len(items)))
	if err != nil || !decision.Allowed {
		writeError(w, r, 402, "feature_unavailable", "Task assignment is unavailable for this plan")
		return
	}
	created, err := s.config.Extraction.AssignReviewTasks(r.Context(), session.UserID, membership.OrganizationID,
		r.PostForm.Get("assignee_user_id"), items, requestID(r), s.config.Now().UTC())
	if errors.Is(err, extraction.ErrConflict) {
		writeError(w, r, 409, "review_changed", "Review item or assignee changed; reload")
		return
	}
	if err != nil {
		writeError(w, r, 503, "review_unavailable", "Assignment could not be saved")
		return
	}
	for _, taskID := range created {
		if recordErr := s.config.Gate.Record(r.Context(), membership.OrganizationID, work.CreateTasks, 1, taskID); recordErr != nil {
			s.config.Logger.Warn("usage_record_failed", "request_id", requestID(r), "capability", work.CreateTasks)
		}
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, 200, map[string]any{"assigned": len(items)})
}
