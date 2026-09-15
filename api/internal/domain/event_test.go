package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEventEnvelopeValidationAndSerialization(t *testing.T) {
	event := Event{ID: "dom-1", Type: "invoice.created", OccurredAt: time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC), SubjectID: "invoice-1", SourceEventID: 1, Data: json.RawMessage(`{"opaque":true}`)}
	if err := event.Validate(); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(event)
	if err != nil || !json.Valid(data) {
		t.Fatalf("serialization: %s, %v", data, err)
	}
	event.Type = "invoice.create"
	if err := event.Validate(); err == nil {
		t.Fatal("non-past-tense event type accepted")
	}
}
