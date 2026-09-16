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
	CreateOrganization(context.Context, string, string, string, time.Time) (Organization, error)
	ListOrganizations(context.Context, string) ([]Organization, error)
	ResolveMembership(context.Context, string, string) (Membership, error)
	ListMemberships(context.Context, string, string) ([]Membership, error)
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
