package drive

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"
)

type memoryStore struct {
	attempt      Attempt
	connection   Connection
	reconnects   int
	disconnected bool
}

func (s *memoryStore) SaveAttempt(_ context.Context, attempt Attempt) error {
	s.attempt = attempt
	return nil
}
func (s *memoryStore) ConsumeAttempt(context.Context, []byte, time.Time) (Attempt, error) {
	return s.attempt, nil
}
func (s *memoryStore) SaveConnection(_ context.Context, connection Connection) error {
	s.connection = connection
	return nil
}
func (s *memoryStore) GetConnection(context.Context, string, string) (Connection, error) {
	return s.connection, nil
}
func (s *memoryStore) Disconnect(context.Context, string, string, time.Time) (Connection, error) {
	s.disconnected = true
	connection := s.connection
	s.connection.Status, s.connection.EncryptedRefreshToken, s.connection.TokenNonce = StatusNotConnected, nil, nil
	return connection, nil
}
func (s *memoryStore) RequireReconnect(context.Context, string, int64, time.Time) error {
	s.reconnects++
	s.connection.Status, s.connection.EncryptedRefreshToken, s.connection.TokenNonce = StatusReauthorizationRequired, nil, nil
	return nil
}

type providerDouble struct{ refreshErr error }

func (*providerDouble) AuthorizationURL(attempt OAuthAttempt) string {
	return "https://accounts.example/auth?" + url.Values{
		"scope": {strings.Join(attempt.Scopes, " ")}, "access_type": {"offline"}, "state": {attempt.State},
	}.Encode()
}
func (*providerDouble) Exchange(context.Context, string, string) (Credential, error) {
	return Credential{AccessToken: "access", RefreshToken: "refresh", Subject: "google-user", Email: "owner@example.com"}, nil
}
func (*providerDouble) CreateFolder(context.Context, string, string) (string, error) {
	return "folder-1", nil
}
func (p *providerDouble) Refresh(context.Context, string) (string, error) { return "", p.refreshErr }
func (*providerDouble) Revoke(context.Context, string) error {
	return errors.New("provider unavailable")
}

func TestDriveAuthorizationIsSeparateAndMinimumScope(t *testing.T) {
	store := &memoryStore{}
	service, err := New(Config{Provider: &providerDouble{}, Store: store, EncryptionKey: make([]byte, 32), Now: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	start, err := service.Begin(context.Background(), "org-1", "user-1", "session-1", "Acme")
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := url.Parse(start.URL)
	scope := parsed.Query().Get("scope")
	if scope != "openid email https://www.googleapis.com/auth/drive.file" || parsed.Query().Get("access_type") != "offline" || strings.Contains(scope, "profile") {
		t.Fatalf("url=%s", start.URL)
	}
	connection, err := service.Complete(context.Background(), start.State, start.BrowserSecret, "user-1", "session-1", "code")
	if err != nil {
		t.Fatal(err)
	}
	if connection.FolderID != "folder-1" || string(connection.EncryptedRefreshToken) == "refresh" || len(connection.TokenNonce) == 0 {
		t.Fatalf("connection=%+v", connection)
	}
}

func TestRevokedCredentialStopsAndCreatesOneReconnectTask(t *testing.T) {
	store := &memoryStore{}
	provider := &providerDouble{refreshErr: ErrInvalidGrant}
	service, _ := New(Config{Provider: provider, Store: store, EncryptionKey: make([]byte, 32), Now: time.Now})
	start, _ := service.Begin(context.Background(), "org-1", "user-1", "session-1", "Acme")
	_, _ = service.Complete(context.Background(), start.State, start.BrowserSecret, "user-1", "session-1", "code")
	if err := service.Check(context.Background(), "user-1", "org-1"); !errors.Is(err, ErrReconnectRequired) {
		t.Fatalf("check err=%v", err)
	}
	if store.reconnects != 1 || store.connection.Status != StatusReauthorizationRequired || len(store.connection.EncryptedRefreshToken) != 0 {
		t.Fatalf("reconnects=%d connection=%+v", store.reconnects, store.connection)
	}
	if err := service.Check(context.Background(), "user-1", "org-1"); !errors.Is(err, ErrNotConnected) || store.reconnects != 1 {
		t.Fatalf("retry err=%v reconnects=%d", err, store.reconnects)
	}
}

func TestDisconnectStopsLocallyEvenWhenProviderRevocationFails(t *testing.T) {
	store := &memoryStore{}
	service, _ := New(Config{Provider: &providerDouble{}, Store: store, EncryptionKey: make([]byte, 32), Now: time.Now})
	start, _ := service.Begin(context.Background(), "org-1", "user-1", "session-1", "Acme")
	_, _ = service.Complete(context.Background(), start.State, start.BrowserSecret, "user-1", "session-1", "code")
	if err := service.Disconnect(context.Background(), "user-1", "org-1"); err != nil || !store.disconnected || len(store.connection.EncryptedRefreshToken) != 0 {
		t.Fatalf("disconnected=%v connection=%+v err=%v", store.disconnected, store.connection, err)
	}
}
