package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"plaiflow/api/internal/tenant"
)

func (s *Store) CreateOrganization(ctx context.Context, userID, organizationID, name string, now time.Time) (tenant.Organization, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return tenant.Organization{}, err
	}
	defer tx.Rollback(ctx)
	if err := setUserContext(ctx, tx, userID); err != nil {
		return tenant.Organization{}, err
	}
	var organization tenant.Organization
	err = tx.QueryRow(ctx, `SELECT id,name,role FROM create_user_organization($1,$2,$3,$4)`, userID, organizationID, name, now).Scan(
		&organization.ID, &organization.Name, &organization.Role)
	if err != nil {
		return tenant.Organization{}, err
	}
	var hasCategories bool
	if err := tx.QueryRow(ctx, `SELECT to_regclass('public.expense_categories') IS NOT NULL`).Scan(&hasCategories); err != nil {
		return tenant.Organization{}, err
	}
	if hasCategories {
		if _, err := tx.Exec(ctx, `SELECT set_config('app.organization_id',$1,true)`, organizationID); err != nil {
			return tenant.Organization{}, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO expense_categories(id,organization_id,name,created_at,updated_at)
			SELECT gen_random_uuid(),$1,seed.name,$2,$2 FROM (VALUES ('ค่าเดินทาง'),('ค่าอาหาร'),('วัสดุสำนักงาน')) AS seed(name)`,
			organizationID, now); err != nil {
			return tenant.Organization{}, err
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events (organization_id,actor_user_id,event_type,target_type,target_id,outcome,occurred_at)
        VALUES ($1,$2,'organization.create','organization',$4,'success',$3)`, organizationID, userID, now, organizationID); err != nil {
		return tenant.Organization{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return tenant.Organization{}, err
	}
	return organization, nil
}

func (s *Store) ListOrganizations(ctx context.Context, userID string) ([]tenant.Organization, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err := setUserContext(ctx, tx, userID); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT id,name,role FROM list_user_organizations($1)`, userID)
	if err != nil {
		return nil, err
	}
	organizations, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (tenant.Organization, error) {
		var organization tenant.Organization
		err := row.Scan(&organization.ID, &organization.Name, &organization.Role)
		return organization, err
	})
	if err != nil {
		return nil, err
	}
	return organizations, tx.Commit(ctx)
}

func (s *Store) ResolveMembership(ctx context.Context, userID, organizationID string) (tenant.Membership, error) {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return tenant.Membership{}, err
	}
	defer tx.Rollback(ctx)
	membership, err := membershipFor(ctx, tx, userID, organizationID)
	if err != nil {
		return tenant.Membership{}, err
	}
	return membership, tx.Commit(ctx)
}

func (s *Store) ListMemberships(ctx context.Context, userID, organizationID string) ([]tenant.Membership, error) {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err := membershipFor(ctx, tx, userID, organizationID); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT m.organization_id,m.user_id,coalesce(u.display_name,''),m.role,m.created_at
        FROM memberships m JOIN users u ON u.id=m.user_id
        WHERE m.organization_id=$1 ORDER BY m.role,m.created_at,m.user_id`, organizationID)
	if err != nil {
		return nil, err
	}
	memberships, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (tenant.Membership, error) {
		var membership tenant.Membership
		err := row.Scan(&membership.OrganizationID, &membership.UserID, &membership.DisplayName, &membership.Role, &membership.CreatedAt)
		return membership, err
	})
	if err != nil {
		return nil, err
	}
	return memberships, tx.Commit(ctx)
}

