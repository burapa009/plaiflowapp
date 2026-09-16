package httpapi

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"plaiflow/api/internal/drive"
	"plaiflow/api/internal/plan"
	"plaiflow/api/internal/tenant"
)

const DriveCookieName = "__Host-plaiflow-drive"

func (s *server) registerDriveRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/o/{organization}/drive", s.driveStatus)
	mux.HandleFunc("POST /v1/o/{organization}/drive/connect", s.connectDrive)
	mux.HandleFunc("POST /v1/o/{organization}/drive/disconnect", s.disconnectDrive)
	mux.HandleFunc("POST /v1/o/{organization}/drive/check", s.checkDrive)
	mux.HandleFunc("GET /v1/drive/callback", s.driveCallback)
}

func (s *server) driveStatus(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	if s.config.Drive == nil {
		writeJSON(w, http.StatusOK, map[string]any{"configured": false, "connection": drive.Connection{OrganizationID: membership.OrganizationID, Status: drive.StatusNotConnected}})
		return
	}
	connection, err := s.config.Drive.Status(r.Context(), session.UserID, membership.OrganizationID)
	if err != nil {
		writeDriveError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"configured": true, "connection": connection})
}

func (s *server) connectDrive(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner {
		writeError(w, r, http.StatusForbidden, "forbidden", "Only the Owner can connect Google Drive")
		return
	}
	if s.config.Now().UTC().Sub(session.AuthenticatedAt) > 10*time.Minute {
		writeError(w, r, http.StatusForbidden, "recent_auth_required", "Recent authentication is required")
		return
	}
	if s.config.Drive == nil {
		writeError(w, r, http.StatusServiceUnavailable, "drive_not_configured", "Google Drive is not configured")
		return
	}
	decision, err := s.config.Gate.Check(r.Context(), membership.OrganizationID, plan.ExportDrive, 1)
	if err != nil || !decision.Allowed {
		writeError(w, r, http.StatusPaymentRequired, "feature_unavailable", "Google Drive is unavailable for this Plan")
		return
	}
	organizationName := membership.OrganizationName
	if organizationName == "" {
		organizationName = membership.OrganizationID
	}
	start, err := s.config.Drive.Begin(r.Context(), membership.OrganizationID, session.UserID, session.ID, organizationName)
	if err != nil {
		writeDriveError(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: DriveCookieName, Value: start.BrowserSecret, Path: "/", MaxAge: 600, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, start.URL, http.StatusSeeOther)
}

func (s *server) driveCallback(w http.ResponseWriter, r *http.Request) {
	if s.config.Drive == nil {
		writeError(w, r, http.StatusNotFound, "drive_not_configured", "Google Drive is not configured")
		return
	}
	session, ok := s.authenticated(w, r)
	if !ok {
		return
	}
	browser, err := r.Cookie(DriveCookieName)
	if err != nil || r.URL.Query().Get("error") != "" {
		http.Redirect(w, r, "/organizations?drive=access_denied", http.StatusSeeOther)
		return
	}
	connection, err := s.config.Drive.Complete(r.Context(), r.URL.Query().Get("state"), browser.Value, session.UserID, session.ID, r.URL.Query().Get("code"))
	if err != nil {
		http.Redirect(w, r, "/organizations?drive=failed", http.StatusSeeOther)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: DriveCookieName, Value: "", Path: "/", MaxAge: -1, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	location := "/o/" + url.PathEscape(connection.OrganizationID) + "/connections?drive=connected"
	http.Redirect(w, r, location, http.StatusSeeOther)
}

func (s *server) disconnectDrive(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner {
		writeError(w, r, http.StatusForbidden, "forbidden", "Only the Owner can disconnect Google Drive")
		return
	}
	if s.config.Drive == nil {
		writeError(w, r, http.StatusServiceUnavailable, "drive_not_configured", "Google Drive is not configured")
		return
	}
	if err := s.config.Drive.Disconnect(r.Context(), session.UserID, membership.OrganizationID); err != nil {
		writeDriveError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) checkDrive(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner {
		writeError(w, r, http.StatusForbidden, "forbidden", "Only the Owner can check Google Drive")
		return
	}
	if s.config.Drive == nil {
		writeError(w, r, http.StatusServiceUnavailable, "drive_not_configured", "Google Drive is not configured")
		return
	}
	if err := s.config.Drive.Check(r.Context(), session.UserID, membership.OrganizationID); err != nil {
		writeDriveError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeDriveError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, drive.ErrReconnectRequired):
		writeError(w, r, http.StatusConflict, "reconnect_required", "Google Drive must be reconnected")
	case errors.Is(err, drive.ErrNotConnected):
		writeError(w, r, http.StatusConflict, "drive_not_connected", "Google Drive is not connected")
	case errors.Is(err, drive.ErrInvalidAttempt):
		writeError(w, r, http.StatusBadRequest, "invalid_drive_attempt", "Google Drive authorization could not be completed")
	case errors.Is(err, tenant.ErrForbidden):
		writeError(w, r, http.StatusForbidden, "forbidden", "Action is not allowed")
	default:
		writeError(w, r, http.StatusServiceUnavailable, "drive_unavailable", "Google Drive is unavailable")
	}
}
