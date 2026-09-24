package postgres

import (
	"context"
	"time"

	"plaiflow/api/internal/document"
	"plaiflow/api/internal/extraction"
	"plaiflow/api/internal/tenant"
)

func (s *Store) ListReviewQueue(ctx context.Context, user, org, cursorValue, view, status string, limit int) (extraction.QueuePage, error) {
	if limit < 1 || limit > 50 || view != "all" && view != "mine" && view != "unassigned" || status != "Available" && status != "Archived" {
		return extraction.QueuePage{}, document.ErrInvalidCursor
	}
	cursor, err := decodeDocumentCursor(cursorValue)
	if err != nil {
		return extraction.QueuePage{}, err
	}
	tx, err := s.organizationTx(ctx, user, org)
	if err != nil {
		return extraction.QueuePage{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, user, org)
	if err != nil {
		return extraction.QueuePage{}, tenant.ErrNotFound
	}
	manager := role == tenant.Owner || role == tenant.Admin
	var cursorAt, cursorID any
	if cursorValue != "" {
		cursorAt, cursorID = cursor.At, cursor.ID
	}
	rows, err := tx.Query(ctx, `SELECT d.id,d.status,d.accepted_at,o.job_id,coalesce(e.revision,0),coalesce(rd.revision,0),
		coalesce(t.id::text,''),coalesce(am.user_id::text,'')
		FROM documents d
		JOIN LATERAL (SELECT job_id FROM document_ocr_runs WHERE organization_id=d.organization_id AND document_id=d.id
			AND published_at IS NOT NULL AND superseded_at IS NULL AND deleted_at IS NULL
			ORDER BY published_at DESC,job_id DESC LIMIT 1) o ON true
		LEFT JOIN document_extraction_reviews e ON e.organization_id=d.organization_id AND e.document_id=d.id
			AND e.ocr_job_id=o.job_id AND e.superseded_at IS NULL
		LEFT JOIN document_review_drafts rd ON rd.organization_id=d.organization_id AND rd.document_id=d.id
		LEFT JOIN accounting_suggestions a ON a.organization_id=d.organization_id AND a.document_id=d.id
			AND a.review_id=e.id AND a.superseded_at IS NULL AND coalesce(rd.revision,0)<=coalesce(e.draft_revision,0)
		LEFT JOIN document_review_tasks rt ON rt.organization_id=d.organization_id AND rt.document_id=d.id
		LEFT JOIN tasks t ON t.organization_id=rt.organization_id AND t.id=rt.task_id AND t.status IN ('Open','InProgress')
		LEFT JOIN memberships am ON am.organization_id=t.organization_id AND am.user_id=t.assignee_user_id
			AND (am.role IN ('Owner','Admin') OR (d.submitted_by_user_id=am.user_id AND NOT d.group_restricted)
				OR d.assignee_user_id=am.user_id OR EXISTS (SELECT 1 FROM document_sources ds
					WHERE ds.organization_id=d.organization_id AND ds.document_id=d.id
					AND ds.submitted_by_user_id=am.user_id AND NOT ds.group_source))
		WHERE d.organization_id=$1 AND d.status=$2 AND a.id IS NULL
		AND ($3::boolean OR (d.submitted_by_user_id=$4::uuid AND NOT d.group_restricted) OR d.assignee_user_id=$4::uuid
			OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id AND ds.document_id=d.id
				AND ds.submitted_by_user_id=$4::uuid AND NOT ds.group_source))
		AND ($5='all' OR ($5='mine' AND am.user_id=$4::uuid) OR ($5='unassigned' AND am.user_id IS NULL))
		AND ($6::boolean OR (d.accepted_at,d.id)>($7::timestamptz,$8::uuid))
		ORDER BY d.accepted_at,d.id LIMIT $9`, org, status, manager, user, view, cursorValue == "", cursorAt, cursorID, limit+1)
	if err != nil {
		return extraction.QueuePage{}, err
	}
	defer rows.Close()
	page := extraction.QueuePage{Items: []extraction.QueueItem{}}
	for rows.Next() {
		var item extraction.QueueItem
		if err = rows.Scan(&item.DocumentID, &item.Status, &item.AcceptedAt, &item.OCRJobID,
			&item.ReviewRevision, &item.DraftRevision, &item.TaskID, &item.AssigneeID); err != nil {
			return extraction.QueuePage{}, err
		}
		page.Items = append(page.Items, item)
	}
	if err = rows.Err(); err != nil {
		return extraction.QueuePage{}, err
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = encodeDocumentCursor(document.Document{ID: last.DocumentID, AcceptedAt: last.AcceptedAt})
	}
	if err := auditTenant(ctx, tx, org, user, "review.queue.read", "organization", org, time.Now().UTC()); err != nil {
		return extraction.QueuePage{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return extraction.QueuePage{}, err
	}
	return page, nil
}
