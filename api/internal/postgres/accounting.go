package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"plaiflow/api/internal/accounting"
	"plaiflow/api/internal/extraction"
	"plaiflow/api/internal/tenant"
)

func (s *Store) ReadyAccounting(ctx context.Context) error {
	var version int
	var dirty bool
	if err := s.pool.QueryRow(ctx, `SELECT version,dirty FROM schema_migrations LIMIT 1`).Scan(&version, &dirty); err != nil || dirty || version < 10 {
		return errors.New("accounting migration is not ready")
	}
	return nil
}

func (s *Store) ListCategories(ctx context.Context, user, org string) ([]accounting.Category, error) {
	tx, err := s.organizationTx(ctx, user, org)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT id,name,archived_at FROM expense_categories WHERE organization_id=$1 ORDER BY archived_at NULLS FIRST,lower(name),id LIMIT 501`, org)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []accounting.Category{}
	for rows.Next() {
		var category accounting.Category
		if err := rows.Scan(&category.ID, &category.Name, &category.ArchivedAt); err != nil {
			return nil, err
		}
		out = append(out, category)
	}
	if err := rows.Err(); err != nil || len(out) > 500 {
		return nil, accounting.ErrInvalid
	}
	return out, tx.Commit(ctx)
}

func (s *Store) CreateCategory(ctx context.Context, user, org string, category accounting.Category, now time.Time) (accounting.Category, error) {
	category.Name = strings.TrimSpace(category.Name)
	if category.ID == "" || len(category.Name) == 0 || len([]rune(category.Name)) > 120 {
		return accounting.Category{}, accounting.ErrInvalid
	}
	tx, err := s.organizationTx(ctx, user, org)
	if err != nil {
		return accounting.Category{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, user, org)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return accounting.Category{}, tenant.ErrForbidden
	}
	err = tx.QueryRow(ctx, `INSERT INTO expense_categories(id,organization_id,name,created_at,updated_at) VALUES($1,$2,$3,$4,$4)
		RETURNING id,name,archived_at`, category.ID, org, category.Name, now).Scan(&category.ID, &category.Name, &category.ArchivedAt)
	if duplicateDatabaseError(err) {
		return accounting.Category{}, accounting.ErrConflict
	}
	if err != nil {
		return accounting.Category{}, err
	}
	if err := auditTenant(ctx, tx, org, user, "accounting.category.create", "expense_category", category.ID, now); err != nil {
		return accounting.Category{}, err
	}
	return category, tx.Commit(ctx)
}

func (s *Store) SetCategoryArchived(ctx context.Context, user, org, categoryID string, archived bool, now time.Time) error {
	tx, err := s.organizationTx(ctx, user, org)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, user, org)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return tenant.ErrForbidden
	}
	var archivedAt any
	if archived {
		archivedAt = now
	}
	var changedID string
	err = tx.QueryRow(ctx, `UPDATE expense_categories SET archived_at=$3,updated_at=$4 WHERE organization_id=$1 AND id=$2
		AND ((archived_at IS NULL AND $3::timestamptz IS NOT NULL) OR (archived_at IS NOT NULL AND $3::timestamptz IS NULL))
		RETURNING id`, org, categoryID, archivedAt, now).Scan(&changedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return accounting.ErrConflict
	}
	if duplicateDatabaseError(err) {
		return accounting.ErrConflict
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO accounting_rule_sets(organization_id,version) VALUES($1,1)
		ON CONFLICT(organization_id) DO UPDATE SET version=accounting_rule_sets.version+1`, org)
	if err != nil {
		return err
	}
	action := "accounting.category.restore"
	if archived {
		action = "accounting.category.archive"
	}
	if err := auditTenant(ctx, tx, org, user, action, "expense_category", categoryID, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) RenameCategory(ctx context.Context, user, org, categoryID, name string, now time.Time) error {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 120 {
		return accounting.ErrInvalid
	}
	tx, err := s.organizationTx(ctx, user, org)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, user, org)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return tenant.ErrForbidden
	}
	var id string
	err = tx.QueryRow(ctx, `UPDATE expense_categories SET name=$3,updated_at=$4 WHERE organization_id=$1 AND id=$2
		AND archived_at IS NULL AND name<>$3 RETURNING id`, org, categoryID, name, now).Scan(&id)
	if duplicateDatabaseError(err) || errors.Is(err, pgx.ErrNoRows) {
		return accounting.ErrConflict
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO accounting_rule_sets(organization_id,version) VALUES($1,1)
		ON CONFLICT(organization_id) DO UPDATE SET version=accounting_rule_sets.version+1`, org)
	if err != nil {
		return err
	}
	if err := auditTenant(ctx, tx, org, user, "accounting.category.rename", "expense_category", categoryID, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) Evaluate(ctx context.Context, user, org string, review extraction.Review) (accounting.Evaluation, error) {
	if review.OrganizationID != org || review.ID == "" || review.Values == nil {
		return accounting.Evaluation{}, accounting.ErrInvalid
	}
	tx, err := s.organizationTx(ctx, user, org)
	if err != nil {
		return accounting.Evaluation{}, err
	}
	defer tx.Rollback(ctx)
	var currentID string
	err = tx.QueryRow(ctx, `SELECT e.id FROM document_extraction_reviews e JOIN documents d ON d.organization_id=e.organization_id AND d.id=e.document_id
		JOIN document_ocr_runs o ON o.job_id=e.ocr_job_id AND o.organization_id=e.organization_id AND o.document_id=e.document_id
		WHERE e.organization_id=$1 AND e.document_id=$2 AND e.superseded_at IS NULL AND d.status IN ('Available','Archived')
		AND o.published_at IS NOT NULL AND o.superseded_at IS NULL AND o.deleted_at IS NULL`, org, review.DocumentID).Scan(&currentID)
	if errors.Is(err, pgx.ErrNoRows) || currentID != review.ID {
		return accounting.Evaluation{}, accounting.ErrConflict
	}
	if err != nil {
		return accounting.Evaluation{}, err
	}
	result := accounting.Evaluation{}
	err = tx.QueryRow(ctx, `SELECT version FROM accounting_rule_sets WHERE organization_id=$1`, org).Scan(&result.RuleSetVersion)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return accounting.Evaluation{}, err
	}
	branch := review.Values["seller_branch"]
	taxID := review.Values["seller_tax_id"]
	if branch != "" && taxID != "" {
		var vendorID string
		err = tx.QueryRow(ctx, `SELECT id FROM business_contacts WHERE organization_id=$1 AND country='TH' AND tax_id=$2
			AND branch_code=$3 AND is_vendor AND archived_at IS NULL`, org, taxID, branch).Scan(&vendorID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return accounting.Evaluation{}, err
		}
		result.VendorID = vendorID
	}
	var rules []accounting.Rule
	if result.VendorID != "" {
		rows, err := tx.Query(ctx, `SELECT r.id,r.vendor_id,r.document_type,r.category_id,r.version FROM accounting_mapping_rules r
			JOIN expense_categories c ON c.organization_id=r.organization_id AND c.id=r.category_id AND c.archived_at IS NULL
			WHERE r.organization_id=$1 AND r.vendor_id=$2 AND r.retired_at IS NULL AND r.document_type IN ('',$3)
			ORDER BY r.id LIMIT 101`, org, result.VendorID, review.DocumentType)
		if err != nil {
			return accounting.Evaluation{}, err
		}
		for rows.Next() {
			var rule accounting.Rule
			if err := rows.Scan(&rule.ID, &rule.VendorID, &rule.DocumentType, &rule.CategoryID, &rule.Version); err != nil {
				rows.Close()
				return accounting.Evaluation{}, err
			}
			rules = append(rules, rule)
		}
		err = rows.Err()
		rows.Close()
		if err != nil || len(rules) > 100 {
			return accounting.Evaluation{}, accounting.ErrInvalid
		}
	}
	result.Candidate = accounting.Suggest(result.VendorID, review.DocumentType, rules)
	result.Status = "Draft"
	if len(result.Candidate.Warnings) > 0 {
		result.Status = "NeedsReview"
	}
	if branch == "" && taxID != "" {
		result.Candidate.Warnings = append(result.Candidate.Warnings, "branch_unknown")
		result.Status = "NeedsReview"
	}
	var approved accounting.Approval
	err = tx.QueryRow(ctx, `SELECT a.id,a.revision,a.document_id,a.review_id,a.review_revision,a.category_id,a.category_name,
		coalesce(a.vendor_id::text,''),a.vendor_contact_code,a.suggestion_basis,coalesce(a.rule_version,0),a.approved_at
		FROM accounting_suggestions a
		WHERE a.organization_id=$1 AND a.document_id=$2 AND a.superseded_at IS NULL`, org, review.DocumentID).Scan(
		&approved.ID, &approved.Revision, &approved.DocumentID, &approved.ReviewID, &approved.ReviewRevision, &approved.CategoryID, &approved.CategoryName,
		&approved.VendorID, &approved.ContactCode, &approved.Basis, &approved.RuleVersion, &approved.ApprovedAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return accounting.Evaluation{}, err
	}
	if err == nil && approved.ReviewID == review.ID {
		approved.Warnings = []string{}
		result.Approved = &approved
		result.VendorID = approved.VendorID
		result.Candidate = accounting.Candidate{CategoryID: approved.CategoryID, Basis: "human_reviewed", Confidence: "unrated", Warnings: []string{}}
		result.Status = "Approved"
	} else if err == nil {
		result.Candidate.Warnings = append(result.Candidate.Warnings, "source_changed")
		result.Status = "NeedsReview"
	}
	return result, tx.Commit(ctx)
}

func (s *Store) Approve(ctx context.Context, input accounting.ApproveInput) (accounting.Approval, error) {
	if input.ID == "" || input.ReviewID == "" || input.ReviewRevision < 1 || input.CategoryID == "" || input.RuleSetVersion < 0 ||
		input.ExpectedRevision < 0 || len(input.UnmatchedReason) > 240 || input.VendorID == "" && strings.TrimSpace(input.UnmatchedReason) == "" {
		return accounting.Approval{}, accounting.ErrInvalid
	}
	tx, err := s.organizationTx(ctx, input.ActorID, input.OrganizationID)
	if err != nil {
		return accounting.Approval{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, input.ActorID, input.OrganizationID)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return accounting.Approval{}, tenant.ErrForbidden
	}
	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM documents WHERE organization_id=$1 AND id=$2 FOR UPDATE`, input.OrganizationID, input.DocumentID).Scan(&status)
	if err != nil || status != "Available" && status != "Archived" {
		return accounting.Approval{}, tenant.ErrNotFound
	}
	var currentReviewID string
	var currentRevision int
	err = tx.QueryRow(ctx, `SELECT e.id,e.revision FROM document_extraction_reviews e JOIN document_ocr_runs o
		ON o.job_id=e.ocr_job_id AND o.organization_id=e.organization_id AND o.document_id=e.document_id
		WHERE e.organization_id=$1 AND e.document_id=$2 AND e.superseded_at IS NULL
		AND o.published_at IS NOT NULL AND o.superseded_at IS NULL AND o.deleted_at IS NULL FOR UPDATE OF e`, input.OrganizationID, input.DocumentID).Scan(&currentReviewID, &currentRevision)
	if err != nil || currentReviewID != input.ReviewID || currentRevision != input.ReviewRevision {
		return accounting.Approval{}, accounting.ErrConflict
	}
	var ruleSetVersion int
	err = tx.QueryRow(ctx, `SELECT version FROM accounting_rule_sets WHERE organization_id=$1 FOR UPDATE`, input.OrganizationID).Scan(&ruleSetVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		ruleSetVersion = 0
	} else if err != nil {
		return accounting.Approval{}, err
	}
	if ruleSetVersion != input.RuleSetVersion {
		return accounting.Approval{}, accounting.ErrConflict
	}
	var categoryName string
	if err := tx.QueryRow(ctx, `SELECT name FROM expense_categories WHERE organization_id=$1 AND id=$2 AND archived_at IS NULL`, input.OrganizationID, input.CategoryID).Scan(&categoryName); err != nil {
		return accounting.Approval{}, accounting.ErrInvalid
	}
	contactCode := ""
	if input.VendorID != "" {
		if err := tx.QueryRow(ctx, `SELECT contact_code FROM business_contacts WHERE organization_id=$1 AND id=$2 AND is_vendor AND archived_at IS NULL`, input.OrganizationID, input.VendorID).Scan(&contactCode); err != nil {
			return accounting.Approval{}, accounting.ErrInvalid
		}
	}
	var priorID string
	var priorRevision int
	err = tx.QueryRow(ctx, `SELECT id,revision FROM accounting_suggestions WHERE organization_id=$1 AND document_id=$2 AND superseded_at IS NULL FOR UPDATE`, input.OrganizationID, input.DocumentID).Scan(&priorID, &priorRevision)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return accounting.Approval{}, err
	}
	if priorRevision != input.ExpectedRevision {
		return accounting.Approval{}, accounting.ErrConflict
	}
	if priorID != "" {
		if _, err := tx.Exec(ctx, `UPDATE accounting_suggestions SET superseded_at=$2 WHERE id=$1`, priorID, input.ApprovedAt); err != nil {
			return accounting.Approval{}, err
		}
	}
	approved := accounting.Approval{ID: input.ID, Revision: priorRevision + 1, DocumentID: input.DocumentID, ReviewID: input.ReviewID,
		ReviewRevision: input.ReviewRevision, CategoryID: input.CategoryID, CategoryName: categoryName, VendorID: input.VendorID,
		ContactCode: contactCode, Basis: "human_reviewed", Warnings: []string{}, ApprovedAt: input.ApprovedAt}
	_, err = tx.Exec(ctx, `INSERT INTO accounting_suggestions(id,organization_id,document_id,review_id,review_revision,revision,rule_set_version,
		category_id,category_name,vendor_id,vendor_contact_code,unmatched_vendor_reason,suggestion_basis,approved_by,approved_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'human_reviewed',$13,$14)`, input.ID, input.OrganizationID, input.DocumentID,
		input.ReviewID, input.ReviewRevision, priorRevision+1, ruleSetVersion, input.CategoryID, categoryName, nullUUID(input.VendorID),
		contactCode, strings.TrimSpace(input.UnmatchedReason), input.ActorID, input.ApprovedAt)
	if duplicateDatabaseError(err) {
		return accounting.Approval{}, accounting.ErrConflict
	}
	if err != nil {
		return accounting.Approval{}, err
	}
	if err := auditTenant(ctx, tx, input.OrganizationID, input.ActorID, "accounting.suggestion.approve", "accounting_suggestion", input.ID, input.ApprovedAt); err != nil {
		return accounting.Approval{}, err
	}
	return approved, tx.Commit(ctx)
}

