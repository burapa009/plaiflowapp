package firm

import (
	"context"
	"time"

	"plaiflow/api/internal/document"
)

type Grant struct {
	ID                   string     `json:"id"`
	FirmOrganizationID   string     `json:"firm_organization_id"`
	ClientOrganizationID string     `json:"client_organization_id"`
	PartnerName          string     `json:"partner_name"`
	Status               string     `json:"status"`
	Scopes               []string   `json:"scopes"`
	Revision             int        `json:"revision"`
	RequestExpiresAt     time.Time  `json:"request_expires_at"`
	ActiveExpiresAt      *time.Time `json:"active_expires_at,omitempty"`
}

type Assignment struct {
	UserID     string    `json:"user_id"`
	DisplayName string   `json:"display_name"`
	AssignedAt time.Time `json:"assigned_at"`
}

type PortfolioClient struct {
	GrantID              string     `json:"grant_id"`
	ClientOrganizationID string     `json:"client_organization_id"`
	ClientName           string     `json:"client_name"`
	Assigned             bool       `json:"assigned"`
	StaffCount           int        `json:"staff_count,omitempty"`
	ActiveExpiresAt      *time.Time `json:"active_expires_at,omitempty"`
}

type PortfolioPage struct {
	Clients    []PortfolioClient `json:"clients"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

type Store interface {
	ReadyFirm(context.Context) error
	RequestGrant(context.Context, string, string, string, string, time.Time) (Grant, error)
	ListGrants(context.Context, string, string) ([]Grant, error)
	TransitionGrant(context.Context, string, string, string, string, time.Time) error
	AssignStaff(context.Context, string, string, string, string, time.Time) error
	RemoveStaff(context.Context, string, string, string, string, time.Time) error
	ListStaff(context.Context, string, string, string) ([]Assignment, error)
	GetFirmDocument(context.Context, string, string, string, string, string) (document.Document, error)
	ListPortfolio(context.Context, string, string, string) (PortfolioPage, error)
}
