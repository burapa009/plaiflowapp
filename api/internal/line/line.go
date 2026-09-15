package line

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"plaiflow/api/internal/inbound"
)

var (
	ErrInvalidSignature = errors.New("invalid signature")
	ErrMalformed        = errors.New("malformed delivery")
)

type delivery struct {
	Events []struct {
		WebhookEventID string          `json:"webhookEventId"`
		Type           string          `json:"type"`
		Timestamp      int64           `json:"timestamp"`
		Message        json.RawMessage `json:"message,omitempty"`
	} `json:"events"`
}

func ParseSignedDelivery(body []byte, signature, secret, channel string) ([]inbound.Event, error) {
	provided, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return nil, ErrInvalidSignature
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	if !hmac.Equal(mac.Sum(nil), provided) {
		return nil, ErrInvalidSignature
	}

	var raw delivery
	if err := json.Unmarshal(body, &raw); err != nil || raw.Events == nil {
		return nil, ErrMalformed
	}
	events := make([]inbound.Event, 0, len(raw.Events))
	for _, event := range raw.Events {
		if event.WebhookEventID == "" || event.Type == "" {
			return nil, ErrMalformed
		}
		payload, err := json.Marshal(event)
		if err != nil {
			return nil, ErrMalformed
		}
		events = append(events, inbound.Event{
			Provider: "line", Channel: channel, ProviderEventID: event.WebhookEventID,
			Type: event.Type, Payload: payload, OccurredAt: time.UnixMilli(event.Timestamp).UTC(),
		})
	}
	return events, nil
}
