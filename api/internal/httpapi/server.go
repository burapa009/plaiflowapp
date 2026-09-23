package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"plaiflow/api/internal/accounting"
	"plaiflow/api/internal/auth"
	"plaiflow/api/internal/business"
	"plaiflow/api/internal/document"
	"plaiflow/api/internal/drive"
	"plaiflow/api/internal/extraction"
	"plaiflow/api/internal/inbound"
	"plaiflow/api/internal/job"
	lineadapter "plaiflow/api/internal/line"
	"plaiflow/api/internal/ocr"
	"plaiflow/api/internal/plan"
	"plaiflow/api/internal/tenant"
	"plaiflow/api/internal/work"
)

const maxBody = 1 << 20

type Store interface {
	InsertEvents(context.Context, []inbound.Event) error
	Ready(context.Context) error
	Snapshot(context.Context) (inbound.Snapshot, error)
}

type Config struct {
	OCR               ocr.Store
	OCRJobs           job.Store
	OCRAuth           *job.WorkerAuth
	OCRTokens         *job.ArtifactToken
	OCRStorage        job.ArtifactStore
	Extraction        extraction.Store
	ExtractionEnabled bool
	Accounting        accounting.Store
	AccountingEnabled bool
	LineSecret        string
	LineChannel       string
	DashboardTokens   []string
	Logger            *slog.Logger
	Auth              *auth.Service
	Tenants           tenant.Store
	Work              work.Store
	Business          business.Store
	PlanStore         plan.Store
	Drive             *drive.Service
	Documents         *document.Service
	Jobs              job.Store
	JobWorkerAuth     *job.WorkerAuth
	JobArtifacts      job.ArtifactStore
	ArtifactTokens    *job.ArtifactToken
	Gate              work.Gate
	Now               func() time.Time
}

type server struct {
	config       Config
	store        Store
	webhookSlots chan struct{}
	limits       requestLimit
}

