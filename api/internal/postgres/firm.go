package postgres

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"plaiflow/api/internal/document"
	"plaiflow/api/internal/firm"
	"plaiflow/api/internal/tenant"
)

func (s *Store) ReadyFirm(ctx context.Context) error {
	var version int
	var dirty bool
	if err := s.pool.QueryRow(ctx, `SELECT version,dirty FROM schema_migrations LIMIT 1`).Scan(&version, &dirty); err != nil || dirty || version < 13 {
		return errors.New("firm schema is not ready")
	}
	return nil
}

func scanFirmGrant(row pgx.Row) (firm.Grant, error) {
	var g firm.Grant
	err := row.Scan(&g.ID, &g.FirmOrganizationID, &g.ClientOrganizationID, &g.PartnerName,
		&g.Status, &g.Scopes, &g.Revision, &g.RequestExpiresAt, &g.ActiveExpiresAt)
	return g, err
}

const firmGrantColumns = `g.id,g.firm_organization_id,g.client_organization_id,
	coalesce(firm_grant_partner_name(g.id),''),g.status,g.scopes,g.revision,g.request_expires_at,g.active_expires_at`

func (s *Store) RequestGrant(ctx context.Context, actor, firmID, clientID, id string, now time.Time) (firm.Grant, error) {
	if !documentUUID.MatchString(firmID) || !documentUUID.MatchString(clientID) || !documentUUID.MatchString(id) || firmID == clientID {
		return firm.Grant{}, tenant.ErrNotFound
	}
	tx, err := s.organizationTx(ctx, actor, firmID)
	if err != nil {
		return firm.Grant{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actor, firmID)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return firm.Grant{}, tenant.ErrForbidden
	}
	var key string
	if err := tx.QueryRow(ctx, `SELECT effective_organization_plan($1::uuid)`, firmID).Scan(&key); err != nil {
		return firm.Grant{}, err
	}
	if key != "AccountingFirm" {
		return firm.Grant{}, tenant.ErrForbidden
	}
	// Serialize capacity admission for a firm, including concurrent requests.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, firmID); err != nil {
		return firm.Grant{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE firm_access_grants SET status='Expired',ended_at=$2 WHERE firm_organization_id=$1
		AND ((status IN ('Requested','ClientApproved') AND request_expires_at<=$2)
		OR (status='Active' AND active_expires_at<=$2))`, firmID, now); err != nil {
		return firm.Grant{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM firm_access_grants WHERE firm_organization_id=$1
		AND status IN ('Requested','ClientApproved','Active')`, firmID).Scan(&count); err != nil {
		return firm.Grant{}, err
	}
	if count >= 20 {
		return firm.Grant{}, tenant.ErrConflict
	}
	// Foreign client ID is a locator only. No client data is read or exposed here.
	var g firm.Grant
	g, err = scanFirmGrant(tx.QueryRow(ctx, `INSERT INTO firm_access_grants
		(id,firm_organization_id,client_organization_id,status,scopes,requested_by,requested_at,request_expires_at)
		VALUES ($1,$2,$3,'Requested',ARRAY['read']::text[],$4,$5,$5+interval '7 days')
		RETURNING id,firm_organization_id,client_organization_id,''::text,status,scopes,revision,request_expires_at,active_expires_at`,
		id, firmID, clientID, actor, now))
	if err != nil {
		var databaseError *pgconn.PgError
		if errors.As(err, &databaseError) && (databaseError.Code == "23503" || databaseError.Code == "23505") {
			return firm.Grant{}, tenant.ErrConflict
		}
		return firm.Grant{}, err
	}
	if err := auditTenant(ctx, tx, firmID, actor, "firm.grant.request", "firm_grant", id, now); err != nil {
		return firm.Grant{}, err
	}
	return g, tx.Commit(ctx)
}

