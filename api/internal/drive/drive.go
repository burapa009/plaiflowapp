package drive

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"
)

var (
	ErrInvalidAttempt    = errors.New("drive authorization attempt is invalid")
	ErrInvalidGrant      = errors.New("drive credential is invalid")
	ErrReconnectRequired = errors.New("drive reconnection is required")
	ErrNotConnected      = errors.New("drive is not connected")
)

const (
	StatusNotConnected            = "Not Connected"
	StatusConnected               = "Connected"
	StatusReauthorizationRequired = "Reauthorization Required"
)

type OAuthAttempt struct {
	State, CodeChallenge string
	Scopes               []string
}

type Attempt struct {
	StateHash, BrowserHash           []byte
	OrganizationID, OrganizationName string
	UserID, SessionID, CodeVerifier  string
	CreatedAt, ExpiresAt             time.Time
}

type Credential struct {
	AccessToken, RefreshToken string
	Subject, Email            string
}

type Connection struct {
	OrganizationID        string    `json:"organization_id"`
	Status                string    `json:"status"`
	GoogleEmail           string    `json:"google_email,omitempty"`
	GoogleSubject         string    `json:"-"`
	FolderID              string    `json:"folder_id,omitempty"`
	AuthorizerUserID      string    `json:"-"`
	EncryptedRefreshToken []byte    `json:"-"`
	TokenNonce            []byte    `json:"-"`
	CredentialGeneration  int64     `json:"-"`
	ConnectedAt           time.Time `json:"connected_at,omitempty"`
	UpdatedAt             time.Time `json:"updated_at,omitempty"`
}

type Store interface {
	SaveAttempt(context.Context, Attempt) error
	ConsumeAttempt(context.Context, []byte, time.Time) (Attempt, error)
	SaveConnection(context.Context, Connection) error
	GetConnection(context.Context, string, string) (Connection, error)
	Disconnect(context.Context, string, string, time.Time) (Connection, error)
	RequireReconnect(context.Context, string, int64, time.Time) error
}

type Provider interface {
	AuthorizationURL(OAuthAttempt) string
	Exchange(context.Context, string, string) (Credential, error)
	CreateFolder(context.Context, string, string) (string, error)
	Refresh(context.Context, string) (string, error)
	Revoke(context.Context, string) error
}

type Config struct {
	Provider      Provider
	Store         Store
	EncryptionKey []byte
	Now           func() time.Time
}

type Service struct {
	provider Provider
	store    Store
	aead     cipher.AEAD
	now      func() time.Time
}

type Start struct {
	URL, State, BrowserSecret string
}

func New(config Config) (*Service, error) {
	if config.Provider == nil || config.Store == nil || len(config.EncryptionKey) != 32 {
		return nil, errors.New("drive configuration is invalid")
	}
	block, err := aes.NewCipher(config.EncryptionKey)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	return &Service{provider: config.Provider, store: config.Store, aead: aead, now: config.Now}, nil
}

func (s *Service) Begin(ctx context.Context, organizationID, userID, sessionID, organizationName string) (Start, error) {
	state, err := secureToken()
	if err != nil {
		return Start{}, err
	}
	browser, err := secureToken()
	if err != nil {
		return Start{}, err
	}
	verifier, err := secureToken()
	if err != nil {
		return Start{}, err
	}
	digest := sha256.Sum256([]byte(verifier))
	now := s.now().UTC()
	attempt := Attempt{
		StateHash: digestSecret(state), BrowserHash: digestSecret(browser), OrganizationID: organizationID,
		OrganizationName: organizationName, UserID: userID, SessionID: sessionID, CodeVerifier: verifier,
		CreatedAt: now, ExpiresAt: now.Add(10 * time.Minute),
	}
	if err := s.store.SaveAttempt(ctx, attempt); err != nil {
		return Start{}, err
	}
	oauth := OAuthAttempt{State: state, CodeChallenge: base64.RawURLEncoding.EncodeToString(digest[:]), Scopes: []string{
		"openid", "email", "https://www.googleapis.com/auth/drive.file",
	}}
	return Start{URL: s.provider.AuthorizationURL(oauth), State: state, BrowserSecret: browser}, nil
}