func New(config Config, store Store) http.Handler {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	s := &server{config: config, store: store, webhookSlots: make(chan struct{}, 32)}
	if s.config.Now == nil {
		s.config.Now = time.Now
	}
	if s.config.Gate == nil {
		s.config.Gate = work.UnlimitedGate{}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /readyz", s.ready)
	mux.HandleFunc("POST /webhooks/line", s.webhook)
	mux.HandleFunc("GET /v1/dashboard", s.dashboard)
	mux.HandleFunc("GET /v1/plans", s.listPlans)
	if config.OCR != nil && config.OCRAuth != nil {
		s.registerOCRRoutes(mux)
	}
	if config.Jobs != nil && config.JobWorkerAuth != nil {
		s.registerJobRoutes(mux)
	}
	if config.Auth == nil {
		mux.HandleFunc("GET /v1/auth/{provider}/callback", s.authDisabled)
		mux.HandleFunc("GET /v1/auth/{provider}/start", s.authDisabled)
	} else {
		mux.Handle("/v1/auth/", config.Auth)
		mux.Handle("/v1/session", config.Auth)
		mux.Handle("/v1/logout", config.Auth)
		mux.Handle("/v1/logout-all", config.Auth)
	}
	if config.Auth != nil && config.Tenants != nil {
		s.registerDriveRoutes(mux)
		mux.HandleFunc("GET /v1/organizations", s.listOrganizations)
		mux.HandleFunc("POST /v1/organizations", s.createOrganization)
		mux.HandleFunc("GET /v1/o/{organization}", s.organization)
		mux.HandleFunc("GET /v1/o/{organization}/business", s.getBusinessProfile)
		mux.HandleFunc("POST /v1/o/{organization}/business", s.updateBusinessProfile)
		mux.HandleFunc("GET /v1/o/{organization}/business/logo", s.businessLogo)
		mux.HandleFunc("GET /v1/o/{organization}/memberships", s.listMemberships)
		mux.HandleFunc("POST /v1/o/{organization}/invitations", s.createInvitation)
		mux.HandleFunc("POST /v1/o/{organization}/invitations/{invitation}/revoke", s.revokeInvitation)
		mux.HandleFunc("POST /v1/invitations/claim", s.claimInvitation)
		mux.HandleFunc("POST /v1/invitations/accept", s.acceptInvitation)
		mux.HandleFunc("POST /v1/o/{organization}/memberships/{user}/role", s.changeRole)
		mux.HandleFunc("POST /v1/o/{organization}/memberships/{user}/remove", s.removeMembership)
		mux.HandleFunc("POST /v1/o/{organization}/ownership", s.transferOwnership)
		mux.HandleFunc("POST /v1/o/{organization}/leave", s.leaveOrganization)
		mux.HandleFunc("GET /v1/o/{organization}/line-connections", s.listLineConnections)
		mux.HandleFunc("POST /v1/o/{organization}/line-link-codes", s.createLineLinkCode)
		mux.HandleFunc("POST /v1/o/{organization}/line-connections/{connection}/disconnect", s.disconnectLineConnection)
		if config.Work != nil {
			s.registerWorkRoutes(mux)
		}
		if config.Documents != nil {
			s.registerDocumentRoutes(mux)
		}
		if config.ExtractionEnabled && config.Extraction != nil && config.OCR != nil && config.OCRStorage != nil {
			s.registerExtractionRoutes(mux)
			if config.AccountingEnabled && config.Accounting != nil {
				s.registerAccountingRoutes(mux)
			}
		}
		if config.Business != nil && config.PlanStore != nil {
			s.registerBusinessRoutes(mux)
		}
	}
	return s.observe(mux)
}

func (s *server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) ready(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ready(r.Context()); err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "not_ready", "Service dependencies are not ready")
		return
	}
	if s.config.AccountingEnabled && s.config.Accounting != nil {
		if err := s.config.Accounting.ReadyAccounting(r.Context()); err != nil {
			writeError(w, r, http.StatusServiceUnavailable, "not_ready", "Accounting schema is not ready")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *server) webhook(w http.ResponseWriter, r *http.Request) {
	select {
	case s.webhookSlots <- struct{}{}:
		defer func() { <-s.webhookSlots }()
	default:
		writeError(w, r, http.StatusTooManyRequests, "busy", "Webhook capacity is temporarily full")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusRequestEntityTooLarge, "invalid_body", "Request body is invalid or too large")
		return
	}
	events, err := lineadapter.ParseSignedDelivery(body, r.Header.Get("x-line-signature"), s.config.LineSecret, s.config.LineChannel)
	if errors.Is(err, lineadapter.ErrInvalidSignature) {
		writeError(w, r, http.StatusUnauthorized, "invalid_signature", "Signature is invalid")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "malformed_delivery", "Webhook delivery is malformed")
		return
	}
	if len(events) > 0 {
		if err := s.store.InsertEvents(ctx, events); err != nil {
			writeError(w, r, http.StatusServiceUnavailable, "store_unavailable", "Webhook could not be stored")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (s *server) dashboard(w http.ResponseWriter, r *http.Request) {
	if !s.limits.allow() {
		writeError(w, r, http.StatusTooManyRequests, "rate_limited", "Too many requests")
		return
	}
	if !dashboardAuthorized(r, s.config.DashboardTokens) {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "Dashboard access is unauthorized")
		return
	}
	snapshot, err := s.store.Snapshot(r.Context())
	if err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "dashboard_unavailable", "Dashboard data is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func dashboardAuthorized(r *http.Request, candidates []string) bool {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	valid := 0
	for _, candidate := range candidates {
		providedHash, candidateHash := sha256.Sum256([]byte(token)), sha256.Sum256([]byte(candidate))
		valid |= subtle.ConstantTimeCompare(providedHash[:], candidateHash[:])
	}
	return valid == 1
}

func (s *server) authDisabled(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusNotFound, "provider_disabled", "Authentication provider is not enabled")
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

type requestIDKey struct{}

func (s *server) observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestID := newRequestID()
		if r.URL.Path == "/v1/dashboard" && dashboardAuthorized(r, s.config.DashboardTokens) && validRequestID(r.Header.Get("X-Request-ID")) {
			requestID = r.Header.Get("X-Request-ID")
		}
		w.Header().Set("X-Request-ID", requestID)
		r = r.WithContext(context.WithValue(r.Context(), requestIDKey{}, requestID))
		wrapped := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		s.config.Logger.Info("http_request", "request_id", requestID, "method", r.Method, "route", r.URL.Path, "status", wrapped.status, "duration_ms", time.Since(started).Milliseconds())
	})
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeJSON(w, status, map[string]string{"code": code, "message": message, "request_id": requestID(r)})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func requestID(r *http.Request) string {
	value, _ := r.Context().Value(requestIDKey{}).(string)
	return value
}
func newRequestID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "unavailable"
	}
	return hex.EncodeToString(value[:])
}

func validRequestID(value string) bool {
	if len(value) < 8 || len(value) > 64 {
		return false
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '-' && char != '_' {
			return false
		}
	}
	return true
}

type requestLimit struct {
	sync.Mutex
	started time.Time
	count   int
}

func (l *requestLimit) allow() bool {
	l.Lock()
	defer l.Unlock()
	if time.Since(l.started) >= time.Minute {
		l.started = time.Now()
		l.count = 0
	}
	l.count++
	return l.count <= 120
}
