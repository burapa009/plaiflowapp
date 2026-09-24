package httpapi

import (
	"errors"
	"net/http"
	"time"

	"plaiflow/api/internal/job"
	"plaiflow/api/internal/plan"
	"plaiflow/api/internal/tenant"
	"plaiflow/api/internal/work"
)

func reviewExportCapability(format string) work.Capability {
	if format == "xlsx" {
		return plan.ExportXLSX
	}
	return plan.ExportCSV
}

func (s *server) reviewExportCount(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, 403, "forbidden", "Owner or Admin required")
		return
	}
	if s.config.ReviewExports == nil {
		writeError(w, r, 503, "export_unavailable", "Export is unavailable")
		return
	}
	product, status := r.URL.Query().Get("product"), r.URL.Query().Get("status")
	if status == "" {
		status = "Available"
	}
	count, err := s.config.ReviewExports.CountDocumentExport(r.Context(), session.UserID, membership.OrganizationID, product, status,
		r.URL.Query().Get("date_from"), r.URL.Query().Get("date_to"))
	if err != nil {
		writeError(w, r, 400, "invalid_export", "Export filters are invalid")
		return
	}
	threshold, maximum := 100, 5000
	if product == "raw_documents" {
		threshold, maximum = 5000, 20000
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, 200, map[string]any{"count": count, "synchronous_threshold": threshold, "maximum_rows": maximum})
}

func (s *server) requestReviewExport(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, 403, "forbidden", "Owner or Admin required")
		return
	}
	if s.config.ReviewExports == nil || s.config.Jobs == nil {
		writeError(w, r, 503, "export_unavailable", "Export is unavailable")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxReviewBytes)
	if err := r.ParseForm(); err != nil {
		writeError(w, r, 400, "invalid_export", "Export request is invalid")
		return
	}
	product, status, format := r.PostForm.Get("product"), r.PostForm.Get("status"), r.PostForm.Get("format")
	if status == "" {
		status = "Available"
	}
	if format != "csv" && format != "xlsx" || product != "raw_documents" && product != "confirmed_values" && product != "approved_suggestions" ||
		status != "Available" && status != "Archived" && status != "Trash" || status == "Trash" && product != "raw_documents" {
		writeError(w, r, 400, "invalid_export", "Export product, status or format is invalid")
		return
	}
	decision, err := s.config.Gate.Check(r.Context(), membership.OrganizationID, reviewExportCapability(format), 0)
	if err != nil || !decision.Allowed {
		writeError(w, r, 403, "feature_unavailable", "Export is unavailable for this plan")
		return
	}
	state, err := s.config.ReviewExports.QueueDocumentExport(r.Context(), job.DocumentExportRequest{
		ID: newUUID(), OrganizationID: membership.OrganizationID, RequesterUserID: session.UserID,
		Product: product, Status: status, Format: format, DateFrom: r.PostForm.Get("date_from"),
		DateTo: r.PostForm.Get("date_to"), RequestID: requestID(r), Now: s.config.Now().UTC()})
	if err != nil {
		writeError(w, r, 422, "export_unavailable", "Export cannot be queued with these filters")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, 202, state)
}

func (s *server) reviewExportStatus(w http.ResponseWriter, r *http.Request) {
	if s.config.ReviewExports == nil {
		writeError(w, r, 503, "export_unavailable", "Export is unavailable")
		return
	}
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, 403, "forbidden", "Owner or Admin required")
		return
	}
	state, err := s.config.ReviewExports.GetDocumentExport(r.Context(), session.UserID, membership.OrganizationID, r.PathValue("export"))
	if errors.Is(err, tenant.ErrNotFound) {
		writeError(w, r, 404, "not_found", "Export was not found")
		return
	}
	if err != nil {
		writeError(w, r, 503, "export_unavailable", "Export status is unavailable")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, 200, state)
}

func (s *server) reviewExportDownload(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner && membership.Role != tenant.Admin {
		writeError(w, r, 403, "forbidden", "Owner or Admin required")
		return
	}
	if s.config.Jobs == nil || s.config.ArtifactTokens == nil {
		writeError(w, r, 503, "export_unavailable", "Export is unavailable")
		return
	}
	artifact, err := s.config.Jobs.AuthorizeExportArtifact(r.Context(), session.UserID, membership.OrganizationID, r.PathValue("export"))
	if err != nil || artifact.Product != "raw_documents" && artifact.Product != "confirmed_values" && artifact.Product != "approved_suggestions" {
		writeError(w, r, 409, "export_changed", "Export is unavailable; request a new file")
		return
	}
	decision, err := s.config.Gate.Check(r.Context(), membership.OrganizationID, reviewExportCapability(artifact.Format), 0)
	if err != nil || !decision.Allowed {
		writeError(w, r, 403, "feature_unavailable", "Export is unavailable for this plan")
		return
	}
	token, err := s.config.ArtifactTokens.Sign(artifact.JobID, artifact.OrganizationID, session.UserID, s.config.Now().UTC())
	if err != nil {
		writeError(w, r, 503, "export_unavailable", "Download is unavailable")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, 200, map[string]any{"download_url": "/v1/export-artifacts/" + artifact.JobID + "?token=" + token,
		"expires_at": s.config.Now().UTC().Add(15 * time.Minute)})
}
