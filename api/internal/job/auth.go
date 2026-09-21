package job

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var ErrUnauthorized = errors.New("job worker is unauthorized")

type WorkerIdentity struct {
	WorkerID, Environment string
}

type WorkerAuth struct {
	secret      []byte
	environment string
	allowed     map[string]bool
	mu          sync.Mutex
	seen        map[string]int64
}

type workerClaims struct {
	WorkerID    string   `json:"worker_id"`
	Environment string   `json:"environment"`
	Scopes      []string `json:"scopes"`
	IssuedAt    int64    `json:"iat"`
	ExpiresAt   int64    `json:"exp"`
	Nonce       string   `json:"nonce"`
}

func NewWorkerAuth(secret []byte, environment string, allowedScopes []string) (*WorkerAuth, error) {
	if len(secret) < 32 || strings.TrimSpace(environment) == "" || len(allowedScopes) == 0 {
		return nil, errors.New("invalid worker auth configuration")
	}
	allowed := make(map[string]bool, len(allowedScopes))
	for _, scope := range allowedScopes {
		if strings.TrimSpace(scope) == "" {
			return nil, errors.New("invalid worker auth scope")
		}
		allowed[scope] = true
	}
	return &WorkerAuth{secret: append([]byte(nil), secret...), environment: environment, allowed: allowed, seen: map[string]int64{}}, nil
}

func (a *WorkerAuth) Sign(workerID string, scopes []string, now time.Time, ttl time.Duration) (string, error) {
	if a == nil || strings.TrimSpace(workerID) == "" || len(workerID) > 100 || len(scopes) == 0 || ttl <= 0 || ttl > 5*time.Minute {
		return "", ErrUnauthorized
	}
	for _, scope := range scopes {
		if !a.allowed[scope] {
			return "", ErrUnauthorized
		}
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", err
	}
	claims := workerClaims{WorkerID: workerID, Environment: a.environment, Scopes: scopes, IssuedAt: now.UTC().Unix(), ExpiresAt: now.UTC().Add(ttl).Unix(), Nonce: fmt.Sprintf("%x", nonce[:])}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, a.secret)
	_, _ = mac.Write([]byte(encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (a *WorkerAuth) Verify(token, requiredScope string, now time.Time) (WorkerIdentity, error) {
	if a == nil || !a.allowed[requiredScope] {
		return WorkerIdentity{}, ErrUnauthorized
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return WorkerIdentity{}, ErrUnauthorized
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return WorkerIdentity{}, ErrUnauthorized
	}
	mac := hmac.New(sha256.New, a.secret)
	_, _ = mac.Write([]byte(parts[0]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return WorkerIdentity{}, ErrUnauthorized
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return WorkerIdentity{}, ErrUnauthorized
	}
	var claims workerClaims
	if json.Unmarshal(payload, &claims) != nil || claims.WorkerID == "" || len(claims.Nonce) != 32 || claims.Environment != a.environment || claims.IssuedAt > now.UTC().Add(time.Minute).Unix() || claims.ExpiresAt <= now.UTC().Unix() || claims.ExpiresAt-claims.IssuedAt > int64((5*time.Minute).Seconds()) {
		return WorkerIdentity{}, ErrUnauthorized
	}
	for _, scope := range claims.Scopes {
		if scope == requiredScope && a.allowed[scope] {
			a.mu.Lock()
			defer a.mu.Unlock()
			for nonce, expiry := range a.seen {
				if expiry <= now.UTC().Unix() {
					delete(a.seen, nonce)
				}
			}
			if _, exists := a.seen[claims.Nonce]; exists {
				return WorkerIdentity{}, ErrUnauthorized
			}
			a.seen[claims.Nonce] = claims.ExpiresAt
			return WorkerIdentity{WorkerID: claims.WorkerID, Environment: claims.Environment}, nil
		}
	}
	return WorkerIdentity{}, ErrUnauthorized
}
