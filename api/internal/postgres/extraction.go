package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"plaiflow/api/internal/extraction"
	"plaiflow/api/internal/tenant"
)

func (s *Store) CurrentReview(ctx context.Context, user, org, doc string) (extraction.Review, error) {
	if _, err := s.GetDocument(ctx, user, org, doc); err != nil {
		return extraction.Review{}, err
	}
	tx, err := s.organizationTx(ctx, user, org)
	if err != nil {
		return extraction.Review{}, err
	}
	defer tx.Rollback(ctx)
	var review extraction.Review
	err = tx.QueryRow(ctx, `SELECT id,organization_id,document_id,ocr_job_id,revision,object_key,confirmed_by,confirmed_at
        FROM document_extraction_reviews WHERE organization_id=$1 AND document_id=$2 AND superseded_at IS NULL`, org, doc).Scan(
		&review.ID, &review.OrganizationID, &review.DocumentID, &review.OCRJobID, &review.Revision, &review.ObjectKey, &review.ConfirmedBy, &review.ConfirmedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return extraction.Review{}, tx.Commit(ctx)
	}
	if err != nil {
		return extraction.Review{}, err
	}
	return review, tx.Commit(ctx)
}

func (s *Store) SaveReview(ctx context.Context, review extraction.Review, expectedRevision int) (extraction.Review, error) {
	tx, err := s.organizationTx(ctx, review.ConfirmedBy, review.OrganizationID)
	if err != nil {
		return extraction.Review{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, review.ConfirmedBy, review.OrganizationID)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return extraction.Review{}, tenant.ErrForbidden
	}
	var status string
	if err = tx.QueryRow(ctx, `SELECT status FROM documents WHERE organization_id=$1 AND id=$2 FOR UPDATE`, review.OrganizationID, review.DocumentID).Scan(&status); err != nil || status != "Available" && status != "Archived" {
		return extraction.Review{}, tenant.ErrNotFound
	}
	var currentOCR string
	if err = tx.QueryRow(ctx, `SELECT job_id FROM document_ocr_runs WHERE organization_id=$1 AND document_id=$2
        AND published_at IS NOT NULL AND superseded_at IS NULL AND deleted_at IS NULL`, review.OrganizationID, review.DocumentID).Scan(&currentOCR); err != nil || currentOCR != review.OCRJobID {
		return extraction.Review{}, extraction.ErrConflict
	}
	var previousID string
	var revision int
	err = tx.QueryRow(ctx, `SELECT id,revision FROM document_extraction_reviews WHERE organization_id=$1 AND document_id=$2
        AND superseded_at IS NULL FOR UPDATE`, review.OrganizationID, review.DocumentID).Scan(&previousID, &revision)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return extraction.Review{}, err
	}
	if revision != expectedRevision {
		return extraction.Review{}, extraction.ErrConflict
	}
	if previousID != "" {
		if _, err = tx.Exec(ctx, `UPDATE document_extraction_reviews SET superseded_at=$2 WHERE id=$1`, previousID, review.ConfirmedAt); err != nil {
			return extraction.Review{}, err
		}
	}
	review.Revision = revision + 1
	_, err = tx.Exec(ctx, `INSERT INTO document_extraction_reviews
        (id,organization_id,document_id,ocr_job_id,revision,object_key,confirmed_by,confirmed_at)
        VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, review.ID, review.OrganizationID, review.DocumentID, review.OCRJobID,
		review.Revision, review.ObjectKey, review.ConfirmedBy, review.ConfirmedAt)
	if err != nil {
		return extraction.Review{}, err
	}
	if err = auditTenant(ctx, tx, review.OrganizationID, review.ConfirmedBy, "extraction.confirm", "document", review.DocumentID, review.ConfirmedAt); err != nil {
		return extraction.Review{}, err
	}
	return review, tx.Commit(ctx)
}

func (s *Store) ListCurrentReviews(ctx context.Context, user, org string, limit int) ([]extraction.Review, error) {
	if limit < 1 || limit > 5000 {
		return nil, errors.New("invalid extraction export limit")
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
	rows, err := tx.Query(ctx, `SELECT e.id,e.organization_id,e.document_id,e.ocr_job_id,e.revision,e.object_key,e.confirmed_by,e.confirmed_at
        FROM document_extraction_reviews e JOIN documents d ON d.organization_id=e.organization_id AND d.id=e.document_id
        WHERE e.organization_id=$1 AND e.superseded_at IS NULL AND d.status IN ('Available','Archived')
        AND EXISTS (SELECT 1 FROM document_ocr_runs o WHERE o.job_id=e.ocr_job_id AND o.organization_id=e.organization_id
            AND o.document_id=e.document_id AND o.published_at IS NOT NULL AND o.superseded_at IS NULL AND o.deleted_at IS NULL)
        ORDER BY e.confirmed_at DESC,e.id DESC LIMIT $2`, org, limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []extraction.Review
	for rows.Next() {
		var review extraction.Review
		if err := rows.Scan(&review.ID, &review.OrganizationID, &review.DocumentID, &review.OCRJobID, &review.Revision,
			&review.ObjectKey, &review.ConfirmedBy, &review.ConfirmedAt); err != nil {
			return nil, err
		}
		out = append(out, review)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) > limit {
		return nil, errors.New("extraction export requires asynchronous processing")
	}
	if err := auditTenant(ctx, tx, org, user, "extraction.export", "organization", org, time.Now().UTC()); err != nil {
		return nil, err
	}
	return out, tx.Commit(ctx)
}

func (s *Store) ListRecentReviews(ctx context.Context, user, org string, limit, offset int) ([]extraction.Review, error) {
	if limit < 1 || limit > 100 || offset < 0 || offset > 10000 {
		return nil, errors.New("invalid review queue limit")
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
	rows, err := tx.Query(ctx, `SELECT e.id,e.organization_id,e.document_id,e.ocr_job_id,e.revision,e.object_key,e.confirmed_by,e.confirmed_at
		FROM document_extraction_reviews e JOIN documents d ON d.organization_id=e.organization_id AND d.id=e.document_id
		WHERE e.organization_id=$1 AND e.superseded_at IS NULL AND d.status='Available'
		ORDER BY e.confirmed_at DESC,e.id DESC LIMIT $2 OFFSET $3`, org, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []extraction.Review
	for rows.Next() {
		var review extraction.Review
		if err := rows.Scan(&review.ID, &review.OrganizationID, &review.DocumentID, &review.OCRJobID,
			&review.Revision, &review.ObjectKey, &review.ConfirmedBy, &review.ConfirmedAt); err != nil {
			return nil, err
		}
		out = append(out, review)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, tx.Commit(ctx)
}