func (s *Store) ListApproved(ctx context.Context, user, org string, limit int, documentID string) ([]accounting.Approval, error) {
	if limit < 1 || limit > 100 || len(documentID) > 36 {
		return nil, accounting.ErrInvalid
	}
	tx, err := s.organizationTx(ctx, user, org)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, user, org)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return nil, tenant.ErrForbidden
	}
	rows, err := tx.Query(ctx, `SELECT a.id,a.revision,a.document_id,a.review_id,a.review_revision,a.category_id,a.category_name,
		coalesce(a.vendor_id::text,''),a.vendor_contact_code,a.suggestion_basis,coalesce(a.rule_version,0),a.approved_at
		FROM accounting_suggestions a JOIN document_extraction_reviews e ON e.organization_id=a.organization_id AND e.id=a.review_id
		JOIN documents d ON d.organization_id=a.organization_id AND d.id=a.document_id
		JOIN document_ocr_runs o ON o.job_id=e.ocr_job_id AND o.organization_id=e.organization_id AND o.document_id=e.document_id
		WHERE a.organization_id=$1 AND ($3='' OR a.document_id::text=$3) AND a.superseded_at IS NULL AND e.superseded_at IS NULL AND d.status='Available'
		AND o.published_at IS NOT NULL AND o.superseded_at IS NULL AND o.deleted_at IS NULL
		ORDER BY a.approved_at DESC,a.id DESC LIMIT $2`, org, limit+1, documentID)
	if err != nil {
		return nil, err
	}
	var out []accounting.Approval
	for rows.Next() {
		var approved accounting.Approval
		if err := rows.Scan(&approved.ID, &approved.Revision, &approved.DocumentID, &approved.ReviewID, &approved.ReviewRevision, &approved.CategoryID,
			&approved.CategoryName, &approved.VendorID, &approved.ContactCode, &approved.Basis, &approved.RuleVersion, &approved.ApprovedAt); err != nil {
			rows.Close()
			return nil, err
		}
		approved.Warnings = []string{}
		out = append(out, approved)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if len(out) > limit {
		return nil, accounting.ErrTooMany
	}
	if err := auditTenant(ctx, tx, org, user, "accounting.export", "organization", org, time.Now().UTC()); err != nil {
		return nil, err
	}
	return out, tx.Commit(ctx)
}

