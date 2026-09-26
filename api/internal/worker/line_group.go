package worker

import (
	"context"

	"plaiflow/api/internal/inbound"
)

type LINEGroupLinker interface {
	ConsumeLINEGroupCode(context.Context, inbound.Event) (bool, error)
	DisconnectLINEGroupBySource(context.Context, inbound.Event) (bool, error)
}

func NewLINEGroupProcessor(next ProcessFunc, linker LINEGroupLinker) ProcessFunc {
	return func(ctx context.Context, event inbound.Event) (inbound.Status, string, error) {
		if event.Provider == "line" && event.SourceType == "group" && event.Type == "leave" {
			_, err := linker.DisconnectLINEGroupBySource(ctx, event)
			if err != nil {
				return inbound.Retryable, "LINE group leave failed", err
			}
			return inbound.Processed, "LINE bot left group", nil
		}
		if len(event.LinkCodeHash) == 0 {
			return next(ctx, event)
		}
		linked, err := linker.ConsumeLINEGroupCode(ctx, event)
		if err != nil {
			return inbound.Retryable, "LINE group linking failed", err
		}
		if !linked {
			return inbound.Ignored, "LINE group link code is invalid or unavailable", nil
		}
		return inbound.Processed, "LINE group connected", nil
	}
}
