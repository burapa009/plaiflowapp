package drive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GoogleConfig struct {
	ClientID, ClientSecret, RedirectURI string
	HTTPClient                          *http.Client
	AuthorizationEndpoint               string
	TokenEndpoint                       string
	UserInfoEndpoint                    string
	DriveFilesEndpoint                  string
	RevokeEndpoint                      string
}

type GoogleProvider struct {
	config GoogleConfig
}

func NewGoogleProvider(config GoogleConfig) (*GoogleProvider, error) {
	if config.ClientID == "" || config.ClientSecret == "" || config.RedirectURI == "" {
		return nil, errors.New("google drive OAuth configuration is invalid")
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	if config.AuthorizationEndpoint == "" {
		config.AuthorizationEndpoint = "https://accounts.google.com/o/oauth2/v2/auth"
	}
	if config.TokenEndpoint == "" {
		config.TokenEndpoint = "https://oauth2.googleapis.com/token"
	}
	if config.UserInfoEndpoint == "" {
		config.UserInfoEndpoint = "https://openidconnect.googleapis.com/v1/userinfo"
	}
	if config.DriveFilesEndpoint == "" {
		config.DriveFilesEndpoint = "https://www.googleapis.com/drive/v3/files"
	}
	if config.RevokeEndpoint == "" {
		config.RevokeEndpoint = "https://oauth2.googleapis.com/revoke"
	}
	return &GoogleProvider{config: config}, nil
}

func (p *GoogleProvider) AuthorizationURL(attempt OAuthAttempt) string {
	query := url.Values{
		"client_id": {p.config.ClientID}, "redirect_uri": {p.config.RedirectURI}, "response_type": {"code"},
		"scope": {strings.Join(attempt.Scopes, " ")}, "state": {attempt.State}, "code_challenge": {attempt.CodeChallenge},
		"code_challenge_method": {"S256"}, "access_type": {"offline"}, "prompt": {"consent"},
	}
	return p.config.AuthorizationEndpoint + "?" + query.Encode()
}

func (p *GoogleProvider) Exchange(ctx context.Context, code, verifier string) (Credential, error) {
	var token struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Error        string `json:"error"`
	}
	err := p.postForm(ctx, p.config.TokenEndpoint, url.Values{
		"client_id": {p.config.ClientID}, "client_secret": {p.config.ClientSecret}, "redirect_uri": {p.config.RedirectURI},
		"grant_type": {"authorization_code"}, "code": {code}, "code_verifier": {verifier},
	}, &token)
	if err != nil || token.Error != "" || token.AccessToken == "" || token.RefreshToken == "" {
		return Credential{}, ErrInvalidGrant
	}
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, p.config.UserInfoEndpoint, nil)
	request.Header.Set("Authorization", "Bearer "+token.AccessToken)
	response, err := p.config.HTTPClient.Do(request)
	if err != nil {
		return Credential{}, err
	}
	defer response.Body.Close()
	var identity struct {
		Subject string `json:"sub"`
		Email   string `json:"email"`
	}
	if response.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&identity) != nil || identity.Subject == "" {
		return Credential{}, ErrInvalidGrant
	}
	return Credential{AccessToken: token.AccessToken, RefreshToken: token.RefreshToken, Subject: identity.Subject, Email: identity.Email}, nil
}

func (p *GoogleProvider) CreateFolder(ctx context.Context, accessToken, name string) (string, error) {
	body, _ := json.Marshal(map[string]string{"name": name, "mimeType": "application/vnd.google-apps.folder"})
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, p.config.DriveFilesEndpoint+"?fields=id", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Content-Type", "application/json")
	response, err := p.config.HTTPClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	var result struct {
		ID string `json:"id"`
	}
	if response.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result) != nil || result.ID == "" {
		return "", errors.New("drive folder could not be created")
	}
	return result.ID, nil
}

func (p *GoogleProvider) DownloadFile(ctx context.Context, accessToken, fileID, revision string) (File, error) {
	if fileID == "" || revision == "" {
		return File{}, ErrFileChanged
	}
	endpoint := strings.TrimRight(p.config.DriveFilesEndpoint, "/") + "/" + url.PathEscape(fileID)
	metadataRequest, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?fields=id,name,mimeType,headRevisionId,trashed", nil)
	metadataRequest.Header.Set("Authorization", "Bearer "+accessToken)
	metadataResponse, err := p.config.HTTPClient.Do(metadataRequest)
	if err != nil {
		return File{}, err
	}
	defer metadataResponse.Body.Close()
	var metadata struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		MIME     string `json:"mimeType"`
		Revision string `json:"headRevisionId"`
		Trashed  bool   `json:"trashed"`
	}
	if metadataResponse.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(metadataResponse.Body, 1<<20)).Decode(&metadata) != nil || metadata.ID != fileID || metadata.Revision != revision || metadata.Trashed || strings.HasPrefix(metadata.MIME, "application/vnd.google-apps.") {
		return File{}, ErrFileChanged
	}
	contentRequest, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?alt=media", nil)
	contentRequest.Header.Set("Authorization", "Bearer "+accessToken)
	contentResponse, err := p.config.HTTPClient.Do(contentRequest)
	if err != nil {
		return File{}, err
	}
	if contentResponse.StatusCode != http.StatusOK {
		contentResponse.Body.Close()
		return File{}, ErrFileChanged
	}
	return File{ID: metadata.ID, Name: metadata.Name, MIME: metadata.MIME, Revision: metadata.Revision, Body: contentResponse.Body}, nil
}

func (p *GoogleProvider) Refresh(ctx context.Context, refreshToken string) (string, error) {
	var token struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	err := p.postForm(ctx, p.config.TokenEndpoint, url.Values{
		"client_id": {p.config.ClientID}, "client_secret": {p.config.ClientSecret}, "grant_type": {"refresh_token"}, "refresh_token": {refreshToken},
	}, &token)
	if token.Error == "invalid_grant" {
		return "", ErrInvalidGrant
	}
	if err != nil || token.Error != "" || token.AccessToken == "" {
		return "", errors.New("drive token refresh failed")
	}
	return token.AccessToken, nil
}

func (p *GoogleProvider) Revoke(ctx context.Context, token string) error {
	return p.postForm(ctx, p.config.RevokeEndpoint, url.Values{"token": {token}}, nil)
}

func (p *GoogleProvider) postForm(ctx context.Context, endpoint string, values url.Values, target any) error {
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := p.config.HTTPClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if target != nil {
			_ = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(target)
		}
		return errors.New("google request failed")
	}
	if target == nil {
		return nil
	}
	return json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(target)
}
