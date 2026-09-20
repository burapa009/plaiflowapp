package line

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"plaiflow/api/internal/inbound"
)

var (
	ErrInvalidSignature = errors.New("invalid signature")
	ErrMalformed        = errors.New("malformed delivery")
)

type delivery struct {
	Events []struct {
		WebhookEventID string `json:"webhookEventId"`
		Type           string `json:"type"`
		Timestamp      int64  `json:"timestamp"`
		Source         struct {
			Type    string `json:"type"`
			GroupID string `json:"groupId,omitempty"`
			UserID  string `json:"userId,omitempty"`
		} `json:"source,omitempty"`
		Message struct {
			Type     string `json:"type,omitempty"`
			Text     string `json:"text,omitempty"`
			ID       string `json:"id,omitempty"`
			FileName string `json:"fileName,omitempty"`
		} `json:"message,omitempty"`
	} `json:"events"`
}

type DocumentMessage struct {
	Type, ID, FileName string
}

func ParseDocumentMessage(payload []byte) (DocumentMessage, bool) {
	var event struct {
		Type    string `json:"type"`
		Message struct {
			Type     string `json:"type"`
			ID       string `json:"id"`
			FileName string `json:"fileName"`
		} `json:"message"`
	}
	if json.Unmarshal(payload, &event) != nil || event.Type != "message" || event.Message.ID == "" {
		return DocumentMessage{}, false
	}
	if event.Message.Type != "image" && event.Message.Type != "file" {
		return DocumentMessage{}, false
	}
	return DocumentMessage{Type: event.Message.Type, ID: event.Message.ID, FileName: event.Message.FileName}, true
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
		var linkCodeHash []byte
		if event.Source.Type == "group" && event.Type == "message" && event.Message.Type == "text" {
			if code, ok := linkCodeCandidate(event.Message.Text); ok {
				digest := sha256.Sum256([]byte(code))
				linkCodeHash = digest[:]
				event.Message.Text = "[REDACTED_LINK_CODE]"
			}
		}
		payload, err := json.Marshal(event)
		if err != nil {
			return nil, ErrMalformed
		}
		events = append(events, inbound.Event{
			Provider: "line", Channel: channel, ProviderEventID: event.WebhookEventID,
			Type: event.Type, Payload: payload, SourceType: event.Source.Type, SourceGroupID: event.Source.GroupID,
			SourceUserID: event.Source.UserID, LinkCodeHash: linkCodeHash, OccurredAt: time.UnixMilli(event.Timestamp).UTC(),
		})
	}
	return events, nil
}

func linkCodeCandidate(text string) (string, bool) {
	code := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), "เชื่อม "))
	if code == text || len(code) < 16 || len(code) > 128 {
		return "", false
	}
	for _, character := range code {
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '-' && character != '_' {
			return "", false
		}
	}
	return code, true
}
