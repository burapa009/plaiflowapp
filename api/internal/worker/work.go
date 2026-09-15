package worker

import (
	"context"
	"errors"
	"sync"
	"time"

	"plaiflow/api/internal/work"
)

type WorkStore interface {
	ScheduleReminders(context.Context, time.Time) (int, error)
	ProcessDomainEvents(context.Context, int, time.Duration) (int, error)
	ProcessDigests(context.Context, time.Time) (int, error)
	ClaimLINEDeliveries(context.Context, int, time.Duration) ([]work.LINEDelivery, error)
	CompleteLINEDelivery(context.Context, string, time.Time) error
	FailLINEDelivery(context.Context, work.LINEDelivery, string, time.Duration, bool) error
}

type LINESender interface {
	Send(context.Context, work.LINEDelivery) error
}

type WorkResult struct {
	Reminders, Events, Digests, LINEClaimed int
}

func RunWorkOnce(ctx context.Context, store WorkStore, sender LINESender, now time.Time) (WorkResult, error) {
	var result WorkResult
	var err error
	if result.Reminders, err = store.ScheduleReminders(ctx, now); err != nil {
		return result, err
	}
	if result.Events, err = store.ProcessDomainEvents(ctx, 100, time.Minute); err != nil {
		return result, err
	}
	if result.Digests, err = store.ProcessDigests(ctx, now); err != nil {
		return result, err
	}
	deliveries, err := store.ClaimLINEDeliveries(ctx, 100, time.Minute)
	if err != nil {
		return result, err
	}
	result.LINEClaimed = len(deliveries)
	semaphore := make(chan struct{}, 10)
	errorsFound := make(chan error, len(deliveries))
	var group sync.WaitGroup
	for _, delivery := range deliveries {
		delivery := delivery
		group.Add(1)
		go func() {
			defer group.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			if err := sender.Send(ctx, delivery); err != nil {
				delay, retry := lineRetry(delivery.AttemptCount + 1)
				if errors.Is(err, work.ErrPermanentDelivery) {
					delay, retry = 0, false
				}
				errorsFound <- store.FailLINEDelivery(ctx, delivery, "provider_failed", delay, retry)
				return
			}
			errorsFound <- store.CompleteLINEDelivery(ctx, delivery.ID, now)
		}()
	}
	group.Wait()
	close(errorsFound)
	for err := range errorsFound {
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func lineRetry(attempt int) (time.Duration, bool) {
	delays := []time.Duration{time.Minute, 5 * time.Minute, 30 * time.Minute, 2 * time.Hour, 12 * time.Hour}
	if attempt < 1 || attempt > len(delays) {
		return 0, false
	}
	return delays[attempt-1], true
}