func (s *Store) ListGrants(ctx context.Context, actor, orgID string) ([]firm.Grant, error) {
	if !documentUUID.MatchString(orgID) {
		return nil, tenant.ErrNotFound
	}
	tx, err := s.organizationTx(ctx, actor, orgID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actor, orgID)
	if err != nil {
		return nil, tenant.ErrNotFound
	}
	rows, err := tx.Query(ctx, `SELECT `+firmGrantColumns+` FROM firm_access_grants g WHERE
		(g.firm_organization_id=$1 OR g.client_organization_id=$1)
		AND ($2::text IN ('Owner','Admin') OR
			(g.firm_organization_id=$1 AND g.status='Active' AND g.active_expires_at>now()
			AND EXISTS (SELECT 1 FROM firm_client_assignments a
				WHERE a.grant_id=g.id AND a.staff_user_id=$3 AND a.ended_at IS NULL)))
		ORDER BY g.requested_at DESC,g.id DESC LIMIT 100`, orgID, string(role), actor)
	if err != nil {
		return nil, err
	}
	grants, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (firm.Grant, error) { return scanFirmGrant(row) })
	if err != nil {
		return nil, err
	}
	return grants, tx.Commit(ctx)
}

func (s *Store) TransitionGrant(ctx context.Context, actor, orgID, grantID, action string, now time.Time) error {
	if !documentUUID.MatchString(orgID) || !documentUUID.MatchString(grantID) {
		return tenant.ErrNotFound
	}
	tx, err := s.organizationTx(ctx, actor, orgID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actor, orgID)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return tenant.ErrForbidden
	}
	var firmID, clientID, status string
	var requestExpiry time.Time
	var activeExpiry *time.Time
	err = tx.QueryRow(ctx, `SELECT firm_organization_id,client_organization_id,status,request_expires_at,active_expires_at
		FROM firm_access_grants WHERE id=$1 FOR UPDATE`, grantID).Scan(&firmID, &clientID, &status, &requestExpiry, &activeExpiry)
	if errors.Is(err, pgx.ErrNoRows) {
		return tenant.ErrNotFound
	}
	if err != nil {
		return err
	}
	firmSide, clientSide := orgID == firmID, orgID == clientID
	if !firmSide && !clientSide {
		return tenant.ErrNotFound
	}
	var next string
	switch action {
	case "approve":
		if !clientSide || status != "Requested" || !requestExpiry.After(now) {
			return tenant.ErrConflict
		}
		next = "ClientApproved"
	case "accept":
		if !firmSide || status != "ClientApproved" || !requestExpiry.After(now) {
			return tenant.ErrConflict
		}
		var key string
		if err := tx.QueryRow(ctx, `SELECT effective_organization_plan($1::uuid)`, firmID).Scan(&key); err != nil {
			return err
		}
		if key != "AccountingFirm" {
			return tenant.ErrForbidden
		}
		next = "Active"
	case "cancel":
		if status != "Requested" && status != "ClientApproved" {
			return tenant.ErrConflict
		}
		next = "Cancelled"
	case "revoke":
		if !clientSide || status != "Active" {
			return tenant.ErrConflict
		}
		next = "Revoked"
	case "end":
		if !firmSide || status != "Active" {
			return tenant.ErrConflict
		}
		next = "Cancelled"
	default:
		return tenant.ErrNotFound
	}
	_, err = tx.Exec(ctx, `UPDATE firm_access_grants SET status=$2,revision=revision+1,
		approved_by=CASE WHEN $3='approve' THEN $4::uuid ELSE approved_by END,
		approved_at=CASE WHEN $3='approve' THEN $5::timestamptz ELSE approved_at END,
		accepted_by=CASE WHEN $3='accept' THEN $4::uuid ELSE accepted_by END,
		accepted_at=CASE WHEN $3='accept' THEN $5::timestamptz ELSE accepted_at END,
		active_expires_at=CASE WHEN $3='accept' THEN $5::timestamptz+interval '12 months' ELSE active_expires_at END,
		ended_by=CASE WHEN $2 IN ('Cancelled','Revoked') THEN $4::uuid ELSE ended_by END,
		ended_at=CASE WHEN $2 IN ('Cancelled','Revoked') THEN $5::timestamptz ELSE ended_at END
		WHERE id=$1`, grantID, next, action, actor, now)
	if err != nil {
		return err
	}
	if err := auditTenant(ctx, tx, orgID, actor, "firm.grant."+action, "firm_grant", grantID, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) AssignStaff(ctx context.Context, actor, firmID, grantID, staffID string, now time.Time) error {
	if !documentUUID.MatchString(firmID) || !documentUUID.MatchString(grantID) || !documentUUID.MatchString(staffID) {
		return tenant.ErrNotFound
	}
	tx, err := s.organizationTx(ctx, actor, firmID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actor, firmID)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return tenant.ErrForbidden
	}
	if _, err := membershipFor(ctx, tx, staffID, firmID); err != nil {
		return tenant.ErrNotFound
	}
	var clientID string
	err = tx.QueryRow(ctx, `SELECT client_organization_id FROM firm_access_grants
		WHERE id=$1 AND firm_organization_id=$2 AND status='Active' AND active_expires_at>$3 FOR UPDATE`,
		grantID, firmID, now).Scan(&clientID)
	if errors.Is(err, pgx.ErrNoRows) {
		return tenant.ErrNotFound
	}
	if err != nil {
		return err
	}
	var key string
	if err := tx.QueryRow(ctx, `SELECT effective_organization_plan($1::uuid)`, firmID).Scan(&key); err != nil {
		return err
	}
	if key != "AccountingFirm" {
		return tenant.ErrForbidden
	}
	_, err = tx.Exec(ctx, `INSERT INTO firm_client_assignments
		(id,grant_id,firm_organization_id,client_organization_id,staff_user_id,assigned_by,assigned_at)
		VALUES (gen_random_uuid(),$1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`, grantID, firmID, clientID, staffID, actor, now)
	if err != nil {
		return err
	}
	if err := auditTenant(ctx, tx, firmID, actor, "firm.assignment.add", "firm_grant", grantID, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) RemoveStaff(ctx context.Context, actor, firmID, grantID, staffID string, now time.Time) error {
	if !documentUUID.MatchString(firmID) || !documentUUID.MatchString(grantID) || !documentUUID.MatchString(staffID) {
		return tenant.ErrNotFound
	}
	tx, err := s.organizationTx(ctx, actor, firmID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actor, firmID)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return tenant.ErrForbidden
	}
	command, err := tx.Exec(ctx, `UPDATE firm_client_assignments SET ended_at=$4 WHERE grant_id=$1
		AND firm_organization_id=$2 AND staff_user_id=$3 AND ended_at IS NULL`, grantID, firmID, staffID, now)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return tenant.ErrNotFound
	}
	if err := auditTenant(ctx, tx, firmID, actor, "firm.assignment.remove", "firm_grant", grantID, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ListStaff(ctx context.Context, actor, orgID, grantID string) ([]firm.Assignment, error) {
	if !documentUUID.MatchString(orgID) || !documentUUID.MatchString(grantID) {
		return nil, tenant.ErrNotFound
	}
	tx, err := s.organizationTx(ctx, actor, orgID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actor, orgID)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return nil, tenant.ErrForbidden
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM firm_access_grants WHERE id=$1
		AND (firm_organization_id=$2 OR client_organization_id=$2))`, grantID, orgID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, tenant.ErrNotFound
	}
	rows, err := tx.Query(ctx, `SELECT a.staff_user_id,coalesce(u.display_name,''),a.assigned_at
		FROM firm_client_assignments a JOIN users u ON u.id=a.staff_user_id WHERE a.grant_id=$1
		AND a.ended_at IS NULL AND is_organization_member(a.staff_user_id,a.firm_organization_id)
		ORDER BY a.assigned_at,a.staff_user_id`, grantID)
	if err != nil {
		return nil, err
	}
	staff, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (firm.Assignment, error) {
		var a firm.Assignment
		err := row.Scan(&a.UserID, &a.DisplayName, &a.AssignedAt)
		return a, err
	})
	if err != nil {
		return nil, err
	}
	return staff, tx.Commit(ctx)
}

func (s *Store) GetFirmDocument(ctx context.Context, actor, firmID, clientID, documentID, scope string) (document.Document, error) {
	if !documentUUID.MatchString(firmID) || !documentUUID.MatchString(clientID) || !documentUUID.MatchString(documentID) ||
		(scope != "read" && scope != "original.download") {
		return document.Document{}, tenant.ErrNotFound
	}
	tx, err := s.organizationTx(ctx, actor, firmID)
	if err != nil {
		return document.Document{}, err
	}
	defer tx.Rollback(ctx)
	// Both the explicit predicate and document RLS must approve this exact row.
	var doc document.Document
	err = tx.QueryRow(ctx, `SELECT id,organization_id,display_filename,detected_mime,byte_size,status,storage_key,accepted_at
		FROM documents WHERE organization_id=$1 AND id=$2 AND status IN ('Available','Archived')
		AND NOT group_restricted AND firm_grant_allows($3::uuid,$4::uuid,$1::uuid,$5)`,
		clientID, documentID, actor, firmID, scope).Scan(&doc.ID, &doc.OrganizationID, &doc.Filename, &doc.MIME,
		&doc.Size, &doc.Status, &doc.StorageKey, &doc.AcceptedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return document.Document{}, tenant.ErrNotFound
	}
	if err != nil {
		return document.Document{}, err
	}
	if err := auditTenant(ctx, tx, clientID, actor, "firm.document."+scope, "document", documentID, time.Now().UTC()); err != nil {
		return document.Document{}, err
	}
	return doc, tx.Commit(ctx)
}

func (s *Store) ListPortfolio(ctx context.Context, actor, firmID, cursor string) (firm.PortfolioPage, error) {
	var after struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	}
	if cursor != "" {
		data, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil || json.Unmarshal(data, &after) != nil || !documentUUID.MatchString(after.ID) || len(after.Name) > 640 {
			return firm.PortfolioPage{}, tenant.ErrNotFound
		}
	}
	if !documentUUID.MatchString(firmID) {
		return firm.PortfolioPage{}, tenant.ErrNotFound
	}
	tx, err := s.organizationTx(ctx, actor, firmID)
	if err != nil {
		return firm.PortfolioPage{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actor, firmID)
	if err != nil {
		return firm.PortfolioPage{}, tenant.ErrNotFound
	}
	var key string
	if err := tx.QueryRow(ctx, `SELECT effective_organization_plan($1::uuid)`, firmID).Scan(&key); err != nil {
		return firm.PortfolioPage{}, err
	}
	if key != "AccountingFirm" {
		return firm.PortfolioPage{}, tenant.ErrForbidden
	}
	rows, err := tx.Query(ctx, `WITH eligible AS (
		SELECT g.id,g.client_organization_id,firm_grant_partner_name(g.id) AS name,g.active_expires_at
		FROM firm_access_grants g WHERE g.firm_organization_id=$1 AND g.status='Active' AND g.active_expires_at>now()
	), roster AS (
		SELECT a.grant_id,count(*)::integer AS staff_count,
			bool_or(a.staff_user_id=$2::uuid) AS assigned
		FROM firm_client_assignments a WHERE a.firm_organization_id=$1 AND a.ended_at IS NULL
			AND is_organization_member(a.staff_user_id,$1::uuid)
		GROUP BY a.grant_id
	)
	SELECT e.id,e.client_organization_id,e.name,e.active_expires_at,
		coalesce(r.assigned,false),CASE WHEN $3::boolean THEN coalesce(r.staff_count,0) ELSE 0 END
	FROM eligible e LEFT JOIN roster r ON r.grant_id=e.id
	WHERE ($3::boolean OR coalesce(r.assigned,false))
		AND ($5::uuid IS NULL OR (e.name,e.client_organization_id) > ($4::text,$5::uuid))
	ORDER BY e.name,e.client_organization_id LIMIT 11`, firmID, actor, role == tenant.Owner || role == tenant.Admin, after.Name, nullableUUID(after.ID))
	if err != nil {
		return firm.PortfolioPage{}, err
	}
	page := firm.PortfolioPage{Clients: make([]firm.PortfolioClient, 0, 10)}
	for rows.Next() {
		var item firm.PortfolioClient
		if err := rows.Scan(&item.GrantID, &item.ClientOrganizationID, &item.ClientName, &item.ActiveExpiresAt, &item.Assigned, &item.StaffCount); err != nil {
			rows.Close()
			return firm.PortfolioPage{}, err
		}
		page.Clients = append(page.Clients, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return firm.PortfolioPage{}, err
	}
	if len(page.Clients) > 10 {
		page.Clients = page.Clients[:10]
		last := page.Clients[9]
		data, _ := json.Marshal(struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		}{last.ClientName, last.ClientOrganizationID})
		page.NextCursor = base64.RawURLEncoding.EncodeToString(data)
	}
	return page, tx.Commit(ctx)
}
