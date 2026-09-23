package httpapi

import (
	"bytes"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"plaiflow/api/internal/accounting"
	"plaiflow/api/internal/document"
	"plaiflow/api/internal/plan"
	"plaiflow/api/internal/tenant"
)

func (s *server) registerAccountingRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /v1/o/{organization}/accounting/categories", s.accountingCategories)
	m.HandleFunc("POST /v1/o/{organization}/accounting/categories", s.accountingCategories)
	m.HandleFunc("POST /v1/o/{organization}/accounting/categories/{category}/archive", s.accountingCategoryStatus)
	m.HandleFunc("POST /v1/o/{organization}/accounting/categories/{category}/restore", s.accountingCategoryStatus)
	m.HandleFunc("POST /v1/o/{organization}/accounting/categories/{category}/rename", s.renameAccountingCategory)
	m.HandleFunc("POST /v1/o/{organization}/accounting/rules", s.accountingRule)
	m.HandleFunc("GET /v1/o/{organization}/accounting/rules", s.listAccountingRules)
	m.HandleFunc("POST /v1/o/{organization}/accounting/rules/{rule}/retire", s.retireAccountingRule)
	m.HandleFunc("GET /v1/o/{organization}/accounting/review-queue", s.accountingReviewQueue)
	m.HandleFunc("GET /v1/o/{organization}/documents/{document}/accounting", s.accountingSuggestion)
	m.HandleFunc("POST /v1/o/{organization}/documents/{document}/accounting/approve", s.approveAccounting)
	m.HandleFunc("GET /v1/o/{organization}/documents/accounting.csv", s.exportAccounting)
	m.HandleFunc("GET /v1/o/{organization}/documents/accounting.xlsx", s.exportAccounting)
}

func (s *server) listAccountingRules(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	rules, err := s.config.Accounting.ListRules(r.Context(), session.UserID, membership.OrganizationID, r.URL.Query().Get("vendor_id"))
	if errors.Is(err, accounting.ErrInvalid) {
		writeError(w, r, 400, "invalid_vendor", "Vendor is invalid")
		return
	}
	if err != nil {
		writeError(w, r, 503, "accounting_unavailable", "Rules are unavailable")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, 200, map[string]any{"rules": rules})
}

func (s *server) accountingReviewQueue(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, 403, "forbidden", "Owner or Admin required")
		return
	}
	offset := 0
	if r.URL.Query().Get("offset") != "" {
		var parseErr error
		offset, parseErr = strconv.Atoi(r.URL.Query().Get("offset"))
		if parseErr != nil || offset < 0 || offset > 10000 {
			writeError(w, r, 400, "invalid_offset", "Review queue offset is invalid")
			return
		}
	}
	metas, err := s.config.Extraction.ListRecentReviews(r.Context(), session.UserID, membership.OrganizationID, 20, offset)
	if err != nil {
		writeError(w, r, 503, "accounting_unavailable", "Review queue is unavailable")
		return
	}
	type item struct {
		DocumentID string   `json:"document_id"`
		Basis      string   `json:"suggestion_basis"`
		Warnings   []string `json:"warning_codes"`
	}
	queue := make([]item, 0, len(metas))
	for _, meta := range metas {
		review, err := s.readReview(r.Context(), meta)
		if err != nil {
			writeError(w, r, 503, "accounting_unavailable", "Review queue is unavailable")
			return
		}
		evaluation, err := s.config.Accounting.Evaluate(r.Context(), session.UserID, membership.OrganizationID, *review)
		if errors.Is(err, accounting.ErrConflict) {
			queue = append(queue, item{DocumentID: meta.DocumentID, Basis: "source_changed", Warnings: []string{"source_changed"}})
			continue
		}
		if err != nil {
			writeError(w, r, 503, "accounting_unavailable", "Review queue is unavailable")
			return
		}
		if evaluation.Approved == nil || len(evaluation.Candidate.Warnings) > 0 {
			queue = append(queue, item{DocumentID: meta.DocumentID, Basis: evaluation.Candidate.Basis, Warnings: evaluation.Candidate.Warnings})
		}
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, 200, map[string]any{"items": queue, "limit": 20, "offset": offset, "has_more": len(metas) == 20})
}