func (s *Service) Complete(ctx context.Context, state, browserSecret, userID, sessionID, code string) (Connection, error) {
	if state == "" || browserSecret == "" || code == "" {
		return Connection{}, ErrInvalidAttempt
	}
	attempt, err := s.store.ConsumeAttempt(ctx, digestSecret(state), s.now().UTC())
	if err != nil || !hmac.Equal(attempt.BrowserHash, digestSecret(browserSecret)) || attempt.UserID != userID || attempt.SessionID != sessionID {
		return Connection{}, ErrInvalidAttempt
	}
	credential, err := s.provider.Exchange(ctx, code, attempt.CodeVerifier)
	if err != nil || credential.RefreshToken == "" || credential.Subject == "" {
		return Connection{}, ErrInvalidGrant
	}
	folderID := ""
	if existing, existingErr := s.store.GetConnection(ctx, userID, attempt.OrganizationID); existingErr == nil {
		folderID = existing.FolderID
	} else {
		return Connection{}, existingErr
	}
	if folderID == "" {
		folderID, err = s.provider.CreateFolder(ctx, credential.AccessToken, "PlaiFlow - "+attempt.OrganizationName)
		if err != nil {
			return Connection{}, err
		}
	}
	ciphertext, nonce, err := s.encrypt(credential.RefreshToken, attempt.OrganizationID)
	if err != nil {
		return Connection{}, err
	}
	now := s.now().UTC()
	connection := Connection{
		OrganizationID: attempt.OrganizationID, Status: StatusConnected, GoogleEmail: credential.Email, GoogleSubject: credential.Subject,
		FolderID: folderID, AuthorizerUserID: userID, EncryptedRefreshToken: ciphertext, TokenNonce: nonce,
		CredentialGeneration: 1, ConnectedAt: now, UpdatedAt: now,
	}
	if err := s.store.SaveConnection(ctx, connection); err != nil {
		return Connection{}, err
	}
	return connection, nil
}

func (s *Service) Status(ctx context.Context, userID, organizationID string) (Connection, error) {
	return s.store.GetConnection(ctx, userID, organizationID)
}

func (s *Service) Disconnect(ctx context.Context, userID, organizationID string) error {
	connection, err := s.store.Disconnect(ctx, userID, organizationID, s.now().UTC())
	if err != nil {
		return err
	}
	refreshToken, err := s.decrypt(connection.EncryptedRefreshToken, connection.TokenNonce, organizationID)
	if err == nil && refreshToken != "" {
		_ = s.provider.Revoke(ctx, refreshToken)
	}
	return nil
}

func (s *Service) Check(ctx context.Context, userID, organizationID string) error {
	connection, err := s.store.GetConnection(ctx, userID, organizationID)
	if err != nil || connection.Status != StatusConnected {
		return ErrNotConnected
	}
	refreshToken, err := s.decrypt(connection.EncryptedRefreshToken, connection.TokenNonce, organizationID)
	if err != nil {
		return err
	}
	_, err = s.provider.Refresh(ctx, refreshToken)
	if errors.Is(err, ErrInvalidGrant) {
		if storeErr := s.store.RequireReconnect(ctx, organizationID, connection.CredentialGeneration, s.now().UTC()); storeErr != nil {
			return storeErr
		}
		return ErrReconnectRequired
	}
	return err
}

func (s *Service) encrypt(value, organizationID string) ([]byte, []byte, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	return s.aead.Seal(nil, nonce, []byte(value), []byte(organizationID)), nonce, nil
}

func (s *Service) decrypt(value, nonce []byte, organizationID string) (string, error) {
	plaintext, err := s.aead.Open(nil, nonce, value, []byte(organizationID))
	return string(plaintext), err
}

func secureToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func digestSecret(value string) []byte {
	digest := sha256.Sum256([]byte(value))
	return digest[:]
}
