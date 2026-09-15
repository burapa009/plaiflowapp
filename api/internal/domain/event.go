package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Event struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	OccurredAt    time.Time       `json:"occurred_at"`
	SubjectID     string          `json:"subject_id"`
	SourceEventID int64           `json:"source_event_id"`
	Data          json.RawMessage `json:"data"`
}

func (e Event) Validate() error {
	parts := strings.Split(e.Type, ".")
	if e.ID == "" || len(parts) != 2 || !strings.HasSuffix(parts[1], "ed") || e.OccurredAt.IsZero() || e.SubjectID == "" || e.SourceEventID < 1 || !json.Valid(e.Data) {
		return errors.New("invalid domain event envelope")
	}
	return nil
}