func (s *server) renameAccountingCategory(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, 403, "forbidden", "Owner or Admin required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := r.ParseForm(); err != nil {
		writeError(w, r, 400, "invalid_category", "Category is invalid")
		return
	}
	err := s.config.Accounting.RenameCategory(r.Context(), session.UserID, membership.OrganizationID,
		r.PathValue("category"), r.PostForm.Get("name"), s.config.Now().UTC())
	if errors.Is(err, accounting.ErrInvalid) || errors.Is(err, accounting.ErrConflict) {
		writeError(w, r, 422, "invalid_category", "Category is invalid or already exists")
		return
	}
	if err != nil {
		writeError(w, r, 503, "accounting_unavailable", "Category could not be renamed")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "Renamed"})
}

func (s *server) accountingCategoryStatus(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, 403, "forbidden", "Owner or Admin required")
		return
	}
	archived := strings.HasSuffix(r.URL.Path, "/archive")
	err := s.config.Accounting.SetCategoryArchived(r.Context(), session.UserID, membership.OrganizationID,
		r.PathValue("category"), archived, s.config.Now().UTC())
	if errors.Is(err, accounting.ErrConflict) {
		writeError(w, r, 409, "category_changed", "Category changed; reload")
		return
	}
	if err != nil {
		writeError(w, r, 503, "accounting_unavailable", "Category could not be updated")
		return
	}
	writeJSON(w, 200, map[string]bool{"archived": archived})
}

func (s *server) retireAccountingRule(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, 403, "forbidden", "Owner or Admin required")
		return
	}
	err := s.config.Accounting.RetireRule(r.Context(), session.UserID, membership.OrganizationID,
		r.PathValue("rule"), s.config.Now().UTC())
	if errors.Is(err, accounting.ErrConflict) {
		writeError(w, r, 409, "rule_changed", "Rule changed; reload")
		return
	}
	if err != nil {
		writeError(w, r, 503, "accounting_unavailable", "Rule could not be retired")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "Retired"})
}

func (s *server) accountingCategories(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, r.Method == http.MethodPost)
	if !ok {
		return
	}
	if r.Method == http.MethodGet {
		categories, err := s.config.Accounting.ListCategories(r.Context(), session.UserID, membership.OrganizationID)
		if err != nil {
			writeError(w, r, 503, "accounting_unavailable", "Categories are unavailable")
			return
		}
		w.Header().Set("Cache-Control", "private, no-store")
		writeJSON(w, 200, map[string]any{"categories": categories})
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, 403, "forbidden", "Owner or Admin required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := r.ParseForm(); err != nil {
		writeError(w, r, 400, "invalid_category", "Category is invalid")
		return
	}
	category, err := s.config.Accounting.CreateCategory(r.Context(), session.UserID, membership.OrganizationID,
		accounting.Category{ID: newUUID(), Name: r.PostForm.Get("name")}, s.config.Now().UTC())
	if errors.Is(err, accounting.ErrInvalid) || errors.Is(err, accounting.ErrConflict) {
		writeError(w, r, 422, "invalid_category", "Category is invalid or already exists")
		return
	}
	if err != nil {
		writeError(w, r, 503, "accounting_unavailable", "Category could not be created")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, 201, category)
}

