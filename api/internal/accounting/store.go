package accounting

import (
	"context"
	"errors"
	"time"

	"plaiflow/api/internal/extraction"
)

var (
	ErrConflict = errors.New("accounting review changed")
	ErrInvalid  = errors.New("accounting choice is invalid")
	ErrTooMany  = errors.New("accounting export exceeds synchronous limit")
)

type Evaluation struct {
	Candidate      Candidate `json:"candidate"`
	VendorID       string    `json:"vendor_id,omitempty"`
	RuleSetVersion int       `json:"rule_set_version"`
	Approved       *Approval `json:"approved,omitempty"`
	Status         string    `json:"status"`
}

type ApproveInput struct {
	ID               string
	ActorID          string
	OrganizationID   string
	DocumentID       string
	ReviewID         string
	ReviewRevision   int
	RuleSetVersion   int
	CategoryID       string
	VendorID         string
	RuleID           string
	RuleVersion      int
	SuggestionBasis  string
	UnmatchedReason  string
	ExpectedRevision int
	ApprovedAt       time.Time
	RequestID        string
}

type RuleInput struct {
	ID             string
	ActorID        string
	OrganizationID string
	VendorID       string
	DocumentType   string
	CategoryID     string
	ApprovedAt     time.Time
}

type Store interface {
	ReadyAccounting(context.Context) error
	ListCategories(context.Context, string, string) ([]Category, error)
	CreateCategory(context.Context, string, string, Category, time.Time) (Category, error)
	RenameCategory(context.Context, string, string, string, string, time.Time) error
	SetCategoryArchived(context.Context, string, string, string, bool, time.Time) error
	Evaluate(context.Context, string, string, extraction.Review) (Evaluation, error)
	Approve(context.Context, ApproveInput) (Approval, error)
	ListApproved(context.Context, string, string, int, string) ([]Approval, error)
	ValidateApprovedSnapshot(context.Context, string, string, []Approval) error
	ListRules(context.Context, string, string, string) ([]Rule, error)
	CreateRule(context.Context, RuleInput) (Rule, error)
	RetireRule(context.Context, string, string, string, time.Time) error
}
