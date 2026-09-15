package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"plaiflow/api/internal/work"
)

type fakeWorkStore struct {
	reminders, events, digests int
	delivery                   work.LINEDelivery
	failed                     bool
	retry                      bool
	delay                      time.Duration
}

func (s *fakeWorkStore) ScheduleReminders(context.Context, time.Time) (int, error) {
	return s.reminders, nil
}
func (s *fakeWorkStore) ProcessDomainEvents(context.Context, int, time.Duration) (int, error) {
	return s.events, nil
}
func (s *fakeWorkStore) ProcessDigests(context.Context, time.Time) (int, error) {
	return s.digests, nil
}
func (s *fakeWorkStore) ClaimLINEDeliveries(context.Context, int, time.Duration) ([]work.LINEDelivery, error) {
	return []work.LINEDelivery{s.delivery}, nil
}
func (*fakeWorkStore) CompleteLINEDelivery(context.Context, string, time.Time) error { return nil }
func (s *fakeWorkStore) FailLINEDelivery(_ context.Context, _ work.LINEDelivery, _ string, delay time.Duration, retry bool) error {
	s.failed = true
	s.retry = retry
	s.delay = delay
	return nil
}

type failingSender struct {
	sent work.LINEDelivery
	err  error
}

func (s *failingSender) Send(_ context.Context, delivery work.LINEDelivery) error {
	s.sent = delivery
	if s.err != nil {
		return s.err
	}
	return errors.New("provider unavailable")
}

func TestRunWorkOnceProcessesBoundedJobsAndRetriesLINE(t *testing.T) {
	store := &fakeWorkStore{reminders: 1, events: 2, delivery: work.LINEDelivery{ID: "delivery-1", OrganizationName: "Org", DeepLink: "/o/org/tasks/task"}}
	sender := &failingSender{}
	result, err := RunWorkOnce(context.Background(), store, sender, time.Date(2026, 9, 15, 9, 0, 0, 0, time.UTC))
	if err != nil || result.Reminders != 1 || result.Events != 2 || !store.failed || sender.sent.ID != "delivery-1" {
		t.Fatalf("result=%+v failed=%v sent=%+v err=%v", result, store.failed, sender.sent, err)
	}
}

func TestRunWorkOnceDoesNotRetryPermanentLINEFailure(t *testing.T) {
	store := &fakeWorkStore{delivery: work.LINEDelivery{ID: "delivery-1"}}
	_, err := RunWorkOnce(context.Background(), store, &failingSender{err: work.ErrPermanentDelivery}, time.Now())
	if err != nil || !store.failed || store.retry || store.delay != 0 {
		t.Fatalf("failed=%v retry=%v delay=%v err=%v", store.failed, store.retry, store.delay, err)
	}
}
