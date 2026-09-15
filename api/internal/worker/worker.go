package worker

import (
	"context"
	"log/slog"
	"time"

	"plaiflow/api/internal/inbound"
)

type Store interface {
	Claim(context.Context, int, time.Duration) ([]inbound.Event, error)
	Complete(context.Context, int64, inbound.Status, string) error
	Fail(context.Context, inbound.Event, string, string) error
}

type ProcessFunc func(context.Context, inbound.Event) (inbound.Status, string, error)

func RecordOnly(_ context.Context, event inbound.Event) (inbound.Status, string, error) {
	outcome := inbound.OutcomeForType(event.Type)
	if outcome == inbound.Ignored {
		return outcome, "No Phase 0 handler for this event type", nil
	}
	return outcome, "", nil
}

func RunOnce(ctx context.Context, store Store, logger *slog.Logger, process ProcessFunc) error {
	if logger == nil {
		logger = slog.Default()
	}
	events, err := store.Claim(ctx, 25, time.Minute)
	if err != nil {
		return err
	}
	for _, event := range events {
		outcome, reason, err := process(ctx, event)
		if err != nil {
			if err := store.Fail(ctx, event, "processing_failed", "Event processing failed"); err != nil {
				return err
			}
			continue
		}
		if err := store.Complete(ctx, event.ID, outcome, reason); err != nil {
			return err
		}
		queueAge := time.Since(event.OccurredAt)
		if queueAge > time.Minute {
			logger.Warn("queue_age_high", "event_id", event.ID, "queue_age_ms", queueAge.Milliseconds())
		}
	}
	return nil
}
