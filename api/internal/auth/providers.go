package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	lineIssuer             = "https://access.line.me"
	lineAuthorizationURL   = "https://access.line.me/oauth2/v2.1/authorize"
	lineTokenURL           = "https://api.line.me/oauth2/v2.1/token"
	lineVerifyURL          = "https://api.line.me/oauth2/v2.1/verify"
	googleIssuer           = "https://accounts.google.com"
	googleAuthorizationURL = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL         = "https://oauth2.googleapis.com/token"
	googleJWKSURL          = "https://www.googleapis.com/oauth2/v3/certs"
)

type ProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

type oauthProvider struct {
	name                  string
	issuer                string
	authorizationEndpoint string
	tokenEndpoint         string
	verifyEndpoint        string
	scope                 string
	config                ProviderConfig
	client                *http.Client
}

func NewLINEProvider(config ProviderConfig) Provider {
	return &oauthProvider{
		name: "line", issuer: lineIssuer, authorizationEndpoint: lineAuthorizationURL,
		tokenEndpoint: lineTokenURL, verifyEndpoint: lineVerifyURL, scope: "openid profile",
		config: config, client: &http.Client{Timeout: 5 * time.Second},
	}
}

func NewGoogleProvider(config ProviderConfig) Provider {
	return &oauthProvider{
		name: "google", issuer: googleIssuer, authorizationEndpoint: googleAuthorizationURL,
		tokenEndpoint: googleTokenURL, verifyEndpoint: googleJWKSURL, scope: "openid email profile",
		config: config, client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (p *oauthProvider) Exchange(ctx context.Context, code, verifier, nonce string) (ExternalIdentity, error) {
	if code == "" || verifier == "" || nonce == "" {
		return ExternalIdentity{}, errors.New("callback parameters are incomplete")
	}
	if p.name == "google" {
		return p.exchangeGoogle(ctx, code, verifier, nonce)
	}
	if p.name != "line" {
		return ExternalIdentity{}, errors.New("provider exchange is not implemented")
	}
	token := struct {
		IDToken string `json:"id_token"`
	}{}
	if err := p.postForm(ctx, p.tokenEndpoint, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {p.config.RedirectURI},
		"client_id":     {p.config.ClientID},
		"client_secret": {p.config.ClientSecret},
		"code_verifier": {verifier},
	}, &token); err != nil || token.IDToken == "" {
		return ExternalIdentity{}, errors.New("provider token exchange failed")
	}
	claims := struct {
		Issuer    string `json:"iss"`
		Subject   string `json:"sub"`
		Audience  string `json:"aud"`
		ExpiresAt int64  `json:"exp"`
		Nonce     string `json:"nonce"`
		Name      string `json:"name"`
		Picture   string `json:"picture"`
		Email     string `json:"email"`
	}{}
	if err := p.postForm(ctx, p.verifyEndpoint, url.Values{
		"id_token": {token.IDToken}, "client_id": {p.config.ClientID}, "nonce": {nonce},
	}, &claims); err != nil {
		return ExternalIdentity{}, errors.New("provider identity verification failed")
	}
	if claims.Issuer != p.issuer || claims.Subject == "" || claims.Audience != p.config.ClientID || claims.ExpiresAt <= time.Now().Unix() || claims.Nonce != nonce {
		return ExternalIdentity{}, errors.New("provider identity verification failed")
	}
	return ExternalIdentity{
		Provider: p.name, Issuer: claims.Issuer, Subject: claims.Subject, Email: claims.Email,
		DisplayName: claims.Name, PictureURL: claims.Picture,
	}, nil
}

func (p *oauthProvider) exchangeGoogle(ctx context.Context, code, verifier, nonce string) (ExternalIdentity, error) {
	token := struct {
		IDToken string `json:"id_token"`
	}{}
	if err := p.postForm(ctx, p.tokenEndpoint, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {p.config.RedirectURI},
		"client_id":     {p.config.ClientID},
		"client_secret": {p.config.ClientSecret},
		"code_verifier": {verifier},
	}, &token); err != nil || token.IDToken == "" {
		return ExternalIdentity{}, errors.New("provider token exchange failed")
	}
	return p.verifyGoogleIDToken(ctx, token.IDToken, nonce)
}

