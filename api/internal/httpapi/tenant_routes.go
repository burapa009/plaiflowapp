package httpapi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"plaiflow/api/internal/auth"
	"plaiflow/api/internal/tenant"
)

func (s *server) listOrganizations(w http.ResponseWriter, r *http.Request) {
	session, ok := s.authenticated(w, r)
	if !ok {
		return
	}
	organizations, err := s.config.Tenants.ListOrganizations(r.Context(), session.UserID)
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "organizations_unavailable", "Organizations are unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"organizations": organizations})
}

func (s *server) createOrganization(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		writeError(w, r, http.StatusUnsupportedMediaType, "invalid_content_type", "Request content type is invalid")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	session, ok := s.authenticatedMutation(w, r)
	if !ok {
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" || utf8.RuneCountInString(name) > 160 {
		writeError(w, r, http.StatusUnprocessableEntity, "invalid_name", "Organization name is required")
		return
	}
	organization, err := s.config.Tenants.CreateOrganization(r.Context(), session.UserID, newUUID(), name, time.Now().UTC())
	if err != nil {
		writeError(w, r, http.StatusConflict, "organization_not_created", "Organization could not be created")
		return
	}
	writeJSON(w, http.StatusCreated, organization)
}

func (s *server) organization(w http.ResponseWriter, r *http.Request) {
	session, ok := s.authenticated(w, r)
	if !ok {
		return
	}
	membership, err := s.config.Tenants.ResolveMembership(r.Context(), session.UserID, r.PathValue("organization"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Resource was not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"membership": membership})
}

func (s *server) authenticated(w http.ResponseWriter, r *http.Request) (auth.Session, bool) {
	session, err := s.config.Auth.Authenticate(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "Authentication is required")
		return auth.Session{}, false
	}
	return session, true
}

func (s *server) authenticatedMutation(w http.ResponseWriter, r *http.Request) (auth.Session, bool) {
	session, ok := s.authenticated(w, r)
	if !ok {
		return auth.Session{}, false
	}
	if !s.config.Auth.ValidMutation(r, session) {
		writeError(w, r, http.StatusForbidden, "csrf_rejected", "Request could not be verified")
		return auth.Session{}, false
	}
	return session, true
}

func newUUID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return ""
	}
	value[6] = value[6]&0x0f | 0x40
	value[8] = value[8]&0x3f | 0x80
	encoded := hex.EncodeToString(value)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32]
}

func (s *server) listMemberships(w http.ResponseWriter, r *http.Request) {
	session, ok := s.authenticated(w, r)
	if !ok {
		return
	}
	memberships, err := s.config.Tenants.ListMemberships(r.Context(), session.UserID, r.PathValue("organization"))
	if err != nil {
		writeTenantError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"memberships": memberships})
}

func (s *server) createInvitation(w http.ResponseWriter, r *http.Request) {
	session, ok := s.formMutation(w, r)
	if !ok {
		return
	}
	token, err := randomURLToken(32)
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "invitation_unavailable", "Invitation could not be created")
		return
	}
	now := time.Now().UTC()
	invitationID := newUUID()
	err = s.config.Tenants.CreateInvitation(r.Context(), tenant.InviteCreate{
		ID: invitationID, OrganizationID: r.PathValue("organization"), ActorUserID: session.UserID,
		TokenHash: secretHash(token), Now: now, ExpiresAt: now.Add(24 * time.Hour),
	})
	if err != nil {
		writeTenantError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, map[string]any{"invitation_id": invitationID, "token": token, "expires_at": now.Add(24 * time.Hour)})
}

func (s *server) claimInvitation(w http.ResponseWriter, r *http.Request) {
	if !validFormContentType(r) {
		writeError(w, r, http.StatusUnsupportedMediaType, "invalid_content_type", "Request content type is invalid")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	token := r.FormValue("token")
	returnTo, safe := inviteReturnPath(r.FormValue("return_to"))
	if token == "" || !safe {
		writeError(w, r, http.StatusNotFound, "invalid_invite", "Invitation is unavailable")
		return
	}
	handoff, err := randomURLToken(32)
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "invitation_unavailable", "Invitation is unavailable")
		return
	}
	now := time.Now().UTC()
	invitation, err := s.config.Tenants.ClaimInvitation(r.Context(), tenant.InviteClaim{
		TokenHash: secretHash(token), HandoffHash: secretHash(handoff), HandoffID: newUUID(), ReturnTo: returnTo,
		Now: now, ExpiresAt: now.Add(5 * time.Minute),
	})
	if err != nil {
		writeError(w, r, http.StatusNotFound, "invalid_invite", "Invitation is unavailable")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"invitation": invitation, "handoff_token": handoff})
}

