package inbound

import (
	"testing"
	"time"
)

func TestOutcomeAndRetrySchedule(t *testing.T) {
	if OutcomeForType("message") != Processed || OutcomeForType("follow") != Ignored {
		t.Fatal("unexpected Phase 0 outcomes")
	}
	want := []time.Duration{time.Minute, 5 * time.Minute, 30 * time.Minute, 2 * time.Hour, 12 * time.Hour}
	for attempt, expected := range want {
		got, retry := RetryDelay(attempt + 1)
		if !retry || got != expected {
			t.Fatalf("attempt %d: got %v, %v", attempt+1, got, retry)
		}
	}
	if _, retry := RetryDelay(6); retry {
		t.Fatal("sixth failure must be terminal")
	}
}
