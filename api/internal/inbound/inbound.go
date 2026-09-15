package inbound

import (
	"encoding/json"
	"time"
)

type Status string

const (
	Received   Status = "Received"
	Processing Status = "Processing"
	Processed  Status = "Processed"
	Retryable  Status = "Retryable"
	Failed     Status = "Failed"
	Ignored    Status = "Ignored"
)

type Event struct {
	ID              int64
	Provider        string
	Channel         string
	ProviderEventID string
	Type            string
	Payload         json.RawMessage
	SourceType      string
	SourceGroupID   string
	SourceUserID    string
	LinkCodeHash    []byte
	OccurredAt      time.Time
	AttemptCount    int
}

type Counts struct {
	Received  int64 `json:"received"`
	Processed int64 `json:"processed"`
	Ignored   int64 `json:"ignored"`
	Retryable int64 `json:"retryable"`
	Failed    int64 `json:"failed"`
}

type Snapshot struct {
	Counts   Counts `json:"counts"`
	API      string `json:"api"`
	Database string `json:"database"`
	Worker   string `json:"worker"`
}

func OutcomeForType(eventType string) Status {
	if eventType == "message" {
		return Processed
	}
	return Ignored
}

func RetryDelay(attempt int) (time.Duration, bool) {
	delays := [...]time.Duration{time.Minute, 5 * time.Minute, 30 * time.Minute, 2 * time.Hour, 12 * time.Hour}
	if attempt < 1 || attempt > len(delays) {
		return 0, false
	}
	return delays[attempt-1], true
}
