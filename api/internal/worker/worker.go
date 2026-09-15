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
}

func RunOnce(ctx context.Context, store Store, logger *slog.Logger) error {
	events, err := store.Claim(ctx, 25, time.Minute)
	if err != nil {
		return err
	}
	for _, event := range events {
		outcome := inbound.OutcomeForType(event.Type)
		reason := ""
		if outcome == inbound.Ignored {
			reason = "No Phase 0 handler for this event type"
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