func (p *oauthProvider) verifyGoogleIDToken(ctx context.Context, token, nonce string) (ExternalIdentity, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ExternalIdentity{}, errors.New("provider identity verification failed")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return ExternalIdentity{}, errors.New("provider identity verification failed")
	}
	header := struct {
		Algorithm string `json:"alg"`
		KeyID     string `json:"kid"`
	}{}
	if json.Unmarshal(headerBytes, &header) != nil || header.Algorithm != "RS256" || header.KeyID == "" {
		return ExternalIdentity{}, errors.New("provider identity verification failed")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, p.verifyEndpoint, nil)
	if err != nil {
		return ExternalIdentity{}, errors.New("provider identity verification failed")
	}
	response, err := p.client.Do(request)
	if err != nil {
		return ExternalIdentity{}, errors.New("provider identity verification failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ExternalIdentity{}, errors.New("provider identity verification failed")
	}
	keys := struct {
		Keys []struct {
			KeyID     string `json:"kid"`
			KeyType   string `json:"kty"`
			Algorithm string `json:"alg"`
			Modulus   string `json:"n"`
			Exponent  string `json:"e"`
		} `json:"keys"`
	}{}
	if json.NewDecoder(io.LimitReader(response.Body, 256<<10)).Decode(&keys) != nil {
		return ExternalIdentity{}, errors.New("provider identity verification failed")
	}
	var publicKey *rsa.PublicKey
	for _, key := range keys.Keys {
		if key.KeyID != header.KeyID || key.KeyType != "RSA" || key.Algorithm != "RS256" {
			continue
		}
		modulus, modErr := base64.RawURLEncoding.DecodeString(key.Modulus)
		exponent, expErr := base64.RawURLEncoding.DecodeString(key.Exponent)
		if modErr != nil || expErr != nil || len(exponent) == 0 || len(exponent) > 4 {
			continue
		}
		e := 0
		for _, value := range exponent {
			e = e<<8 | int(value)
		}
		publicKey = &rsa.PublicKey{N: new(big.Int).SetBytes(modulus), E: e}
		break
	}
	if publicKey == nil {
		return ExternalIdentity{}, errors.New("provider identity verification failed")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return ExternalIdentity{}, errors.New("provider identity verification failed")
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signature) != nil {
		return ExternalIdentity{}, errors.New("provider identity verification failed")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ExternalIdentity{}, errors.New("provider identity verification failed")
	}
	claims := struct {
		Issuer          string          `json:"iss"`
		Subject         string          `json:"sub"`
		Audience        json.RawMessage `json:"aud"`
		AuthorizedParty string          `json:"azp"`
		ExpiresAt       int64           `json:"exp"`
		Nonce           string          `json:"nonce"`
		Email           string          `json:"email"`
		EmailVerified   bool            `json:"email_verified"`
		Name            string          `json:"name"`
		Picture         string          `json:"picture"`
	}{}
	if json.Unmarshal(payload, &claims) != nil || claims.Subject == "" || claims.ExpiresAt <= time.Now().Unix() || claims.Nonce != nonce {
		return ExternalIdentity{}, errors.New("provider identity verification failed")
	}
	if claims.Issuer != googleIssuer && claims.Issuer != "accounts.google.com" {
		return ExternalIdentity{}, errors.New("provider identity verification failed")
	}
	if !audienceContains(claims.Audience, p.config.ClientID) || (claims.AuthorizedParty != "" && claims.AuthorizedParty != p.config.ClientID) {
		return ExternalIdentity{}, errors.New("provider identity verification failed")
	}
	return ExternalIdentity{
		Provider: p.name, Issuer: googleIssuer, Subject: claims.Subject, Email: claims.Email,
		EmailVerified: claims.EmailVerified, DisplayName: claims.Name, PictureURL: claims.Picture,
	}, nil
}

func audienceContains(raw json.RawMessage, expected string) bool {
	var single string
	if json.Unmarshal(raw, &single) == nil {
		return single == expected
	}
	var multiple []string
	if json.Unmarshal(raw, &multiple) != nil {
		return false
	}
	for _, audience := range multiple {
		if audience == expected {
			return true
		}
	}
	return false
}

func (p *oauthProvider) postForm(ctx context.Context, endpoint string, form url.Values, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := p.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
		return errors.New("provider returned an error")
	}
	return json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(target)
}

func (p *oauthProvider) Name() string   { return p.name }
func (p *oauthProvider) Issuer() string { return p.issuer }

func (p *oauthProvider) AuthorizationURL(attempt Attempt) (string, error) {
	if p.config.ClientID == "" || p.config.ClientSecret == "" || p.config.RedirectURI == "" {
		return "", errors.New("provider configuration is incomplete")
	}
	redirect, err := url.Parse(p.config.RedirectURI)
	if err != nil || redirect.Scheme != "https" || redirect.Host == "" || redirect.User != nil || redirect.RawQuery != "" || redirect.Fragment != "" {
		return "", errors.New("provider redirect URI is invalid")
	}
	if attempt.State == "" || attempt.Nonce == "" || attempt.CodeChallenge == "" {
		return "", errors.New("authorization attempt is incomplete")
	}
	endpoint, _ := url.Parse(p.authorizationEndpoint)
	query := endpoint.Query()
	query.Set("client_id", p.config.ClientID)
	query.Set("redirect_uri", redirect.String())
	query.Set("response_type", "code")
	query.Set("scope", p.scope)
	query.Set("state", attempt.State)
	query.Set("nonce", attempt.Nonce)
	query.Set("code_challenge", attempt.CodeChallenge)
	query.Set("code_challenge_method", "S256")
	endpoint.RawQuery = query.Encode()
	return endpoint.String(), nil
}