func (s *server) acceptInvitation(w http.ResponseWriter, r *http.Request) {
	session, ok := s.formMutation(w, r)
	if !ok {
		return
	}
	handoff := r.FormValue("handoff_token")
	if handoff == "" {
		writeError(w, r, http.StatusNotFound, "invalid_invite", "Invitation is unavailable")
		return
	}
	organization, err := s.config.Tenants.AcceptInvitation(r.Context(), secretHash(handoff), session.UserID, time.Now().UTC())
	if err != nil {
		writeError(w, r, http.StatusNotFound, "invalid_invite", "Invitation is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, organization)
}

func (s *server) revokeInvitation(w http.ResponseWriter, r *http.Request) {
	session, ok := s.formMutation(w, r)
	if !ok {
		return
	}
	err := s.config.Tenants.RevokeInvitation(r.Context(), r.PathValue("organization"), session.UserID, r.PathValue("invitation"), time.Now().UTC())
	if err != nil {
		writeTenantError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) changeRole(w http.ResponseWriter, r *http.Request) {
	session, ok := s.formMutation(w, r)
	if !ok {
		return
	}
	err := s.config.Tenants.ChangeRole(r.Context(), r.PathValue("organization"), session.UserID, r.PathValue("user"), tenant.Role(r.FormValue("role")), time.Now().UTC())
	if err != nil {
		writeTenantError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) transferOwnership(w http.ResponseWriter, r *http.Request) {
	session, ok := s.formMutation(w, r)
	if !ok {
		return
	}
	targetUserID := r.FormValue("user_id")
	if targetUserID == "" {
		writeError(w, r, http.StatusUnprocessableEntity, "invalid_user", "New owner is required")
		return
	}
	err := s.config.Tenants.TransferOwnership(r.Context(), r.PathValue("organization"), session.UserID, targetUserID, time.Now().UTC())
	if err != nil {
		writeTenantError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) removeMembership(w http.ResponseWriter, r *http.Request) {
	session, ok := s.formMutation(w, r)
	if !ok {
		return
	}
	err := s.config.Tenants.RemoveMembership(r.Context(), r.PathValue("organization"), session.UserID, r.PathValue("user"), time.Now().UTC())
	if err != nil {
		writeTenantError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) leaveOrganization(w http.ResponseWriter, r *http.Request) {
	session, ok := s.formMutation(w, r)
	if !ok {
		return
	}
	err := s.config.Tenants.LeaveOrganization(r.Context(), r.PathValue("organization"), session.UserID, time.Now().UTC())
	if err != nil {
		writeTenantError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) listLineConnections(w http.ResponseWriter, r *http.Request) {
	session, ok := s.authenticated(w, r)
	if !ok {
		return
	}
	connections, err := s.config.Tenants.ListLineConnections(r.Context(), r.PathValue("organization"), session.UserID)
	if err != nil {
		writeTenantError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"connections": connections})
}

func (s *server) createLineLinkCode(w http.ResponseWriter, r *http.Request) {
	session, ok := s.formMutation(w, r)
	if !ok {
		return
	}
	if time.Since(session.AuthenticatedAt) > 10*time.Minute {
		writeError(w, r, http.StatusForbidden, "recent_auth_required", "Recent authentication is required")
		return
	}
	code, err := randomURLToken(16)
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "line_link_unavailable", "LINE link code is unavailable")
		return
	}
	now := time.Now().UTC()
	err = s.config.Tenants.CreateLineLinkCode(r.Context(), tenant.LineCodeCreate{
		ID: newUUID(), OrganizationID: r.PathValue("organization"), ActorUserID: session.UserID, Channel: s.config.LineChannel,
		CodeHash: secretHash(code), Now: now, ExpiresAt: now.Add(10 * time.Minute),
	})
	if err != nil {
		writeTenantError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, map[string]any{"link_code": code, "command": "เชื่อม " + code, "expires_at": now.Add(10 * time.Minute)})
}

func (s *server) disconnectLineConnection(w http.ResponseWriter, r *http.Request) {
	session, ok := s.formMutation(w, r)
	if !ok {
		return
	}
	err := s.config.Tenants.DisconnectLineConnection(r.Context(), r.PathValue("organization"), session.UserID, r.PathValue("connection"), time.Now().UTC())
	if err != nil {
		writeTenantError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) formMutation(w http.ResponseWriter, r *http.Request) (auth.Session, bool) {
	if !validFormContentType(r) {
		writeError(w, r, http.StatusUnsupportedMediaType, "invalid_content_type", "Request content type is invalid")
		return auth.Session{}, false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	return s.authenticatedMutation(w, r)
}

func validFormContentType(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
}

func writeTenantError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, tenant.ErrForbidden):
		writeError(w, r, http.StatusForbidden, "forbidden", "Action is not allowed")
	case errors.Is(err, tenant.ErrConflict):
		writeError(w, r, http.StatusConflict, "conflict", "Action conflicts with current state")
	default:
		writeError(w, r, http.StatusNotFound, "not_found", "Resource was not found")
	}
}

func randomURLToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func secretHash(value string) []byte {
	digest := sha256.Sum256([]byte(value))
	return digest[:]
}

func inviteReturnPath(value string) (string, bool) {
	if value == "" {
		return "/invite/accept", true
	}
	return value, value == "/invite/accept"
}