func (s *server) accountingSuggestion(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	meta, err := s.config.Extraction.CurrentReview(r.Context(), session.UserID, membership.OrganizationID, r.PathValue("document"))
	if err != nil || meta.ID == "" {
		writeError(w, r, 409, "review_required", "A confirmed current extraction is required")
		return
	}
	review, err := s.readReview(r.Context(), meta)
	if err != nil {
		writeError(w, r, 503, "accounting_unavailable", "Confirmed review is unavailable")
		return
	}
	result, err := s.config.Accounting.Evaluate(r.Context(), session.UserID, membership.OrganizationID, *review)
	if errors.Is(err, accounting.ErrConflict) {
		writeError(w, r, 409, "source_changed", "Source changed; review again")
		return
	}
	if err != nil {
		writeError(w, r, 503, "accounting_unavailable", "Suggestion is unavailable")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, 200, map[string]any{"source_review_id": review.ID, "source_review_revision": review.Revision,
		"document_type": review.DocumentType, "reviewed_values": review.Values, "suggestion": result})
	s.config.Logger.Info("accounting_suggestion", "basis", result.Candidate.Basis, "warning_count", len(result.Candidate.Warnings),
		"unresolved", result.Status == "NeedsReview", "conflict", result.Candidate.Basis == "rule_conflict",
		"cache_status", "disabled", "fallback_status", "disabled", "duration_ms", time.Since(started).Milliseconds(), "provider_cost_usd", 0)
}

func (s *server) approveAccounting(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, 403, "forbidden", "Owner or Admin required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := r.ParseForm(); err != nil {
		writeError(w, r, 400, "invalid_choice", "Choice is invalid")
		return
	}
	reviewRevision, e1 := strconv.Atoi(r.PostForm.Get("review_revision"))
	ruleVersion, e2 := strconv.Atoi(r.PostForm.Get("rule_set_version"))
	expectedRevision, e3 := strconv.Atoi(r.PostForm.Get("expected_revision"))
	if e1 != nil || e2 != nil || e3 != nil || reviewRevision < 1 || ruleVersion < 0 || expectedRevision < 0 {
		writeError(w, r, 400, "invalid_choice", "Choice revision is invalid")
		return
	}
	input := accounting.ApproveInput{ID: newUUID(), ActorID: session.UserID, OrganizationID: membership.OrganizationID,
		DocumentID: r.PathValue("document"), ReviewID: r.PostForm.Get("review_id"), ReviewRevision: reviewRevision,
		RuleSetVersion: ruleVersion, CategoryID: r.PostForm.Get("category_id"), VendorID: r.PostForm.Get("vendor_id"),
		UnmatchedReason: r.PostForm.Get("unmatched_vendor_reason"), ExpectedRevision: expectedRevision, ApprovedAt: s.config.Now().UTC()}
	input.RequestID = requestID(r)
	meta, err := s.config.Extraction.CurrentReview(r.Context(), session.UserID, membership.OrganizationID, input.DocumentID)
	if err != nil || meta.ID != input.ReviewID || meta.Revision != input.ReviewRevision {
		writeError(w, r, 409, "source_changed", "Source changed; review again")
		return
	}
	review, err := s.readReview(r.Context(), meta)
	if err != nil {
		writeError(w, r, 503, "accounting_unavailable", "Confirmed review is unavailable")
		return
	}
	evaluation, err := s.config.Accounting.Evaluate(r.Context(), session.UserID, membership.OrganizationID, *review)
	if err != nil || evaluation.RuleSetVersion != input.RuleSetVersion {
		writeError(w, r, 409, "suggestion_changed", "Suggestion changed; reload before approving")
		return
	}
	if evaluation.Candidate.RuleID != "" && evaluation.Candidate.CategoryID == input.CategoryID && evaluation.VendorID == input.VendorID {
		input.RuleID, input.RuleVersion, input.SuggestionBasis = evaluation.Candidate.RuleID, evaluation.Candidate.Version, evaluation.Candidate.Basis
	}
	approved, err := s.config.Accounting.Approve(r.Context(), input)
	if errors.Is(err, accounting.ErrConflict) {
		writeError(w, r, 409, "suggestion_changed", "Suggestion changed; reload before approving")
		return
	}
	if errors.Is(err, accounting.ErrInvalid) {
		writeError(w, r, 422, "invalid_choice", "Choose an active category and explain an unmatched vendor")
		return
	}
	if err != nil {
		writeError(w, r, 503, "accounting_unavailable", "Suggestion could not be approved")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, 200, approved)
}

