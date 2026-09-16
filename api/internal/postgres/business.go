package postgres

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"plaiflow/api/internal/business"
	"plaiflow/api/internal/plan"
	"plaiflow/api/internal/tenant"
)

func (s *Store) EffectivePlan(ctx context.Context, organizationID string) (plan.Key, error) {
	var key plan.Key
	err := s.pool.QueryRow(ctx, `SELECT effective_organization_plan($1::uuid)`, organizationID).Scan(&key)
	return key, err
}

func (s *Store) CreateVendor(ctx context.Context, actorUserID, organizationID string, contact business.Contact, now time.Time) (business.Contact, error) {
	tx, err := s.organizationTx(ctx, actorUserID, organizationID)
	if err != nil {
		return business.Contact{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actorUserID, organizationID)
	if err != nil || (role != tenant.Owner && role != tenant.Admin) {
		return business.Contact{}, business.ErrForbidden
	}
	contact.OrganizationID = organizationID
	contact.CreatedAt = now
	err = tx.QueryRow(ctx, `INSERT INTO business_contacts
        (id,organization_id,display_name,normalized_name,contact_code,country,tax_id,branch_code,is_customer,is_vendor,created_at,updated_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,true,$10,$10)
        RETURNING id,organization_id,display_name,normalized_name,contact_code,country,tax_id,branch_code,is_customer,is_vendor,archived_at,created_at`,
		contact.ID, organizationID, contact.DisplayName, contact.NormalizedName, contact.ContactCode, contact.Country,
		contact.TaxID, contact.BranchCode, contact.Customer, now).Scan(&contact.ID, &contact.OrganizationID, &contact.DisplayName,
		&contact.NormalizedName, &contact.ContactCode, &contact.Country, &contact.TaxID, &contact.BranchCode,
		&contact.Customer, &contact.Vendor, &contact.ArchivedAt, &contact.CreatedAt)
	if duplicateDatabaseError(err) {
		return business.Contact{}, business.ErrDuplicate
	}
	if err != nil {
		return business.Contact{}, err
	}
	if err := auditTenant(ctx, tx, organizationID, actorUserID, "business_contact.create", "business_contact", contact.ID, now); err != nil {
		return business.Contact{}, err
	}
	return contact, tx.Commit(ctx)
}

func (s *Store) ListVendors(ctx context.Context, actorUserID, organizationID, search, cursor string, limit int) (business.Page, error) {
	tx, err := s.organizationTx(ctx, actorUserID, organizationID)
	if err != nil {
		return business.Page{}, business.ErrNotFound
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actorUserID, organizationID)
	if err != nil {
		return business.Page{}, business.ErrNotFound
	}
	cursorName, cursorID, ok := decodeBusinessCursor(cursor)
	if !ok {
		return business.Page{}, business.ErrInvalid
	}
	manager := role == tenant.Owner || role == tenant.Admin
	rows, err := tx.Query(ctx, `SELECT id,organization_id,display_name,normalized_name,contact_code,country,
        CASE WHEN $6 THEN tax_id WHEN tax_id='' THEN '' ELSE '*********'||right(tax_id,4) END,
        branch_code,is_customer,is_vendor,archived_at,created_at
        FROM business_contacts
        WHERE organization_id=$1 AND is_vendor AND archived_at IS NULL
          AND ($2='' OR normalized_name LIKE lower($2)||'%' OR lower(contact_code) LIKE lower($2)||'%' OR ($6 AND tax_id=$2))
          AND ($3='' OR (normalized_name,id) > ($3,$4::uuid))
        ORDER BY normalized_name,id LIMIT $5`, organizationID, search, cursorName, nullUUID(cursorID), limit+1, manager)
	if err != nil {
		return business.Page{}, err
	}
	defer rows.Close()
	contacts, err := pgx.CollectRows(rows, scanBusinessContact)
	if err != nil {
		return business.Page{}, err
	}
	page := business.Page{Vendors: contacts}
	if len(contacts) > limit {
		last := contacts[limit-1]
		page.Vendors = contacts[:limit]
		page.NextCursor = encodeBusinessCursor(last.NormalizedName, last.ID)
	}
	return page, tx.Commit(ctx)
}

func (s *Store) ExportVendors(ctx context.Context, actorUserID, organizationID string, limit int) ([]business.Contact, error) {
	tx, err := s.organizationTx(ctx, actorUserID, organizationID)
	if err != nil {
		return nil, business.ErrNotFound
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actorUserID, organizationID)
	if err != nil || (role != tenant.Owner && role != tenant.Admin) {
		return nil, business.ErrForbidden
	}
	rows, err := tx.Query(ctx, `SELECT id,organization_id,display_name,normalized_name,contact_code,country,tax_id,branch_code,
        is_customer,is_vendor,archived_at,created_at FROM business_contacts
        WHERE organization_id=$1 AND is_vendor ORDER BY normalized_name,id LIMIT $2`, organizationID, limit+1)
	if err != nil {
		return nil, err
	}
	contacts, err := pgx.CollectRows(rows, scanBusinessContact)
	if err != nil {
		return nil, err
	}
	if len(contacts) > limit {
		return nil, business.ErrInvalid
	}
	return contacts, tx.Commit(ctx)
}

func scanBusinessContact(row pgx.CollectableRow) (business.Contact, error) {
	var contact business.Contact
	err := row.Scan(&contact.ID, &contact.OrganizationID, &contact.DisplayName, &contact.NormalizedName, &contact.ContactCode,
		&contact.Country, &contact.TaxID, &contact.BranchCode, &contact.Customer, &contact.Vendor, &contact.ArchivedAt, &contact.CreatedAt)
	return contact, err
}

func (s *Store) CreateImportPreview(ctx context.Context, actorUserID, organizationID, previewID string, contacts []business.Contact, now time.Time) (business.Preview, error) {
	tx, err := s.organizationTx(ctx, actorUserID, organizationID)
	if err != nil {
		return business.Preview{}, business.ErrNotFound
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actorUserID, organizationID)
	if err != nil || (role != tenant.Owner && role != tenant.Admin) {
		return business.Preview{}, business.ErrForbidden
	}
	preview := business.Preview{ID: previewID, OrganizationID: organizationID, ExpiresAt: now.Add(24 * time.Hour)}
	seen := map[string]bool{}
	for index, contact := range contacts {
		row := business.PreviewRow{Row: index + 2, Status: "ready", Contact: contact}
		key := duplicateKey(contact)
		duplicate, err := contactExists(ctx, tx, organizationID, contact)
		if err != nil {
			return business.Preview{}, err
		}
		if duplicate || key != "" && seen[key] {
			row.Status, row.Reason = "duplicate", "strong_duplicate"
			preview.Duplicates++
		} else {
			preview.Ready++
			if key != "" {
				seen[key] = true
			}
		}
		preview.Rows = append(preview.Rows, row)
	}
	encoded, err := json.Marshal(preview.Rows)
	if err != nil {
		return business.Preview{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO business_import_previews
        (id,organization_id,requester_user_id,rows,ready_count,duplicate_count,warning_count,error_count,status,created_at,expires_at)
        VALUES ($1,$2,$3,$4,$5,$6,0,0,'Preview',$7,$8)`, preview.ID, organizationID, actorUserID, encoded, preview.Ready, preview.Duplicates, now, preview.ExpiresAt)
	if err != nil {
		return business.Preview{}, err
	}
	if err := auditTenant(ctx, tx, organizationID, actorUserID, "business_import.preview", "business_import", preview.ID, now); err != nil {
		return business.Preview{}, err
	}
	return preview, tx.Commit(ctx)
}

func (s *Store) GetImportPreview(ctx context.Context, actorUserID, organizationID, previewID string, now time.Time) (business.Preview, error) {
	tx, err := s.organizationTx(ctx, actorUserID, organizationID)
	if err != nil {
		return business.Preview{}, business.ErrNotFound
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actorUserID, organizationID)
	if err != nil || (role != tenant.Owner && role != tenant.Admin) {
		return business.Preview{}, business.ErrForbidden
	}
	var preview business.Preview
	var encoded []byte
	err = tx.QueryRow(ctx, `SELECT id,organization_id,rows,ready_count,duplicate_count,warning_count,error_count,expires_at
        FROM business_import_previews WHERE id=$1 AND organization_id=$2 AND status='Preview' AND expires_at>$3`,
		previewID, organizationID, now).Scan(&preview.ID, &preview.OrganizationID, &encoded, &preview.Ready,
		&preview.Duplicates, &preview.Warnings, &preview.Errors, &preview.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return business.Preview{}, business.ErrExpired
	}
	if err != nil {
		return business.Preview{}, err
	}
	if err := json.Unmarshal(encoded, &preview.Rows); err != nil {
		return business.Preview{}, err
	}
	return preview, tx.Commit(ctx)
}

func (s *Store) CommitImport(ctx context.Context, actorUserID, organizationID, previewID string, now time.Time) (business.ImportResult, error) {
	tx, err := s.organizationTx(ctx, actorUserID, organizationID)
	if err != nil {
		return business.ImportResult{}, business.ErrNotFound
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actorUserID, organizationID)
	if err != nil || (role != tenant.Owner && role != tenant.Admin) {
		return business.ImportResult{}, business.ErrForbidden
	}
	var encoded []byte
	err = tx.QueryRow(ctx, `SELECT rows FROM business_import_previews
        WHERE id=$1 AND organization_id=$2 AND requester_user_id=$3 AND status='Preview' AND expires_at>$4 FOR UPDATE`,
		previewID, organizationID, actorUserID, now).Scan(&encoded)
	if errors.Is(err, pgx.ErrNoRows) {
		return business.ImportResult{}, business.ErrExpired
	}
	if err != nil {
		return business.ImportResult{}, err
	}
	var rows []business.PreviewRow
	if err := json.Unmarshal(encoded, &rows); err != nil {
		return business.ImportResult{}, err
	}
	result := business.ImportResult{}
	for _, row := range rows {
		if row.Status != "ready" {
			result.Skipped++
			continue
		}
		contact := row.Contact
		contact.ID = postgresUUID()
		command, err := tx.Exec(ctx, `INSERT INTO business_contacts
            (id,organization_id,display_name,normalized_name,contact_code,country,tax_id,branch_code,is_customer,is_vendor,import_preview_id,created_at,updated_at)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,true,$10,$11,$11) ON CONFLICT DO NOTHING`,
			contact.ID, organizationID, contact.DisplayName, contact.NormalizedName, contact.ContactCode, contact.Country,
			contact.TaxID, contact.BranchCode, contact.Customer, previewID, now)
		if err != nil {
			return business.ImportResult{}, err
		}
		if command.RowsAffected() == 0 {
			result.Skipped++
		} else {
			result.Created++
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE business_import_previews SET status='Committed',committed_at=$3
        WHERE id=$1 AND organization_id=$2`, previewID, organizationID, now); err != nil {
		return business.ImportResult{}, err
	}
	if err := auditTenant(ctx, tx, organizationID, actorUserID, "business_import.commit", "business_import", previewID, now); err != nil {
		return business.ImportResult{}, err
	}
	return result, tx.Commit(ctx)
}

func contactExists(ctx context.Context, tx pgx.Tx, organizationID string, contact business.Contact) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM business_contacts WHERE organization_id=$1 AND
        (($2<>'' AND country=$3 AND tax_id=$2 AND branch_code=$4) OR ($5<>'' AND lower(contact_code)=lower($5))))`,
		organizationID, contact.TaxID, contact.Country, contact.BranchCode, contact.ContactCode).Scan(&exists)
	return exists, err
}

func duplicateKey(contact business.Contact) string {
	if contact.TaxID != "" {
		return "tax:" + contact.Country + ":" + contact.TaxID + ":" + contact.BranchCode
	}
	if contact.ContactCode != "" {
		return "code:" + strings.ToLower(contact.ContactCode)
	}
	return ""
}

func duplicateDatabaseError(err error) bool {
	var databaseError *pgconn.PgError
	return errors.As(err, &databaseError) && databaseError.Code == "23505"
}

func encodeBusinessCursor(name, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(name + "\x00" + id))
}

func decodeBusinessCursor(value string) (string, string, bool) {
	if value == "" {
		return "", "", true
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return "", "", false
	}
	parts := strings.SplitN(string(decoded), "\x00", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func nullUUID(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func postgresUUID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return ""
	}
	value[6] = value[6]&0x0f | 0x40
	value[8] = value[8]&0x3f | 0x80
	encoded := hex.EncodeToString(value)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32]
}
