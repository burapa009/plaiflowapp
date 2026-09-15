package line

import (
	"bytes"
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

func TestGroupLinkCodeIsHashedAndRedactedBeforePersistence(t *testing.T) {
	body := []byte(`{"events":[{"webhookEventId":"link-1","type":"message","timestamp":1700000000000,"source":{"type":"group","groupId":"group-1","userId":"line-user-1"},"message":{"type":"text","text":"เชื่อม AbCdEf123456789_-"}}]}`)
	mac := hmac.New(sha256.New, []byte("channel-secret"))
	mac.Write(body)
	events, err := ParseSignedDelivery(body, base64.StdEncoding.EncodeToString(mac.Sum(nil)), "channel-secret", "channel-1")
	if err != nil || len(events) != 1 {
		t.Fatalf("events=%+v err=%v", events, err)
	}
	digest := sha256.Sum256([]byte("AbCdEf123456789_-"))
	if events[0].SourceType != "group" || events[0].SourceGroupID != "group-1" || events[0].SourceUserID != "line-user-1" || !bytes.Equal(events[0].LinkCodeHash, digest[:]) {
		t.Fatalf("link event = %+v", events[0])
	}
	if bytes.Contains(events[0].Payload, []byte("AbCdEf123456789_-")) || !bytes.Contains(events[0].Payload, []byte("[REDACTED_LINK_CODE]")) {
		t.Fatalf("payload was not redacted: %s", events[0].Payload)
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
