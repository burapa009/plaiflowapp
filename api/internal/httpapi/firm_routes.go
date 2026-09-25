package httpapi

import (
	"errors"
	"io"
	"net/http"
	"time"

	"plaiflow/api/internal/tenant"
)

func (s *server) registerFirmRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/o/{organization}/firm/grants", s.firmGrants)
	mux.HandleFunc("GET /v1/o/{organization}/firm/portfolio", s.firmPortfolio)
	mux.HandleFunc("POST /v1/o/{organization}/firm/grants", s.requestFirmGrant)
	mux.HandleFunc("POST /v1/o/{organization}/firm/grants/{grant}/approve", s.approveFirmGrant)
	mux.HandleFunc("POST /v1/o/{organization}/firm/grants/{grant}/accept", s.acceptFirmGrant)
	mux.HandleFunc("POST /v1/o/{organization}/firm/grants/{grant}/cancel", s.cancelFirmGrant)
	mux.HandleFunc("POST /v1/o/{organization}/firm/grants/{grant}/revoke", s.revokeFirmGrant)
	mux.HandleFunc("POST /v1/o/{organization}/firm/grants/{grant}/end", s.endFirmGrant)
	mux.HandleFunc("GET /v1/o/{organization}/firm/grants/{grant}/assignments", s.firmAssignments)
	mux.HandleFunc("POST /v1/o/{organization}/firm/grants/{grant}/assignments", s.assignFirmStaff)
	mux.HandleFunc("POST /v1/o/{organization}/firm/grants/{grant}/assignments/{user}/remove", s.removeFirmStaff)
	mux.HandleFunc("GET /v1/o/{organization}/firm/clients/{client}/documents/{document}/original", s.openFirmDocument)
}

func (s *server) firmPortfolio(w http.ResponseWriter, r *http.Request) {
	session, ok := s.authenticated(w, r)
	if !ok {
		return
	}
	page, err := s.config.Firm.ListPortfolio(r.Context(), session.UserID, r.PathValue("organization"), r.URL.Query().Get("cursor"))
	if err != nil {
		writeTenantError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, page)
}

func (s *server) firmGrants(w http.ResponseWriter, r *http.Request) {
	session, ok := s.authenticated(w, r)
	if !ok {
		return
	}
	grants, err := s.config.Firm.ListGrants(r.Context(), session.UserID, r.PathValue("organization"))
	if err != nil {
		writeTenantError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, map[string]any{"grants": grants})
}

func (s *server) requestFirmGrant(w http.ResponseWriter, r *http.Request) {
	session, ok := s.formMutation(w, r)
	if !ok {
		return
	}
	grant, err := s.config.Firm.RequestGrant(r.Context(), session.UserID, r.PathValue("organization"), r.FormValue("client_organization_id"), newUUID(), s.config.Now().UTC())
	if err != nil {
		writeTenantError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusCreated, grant)
}

func (s *server) transitionFirmGrant(w http.ResponseWriter, r *http.Request, action string, recent bool) {
	session, ok := s.formMutation(w, r)
	if !ok {
		return
	}
	if recent && s.config.Now().UTC().Sub(session.AuthenticatedAt) > 10*time.Minute {
		writeError(w, r, http.StatusForbidden, "recent_auth_required", "Recent authentication is required")
		return
	}
	if err := s.config.Firm.TransitionGrant(r.Context(), session.UserID, r.PathValue("organization"), r.PathValue("grant"), action, s.config.Now().UTC()); err != nil {
		writeTenantError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (s *server) approveFirmGrant(w http.ResponseWriter, r *http.Request) {
	s.transitionFirmGrant(w, r, "approve", true)
}
func (s *server) acceptFirmGrant(w http.ResponseWriter, r *http.Request) {
	s.transitionFirmGrant(w, r, "accept", false)
}
func (s *server) cancelFirmGrant(w http.ResponseWriter, r *http.Request) {
	s.transitionFirmGrant(w, r, "cancel", false)
}
func (s *server) revokeFirmGrant(w http.ResponseWriter, r *http.Request) {
	s.transitionFirmGrant(w, r, "revoke", true)
}
func (s *server) endFirmGrant(w http.ResponseWriter, r *http.Request) {
	s.transitionFirmGrant(w, r, "end", false)
}

func (s *server) firmAssignments(w http.ResponseWriter, r *http.Request) {
	session, ok := s.authenticated(w, r)
	if !ok {
		return
	}
	staff, err := s.config.Firm.ListStaff(r.Context(), session.UserID, r.PathValue("organization"), r.PathValue("grant"))
	if err != nil {
		writeTenantError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, map[string]any{"assignments": staff})
}
func (s *server) assignFirmStaff(w http.ResponseWriter, r *http.Request) {
	session, ok := s.formMutation(w, r)
	if !ok {
		return
	}
	err := s.config.Firm.AssignStaff(r.Context(), session.UserID, r.PathValue("organization"), r.PathValue("grant"), r.FormValue("user_id"), s.config.Now().UTC())
	if err != nil {
		writeTenantError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (s *server) removeFirmStaff(w http.ResponseWriter, r *http.Request) {
	session, ok := s.formMutation(w, r)
	if !ok {
		return
	}
	err := s.config.Firm.RemoveStaff(r.Context(), session.UserID, r.PathValue("organization"), r.PathValue("grant"), r.PathValue("user"), s.config.Now().UTC())
	if err != nil {
		writeTenantError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) openFirmDocument(w http.ResponseWriter, r *http.Request) {
	session, ok := s.authenticated(w, r)
	if !ok {
		return
	}
	preview := r.URL.Query().Get("preview") == "1"
	scope := "original.download"
	if preview {
		scope = "read"
	}
	doc, err := s.config.Firm.GetFirmDocument(r.Context(), session.UserID, r.PathValue("organization"),
		r.PathValue("client"), r.PathValue("document"), scope)
	if errors.Is(err, tenant.ErrNotFound) || errors.Is(err, tenant.ErrForbidden) {
		writeError(w, r, http.StatusNotFound, "not_found", "Resource was not found")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "document_unavailable", "Original is unavailable")
		return
	}
	if s.config.Documents == nil || s.config.Documents.Intake.Temporary == nil {
		writeError(w, r, http.StatusServiceUnavailable, "document_unavailable", "Original is unavailable")
		return
	}
	body, err := s.config.Documents.Intake.Temporary.Open(r.Context(), doc.StorageKey)
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "document_unavailable", "Original is unavailable")
		return
	}
	defer body.Close()
	w.Header().Set("Content-Type", doc.MIME)
	if preview {
		w.Header().Set("Content-Disposition", "inline; filename=\"document\"")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	} else {
		w.Header().Set("Content-Disposition", "attachment; filename=\"document\"")
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Security-Policy", "sandbox")
	if _, err := io.Copy(w, body); err != nil {
		s.config.Logger.Error("firm_document_stream_failed", "request_id", requestID(r))
	}
}
