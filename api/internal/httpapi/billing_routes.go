package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"plaiflow/api/internal/billing"
	"plaiflow/api/internal/plan"
	"plaiflow/api/internal/tenant"
)

func (s *server) registerBillingRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/o/{organization}/billing", s.getBilling)
	mux.HandleFunc("POST /v1/o/{organization}/billing/intents", s.startBilling)
	mux.HandleFunc("POST /v1/o/{organization}/billing/cancel", s.cancelBilling)
	mux.HandleFunc("POST /v1/o/{organization}/billing/resume", s.resumeBilling)
}

func (s *server) getBilling(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, false)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner {
		writeError(w, r, http.StatusForbidden, "forbidden", "Billing is Owner-only")
		return
	}
	summary, err := s.config.Billing.Store.Summary(r.Context(), session.UserID, membership.OrganizationID, s.config.Now().UTC())
	if err != nil {
		writeBillingError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, summary)
}

func (s *server) startBilling(w http.ResponseWriter, r *http.Request) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner {
		writeError(w, r, http.StatusForbidden, "forbidden", "Billing is Owner-only")
		return
	}
	intent, err := s.config.Billing.Start(r.Context(), session.UserID, membership.OrganizationID, newUUID(),
		plan.Key(r.FormValue("plan")), plan.Interval(r.FormValue("interval")))
	if err != nil {
		writeBillingError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusCreated, intent)
}

func (s *server) cancelBilling(w http.ResponseWriter, r *http.Request) {
	s.setBillingCancellation(w, r, true)
}
func (s *server) resumeBilling(w http.ResponseWriter, r *http.Request) {
	s.setBillingCancellation(w, r, false)
}
func (s *server) setBillingCancellation(w http.ResponseWriter, r *http.Request, cancel bool) {
	session, membership, ok := s.workContext(w, r, true)
	if !ok {
		return
	}
	if membership.Role != tenant.Owner {
		writeError(w, r, http.StatusForbidden, "forbidden", "Billing is Owner-only")
		return
	}
	if cancel && r.FormValue("confirm") != "cancel" {
		writeError(w, r, http.StatusUnprocessableEntity, "confirmation_required", "Confirm cancellation")
		return
	}
	if s.config.Now().UTC().Sub(session.AuthenticatedAt) > 10*time.Minute {
		writeError(w, r, http.StatusForbidden, "recent_auth_required", "Recent authentication is required")
		return
	}
	if err := s.config.Billing.Store.SetCancellation(r.Context(), session.UserID, membership.OrganizationID, cancel, s.config.Now().UTC()); err != nil {
		writeBillingError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) omiseWebhook(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_webhook", "Invalid webhook")
		return
	}
	if err := billing.VerifyWebhook(body, r.Header.Get("Omise-Signature-Timestamp"), r.Header.Get("Omise-Signature"), s.config.OmiseWebhookSecret, s.config.Now().UTC()); err != nil {
		writeError(w, r, http.StatusUnauthorized, "invalid_webhook", "Invalid webhook")
		return
	}
	var event struct {
		ID   string `json:"id"`
		Key  string `json:"key"`
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &event); err != nil || event.ID == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_webhook", "Invalid webhook")
		return
	}
	if event.Key == "charge.complete" {
		if !strings.HasPrefix(event.Data.ID, "chrg_") {
			writeError(w, r, http.StatusBadRequest, "invalid_webhook", "Invalid webhook")
			return
		}
		if err := s.config.Billing.Store.QueueEvent(r.Context(), billing.Event{ID: event.ID, ChargeID: event.Data.ID, Type: event.Key}); err != nil {
			writeError(w, r, http.StatusServiceUnavailable, "webhook_unavailable", "Webhook is unavailable")
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeBillingError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, billing.ErrForbidden):
		writeError(w, r, http.StatusForbidden, "forbidden", "Billing is Owner-only")
	case errors.Is(err, billing.ErrInvalidPlan):
		writeError(w, r, http.StatusUnprocessableEntity, "invalid_plan", "Plan or interval is invalid")
	case errors.Is(err, billing.ErrConflict):
		writeError(w, r, http.StatusConflict, "billing_conflict", "A payment or plan change is already pending")
	default:
		writeError(w, r, http.StatusServiceUnavailable, "billing_unavailable", "Billing is temporarily unavailable")
	}
}
