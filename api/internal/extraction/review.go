package extraction

import (
	"context"
	"errors"
	"time"
)

var ErrConflict = errors.New("extraction revision changed")

type Review struct {
	ID             string            `json:"id"`
	OrganizationID string            `json:"organization_id"`
	DocumentID     string            `json:"document_id"`
	OCRJobID       string            `json:"ocr_job_id"`
	Revision       int               `json:"revision"`
	DraftRevision  int               `json:"-"`
	RequestID      string            `json:"-"`
	DocumentType   string            `json:"document_type"`
	Values         map[string]string `json:"values"`
	ConfirmedBy    string            `json:"confirmed_by"`
	ConfirmedAt    time.Time         `json:"confirmed_at"`
	ObjectKey      string            `json:"-"`
}

// ReviewDraft is a saved human proposal. It never grants confirmation or approval.
type ReviewDraft struct {
	OrganizationID string            `json:"-"`
	DocumentID     string            `json:"document_id"`
	OCRJobID       string            `json:"ocr_job_id"`
	Revision       int               `json:"revision"`
	Values         map[string]string `json:"values"`
	Decisions      map[string]string `json:"decisions"`
	UpdatedBy      string            `json:"updated_by"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

type QueueItem struct {
	DocumentID     string    `json:"document_id"`
	Status         string    `json:"status"`
	AcceptedAt     time.Time `json:"accepted_at"`
	OCRJobID       string    `json:"ocr_job_id"`
	ReviewRevision int       `json:"review_revision"`
	DraftRevision  int       `json:"draft_revision"`
	TaskID         string    `json:"task_id,omitempty"`
	AssigneeID     string    `json:"assignee_id,omitempty"`
}

type QueuePage struct {
	Items      []QueueItem `json:"items"`
	NextCursor string      `json:"next_cursor,omitempty"`
}

type Assignment struct {
	DocumentID     string
	OCRJobID       string
	ReviewRevision int
	DraftRevision  int
	TaskID         string
}

type ReturnInput struct {
	ID, ActorID, OrganizationID, DocumentID, OCRJobID, ReasonCode, PrivateNote, RequestID string
	ExpectedReviewRevision, ExpectedDraftRevision                                         int
	ReturnedAt                                                                            time.Time
}

type ReturnNotice struct {
	ReasonCode  string    `json:"reason_code"`
	PrivateNote string    `json:"private_note"`
	ReturnedAt  time.Time `json:"returned_at"`
}

type Store interface {
	CurrentReview(context.Context, string, string, string) (Review, error)
	SaveReview(context.Context, Review, int) (Review, error)
	ListCurrentReviews(context.Context, string, string, int) ([]Review, error)
	ListCurrentReviewsFiltered(context.Context, string, string, int, string) ([]Review, error)
	ListRecentReviews(context.Context, string, string, int, int) ([]Review, error)
	CurrentDraft(context.Context, string, string, string) (ReviewDraft, error)
	ListDraftHistory(context.Context, string, string, string, int) ([]ReviewDraft, error)
	SaveDraft(context.Context, ReviewDraft, int, string) (ReviewDraft, error)
	ListReviewQueue(context.Context, string, string, string, string, string, int) (QueuePage, error)
	AssignReviewTasks(context.Context, string, string, string, []Assignment, string, time.Time) ([]string, error)
	ReturnReview(context.Context, ReturnInput) (int, error)
	CurrentReturn(context.Context, string, string, string) (ReturnNotice, error)
}
