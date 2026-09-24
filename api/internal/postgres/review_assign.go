package postgres

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"plaiflow/api/internal/extraction"
	"plaiflow/api/internal/tenant"
)

func (s *Store) AssignReviewTasks(ctx context.Context, actor, org, assignee string, items []extraction.Assignment, requestID string, now time.Time) ([]string, error) {
	if len(items) < 1 || len(items) > 50 {
		return nil, extraction.ErrConflict
	}
	items = append([]extraction.Assignment(nil), items...)
	sort.Slice(items, func(i, j int) bool { return items[i].DocumentID < items[j].DocumentID })
	for i, item := range items {
		if !documentUUID.MatchString(item.DocumentID) || !documentUUID.MatchString(item.OCRJobID) ||
			!documentUUID.MatchString(item.TaskID) || item.ReviewRevision < 0 || item.DraftRevision < 0 ||
			i > 0 && item.DocumentID == items[i-1].DocumentID {
			return nil, extraction.ErrConflict
		}
	}
	tx, err := s.organizationTx(ctx, actor, org)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actor, org)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return nil, tenant.ErrForbidden
	}
	if assignee != "" && !documentUUID.MatchString(assignee) {
		return nil, extraction.ErrConflict
	}
	// Lock each document in a stable order, and validate every selector before changing a Task.
	for _, item := range items {
		var status, ocrID string
		var reviewRevision, draftRevision int
		err = tx.QueryRow(ctx, `SELECT d.status,o.job_id,coalesce(e.revision,0),coalesce(rd.revision,0)
			FROM documents d JOIN LATERAL (SELECT job_id FROM document_ocr_runs WHERE organization_id=d.organization_id
				AND document_id=d.id AND published_at IS NOT NULL AND superseded_at IS NULL AND deleted_at IS NULL
				ORDER BY published_at DESC,job_id DESC LIMIT 1) o ON true
			LEFT JOIN document_extraction_reviews e ON e.organization_id=d.organization_id AND e.document_id=d.id
				AND e.ocr_job_id=o.job_id AND e.superseded_at IS NULL
			LEFT JOIN document_review_drafts rd ON rd.organization_id=d.organization_id AND rd.document_id=d.id AND rd.ocr_job_id=o.job_id
			WHERE d.organization_id=$1 AND d.id=$2 FOR UPDATE OF d`, org, item.DocumentID).Scan(&status, &ocrID, &reviewRevision, &draftRevision)
		if err != nil || status != "Available" && status != "Archived" || ocrID != item.OCRJobID ||
			reviewRevision != item.ReviewRevision || draftRevision != item.DraftRevision {
			return nil, extraction.ErrConflict
		}
		if assignee != "" {
			var allowed bool
			err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM memberships m JOIN documents d ON d.organization_id=m.organization_id
				WHERE m.organization_id=$1 AND m.user_id=$2 AND d.id=$3
				AND (m.role IN ('Owner','Admin') OR (d.submitted_by_user_id=m.user_id AND NOT d.group_restricted)
					OR d.assignee_user_id=m.user_id OR EXISTS (SELECT 1 FROM document_sources ds
						WHERE ds.organization_id=d.organization_id AND ds.document_id=d.id AND ds.submitted_by_user_id=m.user_id
						AND NOT ds.group_source)))`, org, assignee, item.DocumentID).Scan(&allowed)
			if err != nil || !allowed {
				return nil, extraction.ErrConflict
			}
		}
	}
	created := make([]string, 0, len(items))
	for _, item := range items {
		var activeTask string
		err = tx.QueryRow(ctx, `SELECT t.id FROM document_review_tasks rt JOIN tasks t ON t.organization_id=rt.organization_id
			AND t.id=rt.task_id WHERE rt.organization_id=$1 AND rt.document_id=$2 AND t.status IN ('Open','InProgress')
			FOR UPDATE OF t`, org, item.DocumentID).Scan(&activeTask)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		if activeTask == "" && assignee != "" {
			_, err = tx.Exec(ctx, `INSERT INTO tasks(id,organization_id,title,description,creator_user_id,assignee_user_id,
				status,priority,created_at,updated_at,status_changed_at)
				VALUES($1,$2,'Review document','',$3,$4,'Open','Normal',$5,$5,$5)`, item.TaskID, org, actor, assignee, now)
			if err != nil {
				return nil, err
			}
			_, err = tx.Exec(ctx, `INSERT INTO document_review_tasks(organization_id,document_id,task_id) VALUES($1,$2,$3)
				ON CONFLICT(organization_id,document_id) DO UPDATE SET task_id=excluded.task_id`, org, item.DocumentID, item.TaskID)
			if err != nil {
				return nil, err
			}
			if err = insertDomainEvent(ctx, tx, org, item.TaskID, actor, "task.created",
				map[string]any{"task_id": item.TaskID, "assignee_user_id": assignee}, now); err != nil {
				return nil, err
			}
			created = append(created, item.TaskID)
		} else if activeTask != "" {
			_, err = tx.Exec(ctx, `UPDATE tasks SET assignee_user_id=nullif($3,'')::uuid,updated_at=$4
				WHERE organization_id=$1 AND id=$2`, org, activeTask, assignee, now)
			if err != nil {
				return nil, err
			}
			event := "task.unassigned"
			if assignee != "" {
				event = "task.assigned"
			}
			if err = insertDomainEvent(ctx, tx, org, activeTask, actor, event,
				map[string]any{"task_id": activeTask, "assignee_user_id": assignee}, now); err != nil {
				return nil, err
			}
		}
		_, err = tx.Exec(ctx, `INSERT INTO audit_events(organization_id,actor_user_id,event_type,target_type,target_id,
			request_id,outcome,occurred_at,metadata) VALUES($1,$2,'review.assign','document',$3,$4,'success',$5,
			jsonb_build_object('ocr_job_id',$6,'review_revision',$7,'draft_revision',$8))`, org, actor,
			item.DocumentID, requestID, now, item.OCRJobID, item.ReviewRevision, item.DraftRevision)
		if err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
}
