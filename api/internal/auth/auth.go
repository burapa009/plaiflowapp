package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"slices"
)

type Provider interface {
	Name() string
	Issuer() string
	AuthorizationURL(Attempt) (string, error)
	Exchange(context.Context, string, string, string) (ExternalIdentity, error)
}

type ExternalIdentity struct {
	Provider      string
	Issuer        string
	Subject       string
	Email         string
	EmailVerified bool
	DisplayName   string
	PictureURL    string
}

type Attempt struct {
	State         string
	Nonce         string
	CodeVerifier  string
	CodeChallenge string
}

func NewAttempt() (Attempt, error) {
	state, err := randomToken()
	if err != nil {
		return Attempt{}, err
	}
	nonce, err := randomToken()
	if err != nil {
		return Attempt{}, err
	}
	verifier, err := randomToken()
	if err != nil {
		return Attempt{}, err
	}
	digest := sha256.Sum256([]byte(verifier))
	return Attempt{
		State: state, Nonce: nonce, CodeVerifier: verifier,
		CodeChallenge: base64.RawURLEncoding.EncodeToString(digest[:]),
	}, nil
}

func randomToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", errors.New("secure random generation failed")
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

type CallbackConfig struct {
	WebBaseURL     string
	AllowedOrigins []string
}

func (c CallbackConfig) CallbackURL(provider string) (string, error) {
	base, err := url.Parse(c.WebBaseURL)
	if err != nil || base.Host == "" || base.User != nil || base.Path != "" || base.RawQuery != "" || base.Fragment != "" || (base.Scheme != "https" && !(base.Scheme == "http" && isLocalhost(base.Hostname()))) {
		return "", errors.New("invalid web base URL")
	}
	if provider != "line" && provider != "google" {
		return "", errors.New("invalid provider")
	}
	origin := base.Scheme + "://" + base.Host
	if !slices.Contains(c.AllowedOrigins, origin) {
		return "", errors.New("web origin is not allowed")
	}
	base.Path = "/api/auth/" + provider + "/callback"
	return base.String(), nil
}
