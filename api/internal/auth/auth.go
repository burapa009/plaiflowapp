package auth

import (
	"errors"
	"net/url"
	"slices"
)

type Provider interface {
	AuthorizationStart() (url.URL, error)
	Callback(url.Values) (ExternalIdentity, error)
}

type ExternalIdentity struct {
	Provider  string
	SubjectID string
}

type CallbackConfig struct {
	WebBaseURL     string
	AllowedOrigins []string
}

func (c CallbackConfig) CallbackURL(provider string) (string, error) {
	base, err := url.Parse(c.WebBaseURL)
	if err != nil || base.Scheme != "https" || base.Host == "" || base.User != nil {
		return "", errors.New("invalid web base URL")
	}
	origin := base.Scheme + "://" + base.Host
	if !slices.Contains(c.AllowedOrigins, origin) {
		return "", errors.New("web origin is not allowed")
	}
	base.Path = "/auth/callback"
	base.RawQuery = url.Values{"provider": {provider}}.Encode()
	return base.String(), nil
}
