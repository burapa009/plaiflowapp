package tenant

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("organization resource not found")
	ErrForbidden     = errors.New("organization action is forbidden")
	ErrConflict      = errors.New("organization action conflicts with current state")
	ErrInvalidInvite = errors.New("invitation is unavailable")
)

type Role string

const (
	Owner  Role = "Owner"
	Admin  Role = "Admin"
	Member Role = "Member"
)

type Action string

const (
	EditOrganization  Action = "edit_organization"
	InviteMember      Action = "invite_member"
	RemoveMember      Action = "remove_member"
	ManageRoles       Action = "manage_roles"
	TransferOwnership Action = "transfer_ownership"
	ManageGroup       Action = "manage_group"
)

func (r Role) Allows(action Action) bool {
	switch r {
	case Owner:
		return true
	case Admin:
		return action == EditOrganization || action == InviteMember || action == RemoveMember || action == ManageGroup
	default:
		return false
	}
}

type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role Role   `json:"role"`
}

type BusinessProfile struct {
	BusinessType string `json:"business_type"`
	VATStatus    string `json:"vat_status"`
	BranchType   string `json:"branch_type"`
	NameTH       string `json:"name_th"`
	NameEN       string `json:"name_en"`
	TaxID        string `json:"tax_id"`
	Address1     string `json:"address_1"`
	Address2     string `json:"address_2"`
	District     string `json:"district"`
	Province     string `json:"province"`
	PostalCode   string `json:"postal_code"`
	Phone        string `json:"phone"`
	HasLogo      bool   `json:"has_logo"`
	Logo         []byte `json:"-"`
	LogoType     string `json:"-"`
}

type Membership struct {
	OrganizationID   string    `json:"organization_id"`
	OrganizationName string    `json:"organization_name,omitempty"`
	UserID           string    `json:"user_id"`
	DisplayName      string    `json:"display_name,omitempty"`
	Role             Role      `json:"role"`
	CreatedAt        time.Time `json:"created_at"`
}

type Invitation struct {
	ID               string    `json:"id"`
	OrganizationID   string    `json:"organization_id"`
	OrganizationName string    `json:"organization_name"`
	InviterName      string    `json:"inviter_name"`
	Role             Role      `json:"role"`
	ExpiresAt        time.Time `json:"expires_at"`
}

type PendingInvitation struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
}

type LineConnection struct {
	ID        string     `json:"id"`
	GroupID   string     `json:"group_id"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"connected_at"`
	EndedAt   *time.Time `json:"disconnected_at,omitempty"`
}

type InviteCreate struct {
	ID, OrganizationID, ActorUserID string
	TokenHash                       []byte
	Now, ExpiresAt                  time.Time
}

type InviteClaim struct {
	TokenHash, HandoffHash []byte
	HandoffID, ReturnTo    string
	Now, ExpiresAt         time.Time
}

type LineCodeCreate struct {
	ID, OrganizationID, ActorUserID, Channel string
	CodeHash                                 []byte
	Now, ExpiresAt                           time.Time
}

type Store interface {
	GetBusinessProfile(context.Context, string, string) (BusinessProfile, error)
	UpdateBusinessProfile(context.Context, string, string, BusinessProfile, time.Time) error
	CreateOrganization(context.Context, string, string, string, time.Time) (Organization, error)
	ListOrganizations(context.Context, string) ([]Organization, error)
	ResolveMembership(context.Context, string, string) (Membership, error)
	ListMemberships(context.Context, string, string) ([]Membership, error)
	ListPendingInvitations(context.Context, string, string) ([]PendingInvitation, error)
	CreateInvitation(context.Context, InviteCreate) error
	ClaimInvitation(context.Context, InviteClaim) (Invitation, error)
	AcceptInvitation(context.Context, []byte, string, time.Time) (Organization, error)
	RevokeInvitation(context.Context, string, string, string, time.Time) error
	ChangeRole(context.Context, string, string, string, Role, time.Time) error
	TransferOwnership(context.Context, string, string, string, time.Time) error
	RemoveMembership(context.Context, string, string, string, time.Time) error
	LeaveOrganization(context.Context, string, string, time.Time) error
	CreateLineLinkCode(context.Context, LineCodeCreate) error
	ListLineConnections(context.Context, string, string) ([]LineConnection, error)
	DisconnectLineConnection(context.Context, string, string, string, time.Time) error
}
