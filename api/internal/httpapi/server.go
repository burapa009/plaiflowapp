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

	"plaiflow/api/internal/inbound"
	lineadapter "plaiflow/api/internal/line"
)

const maxBody = 1 << 20

type Store interface {
	InsertEvents(context.Context, []inbound.Event) error
	Ready(context.Context) error
	Snapshot(context.Context) (inbound.Snapshot, error)
}

type Config struct {
	LineSecret      string
	LineChannel     string
	DashboardTokens []string
	Logger          *slog.Logger
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
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /readyz", s.ready)
	mux.HandleFunc("POST /webhooks/line", s.webhook)
	mux.HandleFunc("GET /v1/dashboard", s.dashboard)
	mux.HandleFunc("GET /v1/auth/{provider}/callback", s.authDisabled)
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
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	valid := 0
	for _, candidate := range s.config.DashboardTokens {
		providedHash, candidateHash := sha256.Sum256([]byte(token)), sha256.Sum256([]byte(candidate))
		valid |= subtle.ConstantTimeCompare(providedHash[:], candidateHash[:])
	}
	if valid != 1 {
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