func (s *server) accountingRule(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, 403, "forbidden", "Owner or Admin required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := r.ParseForm(); err != nil {
		writeError(w, r, 400, "invalid_rule", "Rule is invalid")
		return
	}
	rule, err := s.config.Accounting.CreateRule(r.Context(), accounting.RuleInput{ID: newUUID(), ActorID: session.UserID,
		OrganizationID: membership.OrganizationID, VendorID: r.PostForm.Get("vendor_id"),
		DocumentType: r.PostForm.Get("document_type"), CategoryID: r.PostForm.Get("category_id"), ApprovedAt: s.config.Now().UTC()})
	if errors.Is(err, accounting.ErrInvalid) {
		writeError(w, r, 422, "invalid_rule", "Rule is invalid")
		return
	}
	if err != nil {
		writeError(w, r, 503, "accounting_unavailable", "Rule could not be saved")
		return
	}
	writeJSON(w, 201, rule)
}

func (s *server) exportAccounting(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, 403, "forbidden", "Owner or Admin required")
		return
	}
	format := "csv"
	capability := plan.ExportCSV
	if strings.HasSuffix(r.URL.Path, ".xlsx") {
		format, capability = "xlsx", plan.ExportXLSX
	}
	decision, err := s.config.Gate.Check(r.Context(), membership.OrganizationID, capability, 0)
	if err != nil || !decision.Allowed {
		writeError(w, r, 403, "feature_unavailable", "Export is unavailable for this plan")
		return
	}
	// ponytail: cap synchronous encrypted reads at 100; a durable job is required before allowing larger exports.
	documentID := r.URL.Query().Get("document_id")
	if len(documentID) > 36 {
		writeError(w, r, 400, "invalid_filter", "Document filter is invalid")
		return
	}
	approved, err := s.config.Accounting.ListApproved(r.Context(), session.UserID, membership.OrganizationID, 100, documentID)
	if errors.Is(err, accounting.ErrTooMany) {
		writeError(w, r, 422, "too_many_rows", "Bulk export supports at most 100 rows; export individual documents instead")
		return
	}
	if err != nil {
		writeError(w, r, 503, "accounting_export_unavailable", "Export is unavailable")
		return
	}
	rows := make([][]string, 0, len(approved))
	for _, item := range approved {
		meta, err := s.config.Extraction.CurrentReview(r.Context(), session.UserID, membership.OrganizationID, item.DocumentID)
		if err != nil || meta.ID != item.ReviewID {
			writeError(w, r, 409, "source_changed", "Source changed; request a new export")
			return
		}
		review, err := s.readReview(r.Context(), meta)
		if err != nil {
			writeError(w, r, 503, "accounting_export_unavailable", "Export is unavailable")
			return
		}
		row, err := accounting.ExportRow(item, *review)
		if err != nil {
			writeError(w, r, 409, "source_changed", "Source changed; request a new export")
			return
		}
		rows = append(rows, row)
	}
	if err := s.config.Accounting.ValidateApprovedSnapshot(r.Context(), session.UserID, membership.OrganizationID, approved); err != nil {
		writeError(w, r, 409, "source_changed", "Authorization or source changed; request a new export")
		return
	}
	var output bytes.Buffer
	if err := document.WriteTable(&output, format, accounting.ExportHeader, rows); err != nil || output.Len() > 128<<20 {
		writeError(w, r, 503, "accounting_export_unavailable", "Export is unavailable")
		return
	}
	w.Header().Set("Content-Type", map[string]string{"csv": "text/csv; charset=utf-8", "xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"}[format])
	w.Header().Set("Content-Disposition", `attachment; filename="accounting_suggestions_v1-generic.`+format+`"`)
	w.Header().Set("Cache-Control", "private, no-store")
	_, _ = w.Write(output.Bytes())
	s.config.Logger.Info("accounting_export", "rows", len(rows), "bytes", output.Len(), "provider_cost_usd", 0, "duration_ms", time.Since(started).Milliseconds())
}
