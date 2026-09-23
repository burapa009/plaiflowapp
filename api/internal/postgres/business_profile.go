package postgres

import (
	"context"
	"time"

	"plaiflow/api/internal/tenant"
)

func (s *Store) GetBusinessProfile(ctx context.Context, userID, organizationID string) (tenant.BusinessProfile, error) {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return tenant.BusinessProfile{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = membershipFor(ctx, tx, userID, organizationID); err != nil {
		return tenant.BusinessProfile{}, err
	}
	var p tenant.BusinessProfile
	err = tx.QueryRow(ctx, `SELECT business_type,vat_status,branch_type,name_th,name_en,tax_id,address_1,address_2,district,province,postal_code,phone,logo,logo_type FROM organizations WHERE id=$1`, organizationID).Scan(
		&p.BusinessType, &p.VATStatus, &p.BranchType, &p.NameTH, &p.NameEN, &p.TaxID, &p.Address1, &p.Address2, &p.District, &p.Province, &p.PostalCode, &p.Phone, &p.Logo, &p.LogoType)
	if err != nil {
		return tenant.BusinessProfile{}, err
	}
	p.HasLogo = len(p.Logo) > 0
	return p, tx.Commit(ctx)
}

func (s *Store) UpdateBusinessProfile(ctx context.Context, userID, organizationID string, p tenant.BusinessProfile, now time.Time) error {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, userID, organizationID)
	if err != nil {
		return err
	}
	if !role.Allows(tenant.EditOrganization) {
		return tenant.ErrForbidden
	}
	_, err = tx.Exec(ctx, `UPDATE organizations SET name=$2,business_type=$3,vat_status=$4,branch_type=$5,name_th=$2,name_en=$6,tax_id=$7,address_1=$8,address_2=$9,district=$10,province=$11,postal_code=$12,phone=$13,logo=coalesce($14,logo),logo_type=CASE WHEN $14::bytea IS NULL THEN logo_type ELSE $15 END,updated_at=$16 WHERE id=$1`,
		organizationID, p.NameTH, p.BusinessType, p.VATStatus, p.BranchType, p.NameEN, p.TaxID, p.Address1, p.Address2, p.District, p.Province, p.PostalCode, p.Phone, p.Logo, p.LogoType, now)
	if err != nil {
		return err
	}
	if err = auditTenant(ctx, tx, organizationID, userID, "organization.business_update", "organization", organizationID, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
