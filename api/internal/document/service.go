package document

import (
	"context"
	"errors"
	"io"
	"time"
)

var (
	ErrQuota          = errors.New("document quota exhausted")
	ErrTrashed        = errors.New("matching document is in trash")
	ErrInvalidCursor  = errors.New("document cursor is invalid")
	ErrStatusConflict = errors.New("document status transition is unavailable")
)

type Document struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Filename       string    `json:"filename"`
	MIME           string    `json:"mime"`
	Size           int64     `json:"size"`
	Status         string    `json:"status"`
	StorageKey     string    `json:"-"`
	AcceptedAt     time.Time `json:"accepted_at"`
	SourceCount    int       `json:"source_count,omitempty"`
	SourceChannel  string    `json:"source_channel,omitempty"`
}

type Page struct {
	Documents  []Document `json:"documents"`
	NextCursor string     `json:"next_cursor,omitempty"`
}

type Summary struct {
	Used      int64     `json:"used"`
	Limit     int64     `json:"limit"`
	ResetAt   time.Time `json:"reset_at"`
	Available int64     `json:"available"`
	Archived  int64     `json:"archived"`
	Trash     int64     `json:"trash"`
	Checking  int64     `json:"checking"`
	Rejected  int64     `json:"rejected"`
	Warning   string    `json:"warning,omitempty"`
}

type Filter struct {
	From, To  *time.Time
	Status    string
	Channel   string
	Filename  string
	Submitter string
	Assignee  string
}

type ExportRow struct {
	ID, Filename, MIME, Status, SourceChannel, SubmitterName, AssigneeName string
	Size, SourceCount                                                      int64
	AcceptedAt, UpdatedAt                                                  time.Time
}

type Exporter interface {
	ExportDocuments(context.Context, string, string, Filter, string, io.Writer) (int64, error)
}

type ExportCounter interface {
	CountDocumentsForExport(context.Context, string, string, Filter) (int64, error)
}

type Reader interface {
	ListDocuments(context.Context, string, string, string, int) (Page, error)
	GetDocument(context.Context, string, string, string) (Document, error)
}

type FilterReader interface {
	ListDocumentsFiltered(context.Context, string, string, string, int, Filter) (Page, error)
}

type SummaryReader interface {
	DocumentSummary(context.Context, string, string) (Summary, error)
}

type StatusUpdater interface {
	ChangeDocumentStatus(context.Context, string, string, string, string, time.Time) (Document, error)
}

type AcceptInput struct {
	OrganizationID    string
	ActorUserID       string
	AttemptID         string
	OriginKey         string
	Filename          string
	Channel           string
	Now               time.Time
	DriveConnectionID string
	DriveFileID       string
	DriveRevision     string
}

type CommitInput struct {
	AcceptInput
	TemporaryKey string
	SHA256       string
	MIME         string
	Size         int64
}

type CommitResult struct {
	Document  Document `json:"document"`
	Accepted  bool     `json:"accepted"`
	Duplicate bool     `json:"duplicate"`
}

type Committer interface {
	CommitPrepared(context.Context, CommitInput) (CommitResult, error)
}

type Rejector interface {
	RecordRejected(context.Context, AcceptInput, string) error
}

type Service struct {
	Intake    Intake
	Committer Committer
	Reader    Reader
}

func (s Service) Accept(ctx context.Context, input AcceptInput, body io.Reader) (CommitResult, error) {
	if s.Committer == nil || input.OrganizationID == "" || input.ActorUserID == "" || input.AttemptID == "" || input.OriginKey == "" || input.Filename == "" || input.Channel == "" {
		return CommitResult{}, errors.New("document intake is not configured")
	}
	prepared, err := s.Intake.Prepare(ctx, input.OrganizationID, input.AttemptID, body)
	if err != nil {
		if code := rejectionCode(err); code != "" {
			if rejector, ok := s.Committer.(Rejector); ok {
				_ = rejector.RecordRejected(ctx, input, code)
			}
		}
		return CommitResult{}, err
	}
	result, err := s.Committer.CommitPrepared(ctx, CommitInput{AcceptInput: input, TemporaryKey: prepared.TemporaryKey,
		SHA256: prepared.SHA256, MIME: prepared.MIME, Size: prepared.Size})
	if err != nil || !result.Accepted {
		if cleanupErr := s.Intake.Temporary.Delete(ctx, prepared.TemporaryKey); cleanupErr != nil && err == nil {
			return CommitResult{}, cleanupErr
		}
	}
	return result, err
}

func rejectionCode(err error) string {
	switch {
	case errors.Is(err, ErrUnsupportedType):
		return "unsupported_type"
	case errors.Is(err, ErrTooLarge):
		return "too_large"
	case errors.Is(err, ErrMalware):
		return "malware"
	default:
		return ""
	}
}

func (s Service) List(ctx context.Context, userID, organizationID, cursor string, limit int) (Page, error) {
	if s.Reader == nil {
		return Page{}, errors.New("document reader is not configured")
	}
	return s.Reader.ListDocuments(ctx, userID, organizationID, cursor, limit)
}

func (s Service) ListFiltered(ctx context.Context, userID, organizationID, cursor string, limit int, filter Filter) (Page, error) {
	if reader, ok := s.Reader.(FilterReader); ok {
		return reader.ListDocumentsFiltered(ctx, userID, organizationID, cursor, limit, filter)
	}
	return s.List(ctx, userID, organizationID, cursor, limit)
}

func (s Service) Summary(ctx context.Context, userID, organizationID string) (Summary, error) {
	if reader, ok := s.Reader.(SummaryReader); ok {
		return reader.DocumentSummary(ctx, userID, organizationID)
	}
	return Summary{}, errors.New("document summary is not configured")
}

func (s Service) ChangeStatus(ctx context.Context, userID, organizationID, documentID, action string, now time.Time) (Document, error) {
	if updater, ok := s.Reader.(StatusUpdater); ok {
		return updater.ChangeDocumentStatus(ctx, userID, organizationID, documentID, action, now)
	}
	return Document{}, errors.New("document status is not configured")
}

func (s Service) Export(ctx context.Context, userID, organizationID string, filter Filter, format string, output io.Writer) (int64, error) {
	if exporter, ok := s.Reader.(Exporter); ok {
		return exporter.ExportDocuments(ctx, userID, organizationID, filter, format, output)
	}
	return 0, errors.New("document exporter is not configured")
}

func (s Service) ExportCount(ctx context.Context, userID, organizationID string, filter Filter) (int64, error) {
	if counter, ok := s.Reader.(ExportCounter); ok {
		return counter.CountDocumentsForExport(ctx, userID, organizationID, filter)
	}
	return 0, errors.New("document export preview is not configured")
}

func (s Service) Open(ctx context.Context, userID, organizationID, documentID string) (Document, io.ReadCloser, error) {
	if s.Reader == nil || s.Intake.Temporary == nil {
		return Document{}, nil, errors.New("document reader is not configured")
	}
	doc, err := s.Reader.GetDocument(ctx, userID, organizationID, documentID)
	if err != nil {
		return Document{}, nil, err
	}
	body, err := s.Intake.Temporary.Open(ctx, doc.StorageKey)
	return doc, body, err
}