func (s *Store) CreateInvitation(ctx context.Context, invitation tenant.InviteCreate) error {
	tx, err := s.organizationTx(ctx, invitation.ActorUserID, invitation.OrganizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, invitation.ActorUserID, invitation.OrganizationID)
	if err != nil || !role.Allows(tenant.InviteMember) {
		return tenant.ErrForbidden
	}
	_, err = tx.Exec(ctx, `INSERT INTO invitations
        (id,organization_id,inviter_user_id,token_hash,created_at,expires_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		invitation.ID, invitation.OrganizationID, invitation.ActorUserID, invitation.TokenHash, invitation.Now, invitation.ExpiresAt)
	if err != nil {
		return err
	}
	if err := auditTenant(ctx, tx, invitation.OrganizationID, invitation.ActorUserID, "invitation.create", "invitation", invitation.ID, invitation.Now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ClaimInvitation(ctx context.Context, claim tenant.InviteClaim) (tenant.Invitation, error) {
	var invitation tenant.Invitation
	err := s.pool.QueryRow(ctx, `SELECT invitation_id,organization_id,organization_name,inviter_name,role,expires_at
        FROM claim_member_invitation($1,$2,$3,$4,$5,$6)`, claim.TokenHash, claim.HandoffHash, claim.HandoffID,
		claim.ReturnTo, claim.Now, claim.ExpiresAt).Scan(&invitation.ID, &invitation.OrganizationID, &invitation.OrganizationName,
		&invitation.InviterName, &invitation.Role, &invitation.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return tenant.Invitation{}, tenant.ErrInvalidInvite
	}
	return invitation, err
}

func (s *Store) AcceptInvitation(ctx context.Context, handoffHash []byte, userID string, now time.Time) (tenant.Organization, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return tenant.Organization{}, err
	}
	defer tx.Rollback(ctx)
	if err := setUserContext(ctx, tx, userID); err != nil {
		return tenant.Organization{}, err
	}
	var organization tenant.Organization
	err = tx.QueryRow(ctx, `SELECT id,name,role FROM accept_member_invitation($1,$2,$3)`, handoffHash, userID, now).Scan(
		&organization.ID, &organization.Name, &organization.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return tenant.Organization{}, tenant.ErrInvalidInvite
	}
	if err != nil {
		return tenant.Organization{}, err
	}
	return organization, tx.Commit(ctx)
}

func (s *Store) RevokeInvitation(ctx context.Context, organizationID, actorUserID, invitationID string, now time.Time) error {
	tx, err := s.organizationTx(ctx, actorUserID, organizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actorUserID, organizationID)
	if err != nil || !role.Allows(tenant.InviteMember) {
		return tenant.ErrForbidden
	}
	command, err := tx.Exec(ctx, `UPDATE invitations SET revoked_at=$4
        WHERE id=$1 AND organization_id=$2 AND revoked_at IS NULL AND accepted_at IS NULL AND expires_at>$4`,
		invitationID, organizationID, actorUserID, now)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return tenant.ErrNotFound
	}
	if err := auditTenant(ctx, tx, organizationID, actorUserID, "invitation.revoke", "invitation", invitationID, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ChangeRole(ctx context.Context, organizationID, actorUserID, targetUserID string, newRole tenant.Role, now time.Time) error {
	if newRole != tenant.Admin && newRole != tenant.Member {
		return tenant.ErrForbidden
	}
	tx, err := s.organizationTx(ctx, actorUserID, organizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actorUserID, organizationID)
	if err != nil || !role.Allows(tenant.ManageRoles) {
		return tenant.ErrForbidden
	}
	command, err := tx.Exec(ctx, `UPDATE memberships SET role=$4,updated_at=$5
        WHERE organization_id=$1 AND user_id=$2 AND user_id<>$3 AND role<>'Owner'`, organizationID, targetUserID, actorUserID, newRole, now)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return tenant.ErrNotFound
	}
	if err := auditTenant(ctx, tx, organizationID, actorUserID, "membership.role_change", "user", targetUserID, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) TransferOwnership(ctx context.Context, organizationID, actorUserID, targetUserID string, now time.Time) error {
	tx, err := s.organizationTx(ctx, actorUserID, organizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT transfer_organization_ownership($1,$2,$3,$4)`, organizationID, actorUserID, targetUserID, now); err != nil {
		return tenant.ErrConflict
	}
	if err := auditTenant(ctx, tx, organizationID, actorUserID, "organization.ownership_transfer", "user", targetUserID, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) RemoveMembership(ctx context.Context, organizationID, actorUserID, targetUserID string, now time.Time) error {
	tx, err := s.organizationTx(ctx, actorUserID, organizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	actorRole, err := currentRole(ctx, tx, actorUserID, organizationID)
	if err != nil || !actorRole.Allows(tenant.RemoveMember) {
		return tenant.ErrForbidden
	}
	var targetRole tenant.Role
	err = tx.QueryRow(ctx, `SELECT role FROM memberships WHERE organization_id=$1 AND user_id=$2 FOR UPDATE`, organizationID, targetUserID).Scan(&targetRole)
	if errors.Is(err, pgx.ErrNoRows) {
		return tenant.ErrNotFound
	}
	if err != nil {
		return err
	}
	if targetRole == tenant.Owner || (actorRole == tenant.Admin && targetRole != tenant.Member) {
		return tenant.ErrForbidden
	}
	if _, err := tx.Exec(ctx, `DELETE FROM memberships WHERE organization_id=$1 AND user_id=$2`, organizationID, targetUserID); err != nil {
		return err
	}
	if err := auditTenant(ctx, tx, organizationID, actorUserID, "membership.remove", "user", targetUserID, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) LeaveOrganization(ctx context.Context, organizationID, userID string, now time.Time) error {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, userID, organizationID)
	if err != nil {
		return err
	}
	if role == tenant.Owner {
		return tenant.ErrConflict
	}
	if _, err := tx.Exec(ctx, `DELETE FROM memberships WHERE organization_id=$1 AND user_id=$2`, organizationID, userID); err != nil {
		return err
	}
	if err := auditTenant(ctx, tx, organizationID, userID, "membership.leave", "user", userID, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) CreateLineLinkCode(ctx context.Context, code tenant.LineCodeCreate) error {
	tx, err := s.organizationTx(ctx, code.ActorUserID, code.OrganizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, code.ActorUserID, code.OrganizationID)
	if err != nil || !role.Allows(tenant.ManageGroup) {
		return tenant.ErrForbidden
	}
	var subject string
	err = tx.QueryRow(ctx, `SELECT subject FROM auth_identities
        WHERE user_id=$1 AND provider='line' AND disabled_at IS NULL`, code.ActorUserID).Scan(&subject)
	if errors.Is(err, pgx.ErrNoRows) {
		return tenant.ErrForbidden
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE line_link_codes SET revoked_at=$3
        WHERE organization_id=$1 AND initiating_user_id=$2 AND revoked_at IS NULL AND consumed_at IS NULL`,
		code.OrganizationID, code.ActorUserID, code.Now); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO line_link_codes
        (id,code_hash,organization_id,initiating_user_id,expected_line_subject,messaging_channel,created_at,expires_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, code.ID, code.CodeHash, code.OrganizationID, code.ActorUserID,
		subject, code.Channel, code.Now, code.ExpiresAt)
	if err != nil {
		return err
	}
	if err := auditTenant(ctx, tx, code.OrganizationID, code.ActorUserID, "line.group_code_create", "line_link_code", code.ID, code.Now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ListLineConnections(ctx context.Context, organizationID, userID string) ([]tenant.LineConnection, error) {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err := membershipFor(ctx, tx, userID, organizationID); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT id,group_id,status,connected_at,disconnected_at
        FROM line_group_connections WHERE organization_id=$1 ORDER BY connected_at DESC,id`, organizationID)
	if err != nil {
		return nil, err
	}
	connections, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (tenant.LineConnection, error) {
		var connection tenant.LineConnection
		err := row.Scan(&connection.ID, &connection.GroupID, &connection.Status, &connection.CreatedAt, &connection.EndedAt)
		return connection, err
	})
	if err != nil {
		return nil, err
	}
	return connections, tx.Commit(ctx)
}

func (s *Store) DisconnectLineConnection(ctx context.Context, organizationID, actorUserID, connectionID string, now time.Time) error {
	tx, err := s.organizationTx(ctx, actorUserID, organizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actorUserID, organizationID)
	if err != nil || !role.Allows(tenant.ManageGroup) {
		return tenant.ErrForbidden
	}
	command, err := tx.Exec(ctx, `UPDATE line_group_connections SET status='disconnected',disconnected_by_user_id=$3,disconnected_at=$4
        WHERE id=$1 AND organization_id=$2 AND status='connected'`, connectionID, organizationID, actorUserID, now)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return tenant.ErrNotFound
	}
	if err := auditTenant(ctx, tx, organizationID, actorUserID, "line.group_disconnect", "line_group_connection", connectionID, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) organizationTx(ctx context.Context, userID, organizationID string) (pgx.Tx, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `SELECT set_config('app.user_id',$1,true),set_config('app.organization_id',$2,true)`, userID, organizationID); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	return tx, nil
}

func setUserContext(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT set_config('app.user_id',$1,true)`, userID)
	return err
}

func membershipFor(ctx context.Context, tx pgx.Tx, userID, organizationID string) (tenant.Membership, error) {
	var membership tenant.Membership
	err := tx.QueryRow(ctx, `SELECT m.organization_id,o.name,m.user_id,m.role,m.created_at FROM memberships m
        JOIN organizations o ON o.id=m.organization_id WHERE m.user_id=$1 AND m.organization_id=$2`, userID, organizationID).Scan(
		&membership.OrganizationID, &membership.OrganizationName, &membership.UserID, &membership.Role, &membership.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return tenant.Membership{}, tenant.ErrNotFound
	}
	return membership, err
}

func currentRole(ctx context.Context, tx pgx.Tx, userID, organizationID string) (tenant.Role, error) {
	membership, err := membershipFor(ctx, tx, userID, organizationID)
	return membership.Role, err
}

func auditTenant(ctx context.Context, tx pgx.Tx, organizationID, actorUserID, eventType, targetType, targetID string, now time.Time) error {
	_, err := tx.Exec(ctx, `INSERT INTO audit_events
        (organization_id,actor_user_id,event_type,target_type,target_id,outcome,occurred_at)
        VALUES ($1,$2,$3,$4,$5,'success',$6)`, organizationID, actorUserID, eventType, targetType, targetID, now)
	return err
}
