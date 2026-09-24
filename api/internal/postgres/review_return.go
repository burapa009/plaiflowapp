package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"plaiflow/api/internal/extraction"
	"plaiflow/api/internal/tenant"
)

func (s *Store) CurrentReturn(ctx context.Context, user, org, doc string) (extraction.ReturnNotice, error) {
	if _, err := s.GetDocument(ctx, user, org, doc); err != nil {
		return extraction.ReturnNotice{}, err
	}
	tx, err := s.organizationTx(ctx, user, org)
	if err != nil {
		return extraction.ReturnNotice{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, user, org)
	if err != nil {
		return extraction.ReturnNotice{}, tenant.ErrNotFound
	}
	var notice extraction.ReturnNotice
	err = tx.QueryRow(ctx, `SELECT r.reason_code,r.private_note,r.returned_at FROM document_review_returns r
		WHERE r.organization_id=$1 AND r.document_id=$2 AND r.action='Return'
		AND EXISTS (SELECT 1 FROM document_ocr_runs o WHERE o.job_id=r.ocr_job_id AND o.organization_id=r.organization_id
			AND o.document_id=r.document_id AND o.published_at IS NOT NULL AND o.superseded_at IS NULL AND o.deleted_at IS NULL)
		AND ($3::boolean OR EXISTS (SELECT 1 FROM document_review_tasks rt JOIN tasks t
			ON t.organization_id=rt.organization_id AND t.id=rt.task_id
			WHERE rt.organization_id=r.organization_id AND rt.document_id=r.document_id
			AND t.assignee_user_id=$4 AND t.status IN ('Open','InProgress')))
		ORDER BY r.returned_at DESC,r.id DESC LIMIT 1`, org, doc,
		role == tenant.Owner || role == tenant.Admin, user).Scan(&notice.ReasonCode, &notice.PrivateNote, &notice.ReturnedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return notice, tx.Commit(ctx)
	}
	if err != nil {
		return extraction.ReturnNotice{}, err
	}
	return notice, tx.Commit(ctx)
}

func (s *Store) ReturnReview(ctx context.Context, input extraction.ReturnInput) (int, error) {
	switch input.ReasonCode {
	case "missing_value", "incorrect_value", "unreadable_original", "other":
	default:
		return 0, extraction.ErrConflict
	}
	input.PrivateNote = strings.TrimSpace(input.PrivateNote)
	if len([]rune(input.PrivateNote)) > 1000 || input.ExpectedReviewRevision < 0 || input.ExpectedDraftRevision < 0 {
		return 0, extraction.ErrConflict
	}
	tx, err := s.organizationTx(ctx, input.ActorID, input.OrganizationID)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, input.ActorID, input.OrganizationID)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return 0, tenant.ErrForbidden
	}
	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM documents WHERE organization_id=$1 AND id=$2 FOR UPDATE`, input.OrganizationID, input.DocumentID).Scan(&status)
	if err != nil || status != "Available" && status != "Archived" {
		return 0, extraction.ErrConflict
	}
	var reviewRevision int
	var reviewOCR string
	err = tx.QueryRow(ctx, `SELECT revision,ocr_job_id FROM document_extraction_reviews WHERE organization_id=$1
		AND document_id=$2 AND superseded_at IS NULL FOR UPDATE`, input.OrganizationID, input.DocumentID).Scan(&reviewRevision, &reviewOCR)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) || reviewRevision != input.ExpectedReviewRevision ||
		reviewOCR != "" && reviewOCR != input.OCRJobID {
		return 0, extraction.ErrConflict
	}
	var currentOCR string
	err = tx.QueryRow(ctx, `SELECT job_id FROM document_ocr_runs WHERE organization_id=$1 AND document_id=$2
		AND published_at IS NOT NULL AND superseded_at IS NULL AND deleted_at IS NULL`, input.OrganizationID, input.DocumentID).Scan(&currentOCR)
	if err != nil || currentOCR != input.OCRJobID {
		return 0, extraction.ErrConflict
	}
	var draftRevision int
	err = tx.QueryRow(ctx, `SELECT revision FROM document_review_drafts WHERE organization_id=$1 AND document_id=$2 FOR UPDATE`, input.OrganizationID, input.DocumentID).Scan(&draftRevision)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	if draftRevision != input.ExpectedDraftRevision {
		return 0, extraction.ErrConflict
	}
	next := draftRevision + 1
	_, err = tx.Exec(ctx, `INSERT INTO document_review_returns(id,organization_id,document_id,ocr_job_id,reason_code,
		private_note,returned_by,returned_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, input.ID, input.OrganizationID,
		input.DocumentID, input.OCRJobID, input.ReasonCode, input.PrivateNote, input.ActorID, input.ReturnedAt)
	if err != nil {
		return 0, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO document_review_drafts(organization_id,document_id,ocr_job_id,revision,
		values,decisions,updated_by,updated_at) VALUES($1,$2,$3,$4,'{}'::jsonb,'{}'::jsonb,$5,$6)
		ON CONFLICT(organization_id,document_id) DO UPDATE SET revision=excluded.revision,
		ocr_job_id=excluded.ocr_job_id,values=CASE WHEN document_review_drafts.ocr_job_id=excluded.ocr_job_id
			THEN document_review_drafts.values ELSE '{}'::jsonb END,
		decisions=CASE WHEN document_review_drafts.ocr_job_id=excluded.ocr_job_id
			THEN document_review_drafts.decisions ELSE '{}'::jsonb END,
		updated_by=excluded.updated_by,updated_at=excluded.updated_at`, input.OrganizationID, input.DocumentID, input.OCRJobID,
		next, input.ActorID, input.ReturnedAt)
	if err != nil {
		return 0, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO document_review_draft_history(organization_id,document_id,revision,ocr_job_id,
		values,decisions,updated_by,updated_at)
		SELECT organization_id,document_id,revision,ocr_job_id,values,decisions,updated_by,updated_at
		FROM document_review_drafts WHERE organization_id=$1 AND document_id=$2`, input.OrganizationID, input.DocumentID)
	if err != nil {
		return 0, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_events(organization_id,actor_user_id,event_type,target_type,target_id,
		request_id,outcome,reason_code,occurred_at,metadata) VALUES($1,$2,'review.return','document',$3,$4,'success',$5,$6,
		jsonb_build_object('old_draft_revision',$7,'new_draft_revision',$8,'review_revision',$9))`, input.OrganizationID,
		input.ActorID, input.DocumentID, input.RequestID, input.ReasonCode, input.ReturnedAt, draftRevision, next, reviewRevision)
	if err != nil {
		return 0, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO notifications(id,organization_id,recipient_user_id,category,logical_key,title,
		deep_link,delivery_mode,created_at)
		SELECT gen_random_uuid(),$1,m.user_id,'TaskChange',$3,'Review needs correction',$4,'Immediate',$5
		FROM document_review_tasks rt JOIN tasks t ON t.organization_id=rt.organization_id AND t.id=rt.task_id
		JOIN memberships m ON m.organization_id=t.organization_id AND m.user_id=t.assignee_user_id
		WHERE rt.organization_id=$1 AND rt.document_id=$2 AND t.status IN ('Open','InProgress')`, input.OrganizationID,
		input.DocumentID, "review-return-"+input.ID, "/o/"+input.OrganizationID+"/documents/"+input.DocumentID, input.ReturnedAt)
	if err != nil {
		return 0, err
	}
	return next, tx.Commit(ctx)
}
