package line

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestParseSignedDelivery(t *testing.T) {
	body := []byte(`{"events":[{"webhookEventId":"evt-1","type":"message","timestamp":1700000000000,"message":{"type":"text","text":"secret"}}]}`)
	mac := hmac.New(sha256.New, []byte("channel-secret"))
	mac.Write(body)
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	events, err := ParseSignedDelivery(body, signature, "channel-secret", "channel-1")
	if err != nil || len(events) != 1 || events[0].ProviderEventID != "evt-1" || events[0].Type != "message" {
		t.Fatalf("unexpected delivery: events=%+v err=%v", events, err)
	}

	if _, err := ParseSignedDelivery(body, "forged", "channel-secret", "channel-1"); err != ErrInvalidSignature {
		t.Fatalf("forged signature error = %v", err)
	}
}

func TestEmptyDeliveryIsValid(t *testing.T) {
	body := []byte(`{"events":[]}`)
	mac := hmac.New(sha256.New, []byte("channel-secret"))
	mac.Write(body)

	events, err := ParseSignedDelivery(body, base64.StdEncoding.EncodeToString(mac.Sum(nil)), "channel-secret", "channel-1")
	if err != nil || len(events) != 0 {
		t.Fatalf("empty delivery: events=%+v err=%v", events, err)
	}
}