func (s *Store) CreateRule(ctx context.Context, input accounting.RuleInput) (accounting.Rule, error) {
	if input.ID == "" || input.VendorID == "" || input.CategoryID == "" || input.DocumentType != "" &&
		input.DocumentType != "receipt" && input.DocumentType != "invoice" && input.DocumentType != "tax_invoice" {
		return accounting.Rule{}, accounting.ErrInvalid
	}
	tx, err := s.organizationTx(ctx, input.ActorID, input.OrganizationID)
	if err != nil {
		return accounting.Rule{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, input.ActorID, input.OrganizationID)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return accounting.Rule{}, tenant.ErrForbidden
	}
	var valid bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM business_contacts v JOIN expense_categories c ON c.organization_id=v.organization_id
		WHERE v.organization_id=$1 AND v.id=$2 AND c.id=$3 AND v.is_vendor AND v.archived_at IS NULL AND c.archived_at IS NULL)`,
		input.OrganizationID, input.VendorID, input.CategoryID).Scan(&valid)
	if err != nil || !valid {
		return accounting.Rule{}, accounting.ErrInvalid
	}
	_, err = tx.Exec(ctx, `INSERT INTO accounting_rule_sets(organization_id,version) VALUES($1,0) ON CONFLICT DO NOTHING`, input.OrganizationID)
	if err != nil {
		return accounting.Rule{}, err
	}
	var version int
	if err := tx.QueryRow(ctx, `UPDATE accounting_rule_sets SET version=version+1 WHERE organization_id=$1 RETURNING version`, input.OrganizationID).Scan(&version); err != nil {
		return accounting.Rule{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO accounting_mapping_rules(id,organization_id,vendor_id,document_type,category_id,version,approved_by,approved_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, input.ID, input.OrganizationID, input.VendorID, input.DocumentType,
		input.CategoryID, version, input.ActorID, input.ApprovedAt)
	if err != nil {
		return accounting.Rule{}, err
	}
	if err := auditTenant(ctx, tx, input.OrganizationID, input.ActorID, "accounting.rule.approve", "accounting_rule", input.ID, input.ApprovedAt); err != nil {
		return accounting.Rule{}, err
	}
	rule := accounting.Rule{ID: input.ID, VendorID: input.VendorID, DocumentType: input.DocumentType, CategoryID: input.CategoryID, Version: version}
	return rule, tx.Commit(ctx)
}

func (s *Store) RetireRule(ctx context.Context, user, org, ruleID string, now time.Time) error {
	tx, err := s.organizationTx(ctx, user, org)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, user, org)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return tenant.ErrForbidden
	}
	var id string
	err = tx.QueryRow(ctx, `UPDATE accounting_mapping_rules SET retired_at=$3 WHERE organization_id=$1 AND id=$2 AND retired_at IS NULL
		RETURNING id`, org, ruleID, now).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return accounting.ErrConflict
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE accounting_rule_sets SET version=version+1 WHERE organization_id=$1`, org)
	if err != nil {
		return err
	}
	if err := auditTenant(ctx, tx, org, user, "accounting.rule.retire", "accounting_rule", id, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ListRules(ctx context.Context, user, org, vendorID string) ([]accounting.Rule, error) {
	if vendorID == "" {
		return nil, accounting.ErrInvalid
	}
	tx, err := s.organizationTx(ctx, user, org)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT id,vendor_id,document_type,category_id,version FROM accounting_mapping_rules
		WHERE organization_id=$1 AND vendor_id=$2 AND retired_at IS NULL ORDER BY document_type,version DESC,id LIMIT 101`, org, vendorID)
	if err != nil {
		return nil, err
	}
	var rules []accounting.Rule
	for rows.Next() {
		var rule accounting.Rule
		if err := rows.Scan(&rule.ID, &rule.VendorID, &rule.DocumentType, &rule.CategoryID, &rule.Version); err != nil {
			rows.Close()
			return nil, err
		}
		rules = append(rules, rule)
	}
	err = rows.Err()
	rows.Close()
	if err != nil || len(rules) > 100 {
		return nil, accounting.ErrInvalid
	}
	return rules, tx.Commit(ctx)
}
