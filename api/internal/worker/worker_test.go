package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"plaiflow/api/internal/inbound"
)

type fakeStore struct {
	event     inbound.Event
	completed inbound.Status
	failed    bool
}

func (s *fakeStore) Claim(context.Context, int, time.Duration) ([]inbound.Event, error) {
	return []inbound.Event{s.event}, nil
}
func (s *fakeStore) Complete(_ context.Context, _ int64, status inbound.Status, _ string) error {
	s.completed = status
	return nil
}
func (s *fakeStore) Fail(context.Context, inbound.Event, string, string) error {
	s.failed = true
	return nil
}

func TestRunOnceCompletesOrRetries(t *testing.T) {
	store := &fakeStore{event: inbound.Event{ID: 1, Type: "message", OccurredAt: time.Now()}}
	if err := RunOnce(context.Background(), store, nil, RecordOnly); err != nil || store.completed != inbound.Processed {
		t.Fatalf("completed=%s err=%v", store.completed, err)
	}
	store.completed = ""
	if err := RunOnce(context.Background(), store, nil, func(context.Context, inbound.Event) (inbound.Status, string, error) {
		return "", "", errors.New("temporary")
	}); err != nil || !store.failed || store.completed != "" {
		t.Fatalf("failed=%v completed=%s err=%v", store.failed, store.completed, err)
	}
}
