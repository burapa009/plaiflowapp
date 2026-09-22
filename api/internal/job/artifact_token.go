package job

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type ArtifactToken struct{ secret []byte }
type artifactClaims struct {
	JobID          string `json:"job_id"`
	OrganizationID string `json:"organization_id"`
	UserID         string `json:"user_id"`
	ExpiresAt      int64  `json:"exp"`
}

func NewArtifactToken(secret []byte) (*ArtifactToken, error) {
	if len(secret) < 32 {
		return nil, errors.New("invalid artifact token configuration")
	}
	return &ArtifactToken{secret: append([]byte(nil), secret...)}, nil
}
func (a *ArtifactToken) Sign(jobID, organizationID, userID string, now time.Time) (string, error) {
	return a.SignTTL(jobID, organizationID, userID, now, 15*time.Minute)
}

func (a *ArtifactToken) SignTTL(jobID, organizationID, userID string, now time.Time, ttl time.Duration) (string, error) {
	if ttl <= 0 || ttl > 15*time.Minute { return "", ErrUnauthorized }
	if jobID == "" || organizationID == "" || userID == "" {
		return "", ErrUnauthorized
	}
	body, err := json.Marshal(artifactClaims{JobID: jobID, OrganizationID: organizationID, UserID: userID, ExpiresAt: now.Add(ttl).Unix()})
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(body)
	mac := hmac.New(sha256.New, a.secret)
	_, _ = mac.Write([]byte(encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
func (a *ArtifactToken) Verify(token string, now time.Time) (string, string, string, error) {
	parts := strings.Split(token, ".")
	if a == nil || len(parts) != 2 {
		return "", "", "", ErrUnauthorized
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", "", ErrUnauthorized
	}
	mac := hmac.New(sha256.New, a.secret)
	_, _ = mac.Write([]byte(parts[0]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return "", "", "", ErrUnauthorized
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	var claims artifactClaims
	if err != nil || json.Unmarshal(body, &claims) != nil || claims.JobID == "" || claims.OrganizationID == "" || claims.UserID == "" || claims.ExpiresAt <= now.Unix() {
		return "", "", "", ErrUnauthorized
	}
	return claims.JobID, claims.OrganizationID, claims.UserID, nil
}
