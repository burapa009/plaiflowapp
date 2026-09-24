package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"plaiflow/api/internal/extraction"
	"plaiflow/api/internal/tenant"
)

func (s *Store) ReadyReview(ctx context.Context) error {
	var ready bool
	if err := s.pool.QueryRow(ctx, `SELECT to_regclass('document_review_drafts') IS NOT NULL AND
		to_regclass('document_review_tasks') IS NOT NULL`).Scan(&ready); err != nil {
		return err
	}
	if !ready {
		return errors.New("review schema is not ready")
	}
	return nil
}

func (s *Store) CurrentDraft(ctx context.Context, user, org, doc string) (extraction.ReviewDraft, error) {
	if _, err := s.GetDocument(ctx, user, org, doc); err != nil {
		return extraction.ReviewDraft{}, err
	}
	tx, err := s.organizationTx(ctx, user, org)
	if err != nil {
		return extraction.ReviewDraft{}, err
	}
	defer tx.Rollback(ctx)
	var draft extraction.ReviewDraft
	var values, decisions []byte
	err = tx.QueryRow(ctx, `SELECT ocr_job_id,revision,values,decisions,updated_by,updated_at
		FROM document_review_drafts WHERE organization_id=$1 AND document_id=$2`, org, doc).Scan(
		&draft.OCRJobID, &draft.Revision, &values, &decisions, &draft.UpdatedBy, &draft.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return extraction.ReviewDraft{}, tx.Commit(ctx)
	}
	if err != nil {
		return extraction.ReviewDraft{}, err
	}
	if err = json.Unmarshal(values, &draft.Values); err != nil {
		return extraction.ReviewDraft{}, err
	}
	if err = json.Unmarshal(decisions, &draft.Decisions); err != nil {
		return extraction.ReviewDraft{}, err
	}
	draft.OrganizationID, draft.DocumentID = org, doc
	return draft, tx.Commit(ctx)
}

func (s *Store) ListDraftHistory(ctx context.Context, user, org, doc string, limit int) ([]extraction.ReviewDraft, error) {
	if limit < 1 || limit > 20 {
		return nil, extraction.ErrConflict
	}
	if _, err := s.GetDocument(ctx, user, org, doc); err != nil {
		return nil, err
	}
	tx, err := s.organizationTx(ctx, user, org)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, user, org)
	if err != nil {
		return nil, tenant.ErrNotFound
	}
	rows, err := tx.Query(ctx, `SELECT ocr_job_id,revision,values,decisions,updated_by,updated_at
		FROM document_review_draft_history WHERE organization_id=$1 AND document_id=$2
		AND ($3::boolean OR updated_by=$4) ORDER BY revision DESC LIMIT $5`, org, doc,
		role == tenant.Owner || role == tenant.Admin, user, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	history := make([]extraction.ReviewDraft, 0, limit)
	for rows.Next() {
		var draft extraction.ReviewDraft
		var values, decisions []byte
		if err = rows.Scan(&draft.OCRJobID, &draft.Revision, &values, &decisions, &draft.UpdatedBy, &draft.UpdatedAt); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(values, &draft.Values); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(decisions, &draft.Decisions); err != nil {
			return nil, err
		}
		draft.DocumentID = doc
		history = append(history, draft)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return history, tx.Commit(ctx)
}

func (s *Store) SaveDraft(ctx context.Context, draft extraction.ReviewDraft, expected int, requestID string) (extraction.ReviewDraft, error) {
	tx, err := s.organizationTx(ctx, draft.UpdatedBy, draft.OrganizationID)
	if err != nil {
		return extraction.ReviewDraft{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = currentRole(ctx, tx, draft.UpdatedBy, draft.OrganizationID); err != nil {
		return extraction.ReviewDraft{}, tenant.ErrForbidden
	}
	var visibleID string
	err = tx.QueryRow(ctx, `SELECT d.id FROM documents d WHERE d.organization_id=$1 AND d.id=$2
		AND d.status IN ('Available','Archived') AND (organization_role($3::uuid,$1::uuid) IN ('Owner','Admin')
		OR (d.submitted_by_user_id=$3::uuid AND NOT d.group_restricted) OR d.assignee_user_id=$3::uuid
		OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id
			AND ds.document_id=d.id AND ds.submitted_by_user_id=$3::uuid AND NOT ds.group_source))
		FOR UPDATE OF d`, draft.OrganizationID, draft.DocumentID, draft.UpdatedBy).Scan(&visibleID)
	if err != nil {
		return extraction.ReviewDraft{}, tenant.ErrNotFound
	}
	var currentOCR string
	err = tx.QueryRow(ctx, `SELECT job_id FROM document_ocr_runs WHERE organization_id=$1 AND document_id=$2
		AND published_at IS NOT NULL AND superseded_at IS NULL AND deleted_at IS NULL`, draft.OrganizationID, draft.DocumentID).Scan(&currentOCR)
	if err != nil || currentOCR != draft.OCRJobID {
		return extraction.ReviewDraft{}, extraction.ErrConflict
	}
	var revision int
	err = tx.QueryRow(ctx, `SELECT revision FROM document_review_drafts WHERE organization_id=$1 AND document_id=$2 FOR UPDATE`, draft.OrganizationID, draft.DocumentID).Scan(&revision)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return extraction.ReviewDraft{}, err
	}
	if revision != expected {
		return extraction.ReviewDraft{}, extraction.ErrConflict
	}
	values, err := json.Marshal(draft.Values)
	if err != nil {
		return extraction.ReviewDraft{}, err
	}
	decisions, err := json.Marshal(draft.Decisions)
	if err != nil {
		return extraction.ReviewDraft{}, err
	}
	draft.Revision = revision + 1
	_, err = tx.Exec(ctx, `INSERT INTO document_review_drafts
		(organization_id,document_id,ocr_job_id,revision,values,decisions,updated_by,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT(organization_id,document_id) DO UPDATE SET ocr_job_id=excluded.ocr_job_id,
		revision=excluded.revision,values=excluded.values,decisions=excluded.decisions,
		updated_by=excluded.updated_by,updated_at=excluded.updated_at`, draft.OrganizationID, draft.DocumentID,
		draft.OCRJobID, draft.Revision, values, decisions, draft.UpdatedBy, draft.UpdatedAt)
	if err != nil {
		return extraction.ReviewDraft{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO document_review_draft_history
		(organization_id,document_id,revision,ocr_job_id,values,decisions,updated_by,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, draft.OrganizationID, draft.DocumentID,
		draft.Revision, draft.OCRJobID, values, decisions, draft.UpdatedBy, draft.UpdatedAt)
	if err != nil {
		return extraction.ReviewDraft{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_events(organization_id,actor_user_id,event_type,target_type,target_id,
		request_id,outcome,occurred_at,metadata) VALUES($1,$2,'review.draft.save','document',$3,$4,'success',$5,
		jsonb_build_object('old_revision',$6,'new_revision',$7))`, draft.OrganizationID, draft.UpdatedBy,
		draft.DocumentID, requestID, draft.UpdatedAt, revision, draft.Revision)
	if err != nil {
		return extraction.ReviewDraft{}, err
	}
	return draft, tx.Commit(ctx)
}
