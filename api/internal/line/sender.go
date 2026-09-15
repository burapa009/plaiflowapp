package line

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"plaiflow/api/internal/work"
)

type Sender struct {
	client                  *http.Client
	apiBase, token, webBase string
}

func NewSender(client *http.Client, apiBase, token, webBase string) *Sender {
	if client == nil {
		client = http.DefaultClient
	}
	return &Sender{client: client, apiBase: strings.TrimRight(apiBase, "/"), token: token, webBase: strings.TrimRight(webBase, "/")}
}

func (s *Sender) Send(ctx context.Context, delivery work.LINEDelivery) error {
	if delivery.To == "" {
		return fmt.Errorf("%w: recipient unavailable", work.ErrPermanentDelivery)
	}
	organization := strings.Join(strings.Fields(delivery.OrganizationName), " ")
	payload := map[string]any{
		"to":       delivery.To,
		"messages": []map[string]string{{"type": "text", "text": "มีการแจ้งเตือนใหม่จาก " + organization + "\nเปิด PlaiFlow: " + s.webBase + delivery.DeepLink}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiBase+"/v2/bot/message/push", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+s.token)
	request.Header.Set("Content-Type", "application/json")
	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
	if response.StatusCode >= 400 && response.StatusCode < 500 && response.StatusCode != http.StatusRequestTimeout && response.StatusCode != http.StatusTooManyRequests {
		return fmt.Errorf("%w: LINE delivery failed", work.ErrPermanentDelivery)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return errors.New("LINE delivery failed")
	}
	return nil
}
