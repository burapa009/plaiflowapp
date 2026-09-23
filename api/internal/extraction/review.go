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
	DocumentType   string            `json:"document_type"`
	Values         map[string]string `json:"values"`
	ConfirmedBy    string            `json:"confirmed_by"`
	ConfirmedAt    time.Time         `json:"confirmed_at"`
	ObjectKey      string            `json:"-"`
}

type Store interface {
	CurrentReview(context.Context, string, string, string) (Review, error)
	SaveReview(context.Context, Review, int) (Review, error)
	ListCurrentReviews(context.Context, string, string, int) ([]Review, error)
	ListRecentReviews(context.Context, string, string, int, int) ([]Review, error)
}
