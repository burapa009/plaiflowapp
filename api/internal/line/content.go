package line

import (
	"context"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type ContentDownloader interface {
	Download(context.Context, string) (io.ReadCloser, error)
}

type ContentClient struct {
	client  *http.Client
	baseURL string
	token   string
}

func NewContentClient(client *http.Client, baseURL, token string) (*ContentClient, error) {
	if token == "" {
		return nil, errors.New("LINE content token is missing")
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	if baseURL == "" {
		baseURL = "https://api-data.line.me"
	}
	return &ContentClient{client: client, baseURL: strings.TrimRight(baseURL, "/"), token: token}, nil
}

var contentIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

func (c *ContentClient) Download(ctx context.Context, messageID string) (io.ReadCloser, error) {
	if !contentIDPattern.MatchString(messageID) {
		return nil, errors.New("invalid LINE content id")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v2/bot/message/"+messageID+"/content", nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		response.Body.Close()
		return nil, errors.New("LINE content download failed")
	}
	return response.Body, nil
}
