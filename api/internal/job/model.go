package job

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"plaiflow/api/internal/document"
	"plaiflow/api/internal/work"
)

var ErrLeaseLost = errors.New("job lease is no longer current")

type Kind string

const (
	Export Kind = "export"
	OCR    Kind = "ocr"
)

type Status string

const (
	Queued    Status = "Queued"
	Running   Status = "Running"
	Completed Status = "Completed"
	Ready     Status = "Ready"
	Expired   Status = "Expired"
	Failed    Status = "Failed"
	Cancelled Status = "Cancelled"
)

type Job struct {
	ID              string          `json:"id"`
	OrganizationID  string          `json:"organization_id"`
	RequesterUserID string          `json:"requester_user_id,omitempty"`
	Kind            Kind            `json:"kind"`
	Status          Status          `json:"status"`
	Payload         json.RawMessage `json:"payload,omitempty"`
	AttemptID       string          `json:"attempt_id,omitempty"`
	AttemptCount    int             `json:"attempt_count"`
	CreatedAt       time.Time       `json:"created_at,omitempty"`
}

type ClaimCommand struct {
	WorkerID    string
	Environment string
	Kinds       []Kind
	Limit       int
	Lease       time.Duration
	Now         time.Time
}

type Claimed struct {
	Job            Job       `json:"job"`
	LeaseToken     string    `json:"lease_token"`
	LeaseExpiresAt time.Time `json:"lease_expires_at"`
}

type LeaseCommand struct {
	JobID, AttemptID, LeaseToken string
	Now                          time.Time
}

type FailureCommand struct {
	LeaseCommand
	Code       string
	RetryAfter time.Duration
}

type ExportPage struct {
	Rows       []work.ExportRow      `json:"rows"`
	Product    string                `json:"product,omitempty"`
	Header     []string              `json:"header,omitempty"`
	Values     [][]string            `json:"values,omitempty"`
	Entries    []DocumentExportEntry `json:"-"`
	NextCursor string                `json:"next_cursor,omitempty"`
	Done       bool                  `json:"done"`
}

type DocumentExportRequest struct {
	ID, OrganizationID, RequesterUserID, Product, Status, Format, DateFrom, DateTo, RequestID string
	Now                                                                                       time.Time
}

type DocumentExportState struct {
	ID          string    `json:"id"`
	Product     string    `json:"product"`
	Status      Status    `json:"status"`
	RowCount    int64     `json:"row_count"`
	FailureCode string    `json:"failure_code,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type DocumentExportStore interface {
	CountDocumentExport(context.Context, string, string, string, string, string, string) (int64, error)
	QueueDocumentExport(context.Context, DocumentExportRequest) (DocumentExportState, error)
	GetDocumentExport(context.Context, string, string, string) (DocumentExportState, error)
}

type DocumentExportEntry struct {
	Document document.ExportRow
	Review   ReviewExportMeta
	Approval ApprovalExportMeta
}

type ReviewExportMeta struct {
	ID, OrganizationID, DocumentID, OCRJobID, ObjectKey, ConfirmedBy string
	Revision                                                         int
	ConfirmedAt                                                      time.Time
}

type ApprovalExportMeta struct {
	ID, DocumentID, ReviewID, CategoryID, CategoryName, VendorID, ContactCode, Basis string
	Revision, ReviewRevision, RuleVersion                                            int
	ApprovedAt                                                                       time.Time
}

type ExportArtifact struct {
	JobID, OrganizationID, ObjectKey, Format, SHA256 string
	Product                                          string
	RowCount, ByteCount                              int64
	ExpiresAt                                        time.Time
}

type CompleteExportCommand struct {
	LeaseCommand
	Format              string
	RowCount, ByteCount int64
	SHA256, ObjectKey   string
	ExpiresAt           time.Time
}

type Store interface {
	ClaimJobs(context.Context, ClaimCommand) ([]Claimed, error)
	HeartbeatJob(context.Context, LeaseCommand, time.Duration) (time.Time, error)
	FailJob(context.Context, FailureCommand) error
	ReadExportPage(context.Context, LeaseCommand, string, int) (ExportPage, error)
	CompleteExport(context.Context, CompleteExportCommand) (ExportArtifact, error)
	AuthorizeExportArtifact(context.Context, string, string, string) (ExportArtifact, error)
	RecordArtifactDownload(context.Context, string, string, string, time.Time) error
}

type ArtifactStore interface {
	Put(context.Context, string, io.Reader) error
	Open(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}
