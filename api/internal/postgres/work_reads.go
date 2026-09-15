package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"plaiflow/api/internal/tenant"
	"plaiflow/api/internal/work"
)

func (s *Store) ListNotifications(ctx context.Context, userID, organizationID, cursor string, limit int) (work.NotificationPage, error) {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return work.NotificationPage{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := membershipFor(ctx, tx, userID, organizationID); err != nil {
		return work.NotificationPage{}, work.ErrNotFound
	}
	cursorTime, cursorID, err := decodeCursor(cursor)
	if err != nil {
		return work.NotificationPage{}, work.ErrConflict
	}
	rows, err := tx.Query(ctx, `SELECT id,organization_id,recipient_user_id,category,title,deep_link,created_at,read_at
		FROM notifications WHERE organization_id=$1 AND recipient_user_id=$2
		  AND (delivery_mode='Immediate' OR (delivery_mode='Digest' AND digested_at IS NOT NULL))
          AND ($3::timestamptz IS NULL OR (created_at,id)<($3,$4::uuid))
        ORDER BY created_at DESC,id DESC LIMIT $5`, organizationID, userID, cursorTime, nilIfEmpty(cursorID), limit+1)
	if err != nil {
		return work.NotificationPage{}, err
	}
	defer rows.Close()
	items := make([]work.Notification, 0, limit)
	for rows.Next() {
		var item work.Notification
		if err := rows.Scan(&item.ID, &item.OrganizationID, &item.RecipientUserID, &item.Category, &item.Title, &item.DeepLink, &item.CreatedAt, &item.ReadAt); err != nil {
			return work.NotificationPage{}, err
		}
		items = append(items, item)
	}
	page := work.NotificationPage{Notifications: items}
	if len(items) > limit {
		last := items[limit-1]
		page.Notifications = items[:limit]
		page.NextCursor = encodeCursor(last.CreatedAt, last.ID)
	}
	return page, tx.Commit(ctx)
}

func (s *Store) MarkNotificationRead(ctx context.Context, userID, organizationID, notificationID string, now time.Time) error {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	command, err := tx.Exec(ctx, `UPDATE notifications SET read_at=coalesce(read_at,$4)
        WHERE id=$1 AND organization_id=$2 AND recipient_user_id=$3`, notificationID, organizationID, userID, now)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return work.ErrNotFound
	}
	return tx.Commit(ctx)
}

func (s *Store) MarkAllNotificationsRead(ctx context.Context, userID, organizationID string, now time.Time) error {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := membershipFor(ctx, tx, userID, organizationID); err != nil {
		return work.ErrNotFound
	}
	if _, err := tx.Exec(ctx, `UPDATE notifications SET read_at=$3
        WHERE organization_id=$1 AND recipient_user_id=$2 AND read_at IS NULL`, organizationID, userID, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ListPreferences(ctx context.Context, userID, organizationID string) ([]work.Preference, error) {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err := membershipFor(ctx, tx, userID, organizationID); err != nil {
		return nil, work.ErrNotFound
	}
	rows, err := tx.Query(ctx, `WITH defaults(category,web_mode,line_mode) AS (VALUES
        ('Assignment','Immediate','Off'),('TaskChange','Digest','Off'),('DueReminder','Immediate','Off'))
        SELECT $1::uuid,$2::uuid,d.category,coalesce(p.web_mode,d.web_mode),coalesce(p.line_mode,d.line_mode),coalesce(p.updated_at,now())
        FROM defaults d LEFT JOIN notification_preferences p
          ON p.organization_id=$1 AND p.user_id=$2 AND p.category=d.category ORDER BY d.category`, organizationID, userID)
	if err != nil {
		return nil, err
	}
	preferences, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (work.Preference, error) {
		var preference work.Preference
		err := row.Scan(&preference.OrganizationID, &preference.UserID, &preference.Category, &preference.WebMode, &preference.LineMode, &preference.UpdatedAt)
		return preference, err
	})
	if err != nil {
		return nil, err
	}
	return preferences, tx.Commit(ctx)
}

func (s *Store) SetPreference(ctx context.Context, preference work.Preference) error {
	tx, err := s.organizationTx(ctx, preference.UserID, preference.OrganizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := membershipFor(ctx, tx, preference.UserID, preference.OrganizationID); err != nil {
		return work.ErrNotFound
	}
	_, err = tx.Exec(ctx, `INSERT INTO notification_preferences (organization_id,user_id,category,web_mode,line_mode,updated_at)
        VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (organization_id,user_id,category)
        DO UPDATE SET web_mode=excluded.web_mode,line_mode=excluded.line_mode,updated_at=excluded.updated_at`,
		preference.OrganizationID, preference.UserID, preference.Category, preference.WebMode, preference.LineMode, preference.UpdatedAt)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) Dashboard(ctx context.Context, userID, organizationID string) (work.Dashboard, error) {
	started := time.Now()
	defer s.slow(started, "work_dashboard")
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return work.Dashboard{}, err
	}
	defer tx.Rollback(ctx)
	var result work.Dashboard
	result.Scope = organizationID
	err = tx.QueryRow(ctx, `SELECT m.role,
        count(DISTINCT t.id) FILTER (WHERE t.status IN ('Open','InProgress')),
        count(DISTINCT t.id) FILTER (WHERE t.status IN ('Open','InProgress') AND t.due_on<(now() AT TIME ZONE o.timezone)::date),
		(SELECT count(*) FROM notifications n WHERE n.organization_id=$1 AND n.recipient_user_id=$2 AND n.read_at IS NULL
		  AND (n.delivery_mode='Immediate' OR (n.delivery_mode='Digest' AND n.digested_at IS NOT NULL)))
        FROM memberships m JOIN organizations o ON o.id=m.organization_id
        LEFT JOIN tasks t ON t.organization_id=m.organization_id
        WHERE m.organization_id=$1 AND m.user_id=$2 GROUP BY m.role,o.timezone`, organizationID, userID).Scan(
		&result.Role, &result.OpenCount, &result.OverdueCount, &result.UnreadCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return work.Dashboard{}, work.ErrNotFound
	}
	if err != nil {
		return work.Dashboard{}, err
	}
	result.MyFocus, err = dashboardTasks(ctx, tx, organizationID, `t.assignee_user_id=$2`, userID)
	if err != nil {
		return work.Dashboard{}, err
	}
	result.Watching, err = dashboardTasks(ctx, tx, organizationID, `EXISTS (SELECT 1 FROM task_watchers x WHERE x.organization_id=t.organization_id AND x.task_id=t.id AND x.user_id=$2)`, userID)
	if err != nil {
		return work.Dashboard{}, err
	}
	if result.Role == tenant.Owner || result.Role == tenant.Admin {
		result.TeamQueue, err = dashboardTasks(ctx, tx, organizationID, `t.assignee_user_id IS NULL`, userID)
		if err != nil {
			return work.Dashboard{}, err
		}
	}
	return result, tx.Commit(ctx)
}

func dashboardTasks(ctx context.Context, tx pgx.Tx, organizationID, condition, userID string) ([]work.Task, error) {
	rows, err := tx.Query(ctx, `SELECT `+taskColumns+taskJoins+` WHERE t.organization_id=$1 AND t.status IN ('Open','InProgress') AND `+condition+taskGroup+`
        ORDER BY t.due_on NULLS LAST,t.priority DESC,t.id LIMIT 10`, organizationID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]work.Task, 0, 10)
	for rows.Next() {
		item, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

const exportWhere = ` FROM tasks t JOIN organizations o ON o.id=t.organization_id
    WHERE t.organization_id=$1 AND ($2='' OR t.status=$2) AND ($3='' OR t.priority=$3)
      AND ($4='' OR t.assignee_user_id=nullif($4,'')::uuid) AND ($5='' OR t.creator_user_id=nullif($5,'')::uuid)
      AND ($6='' OR t.due_on>=nullif($6,'')::date) AND ($7='' OR t.due_on<=nullif($7,'')::date)
      AND ($8='' OR (t.created_at AT TIME ZONE o.timezone)::date>=nullif($8,'')::date)
      AND ($9='' OR (t.created_at AT TIME ZONE o.timezone)::date<=nullif($9,'')::date)
      AND (NOT $10 OR (t.status IN ('Open','InProgress') AND t.due_on<(now() AT TIME ZONE o.timezone)::date)=$11) `

func exportArgs(organizationID string, filter work.TaskFilter) []any {
	return []any{organizationID, filter.Status, filter.Priority, filter.AssigneeUserID, filter.CreatorUserID, filter.DueFrom, filter.DueTo,
		filter.CreatedFrom, filter.CreatedTo, filter.Overdue != nil, boolValue(filter.Overdue)}
}

func (s *Store) CountExport(ctx context.Context, userID, organizationID string, filter work.TaskFilter) (int64, error) {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, userID, organizationID)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return 0, work.ErrForbidden
	}
	var count int64
	if err := tx.QueryRow(ctx, `SELECT count(*)`+exportWhere, exportArgs(organizationID, filter)...).Scan(&count); err != nil {
		return 0, err
	}
	return count, tx.Commit(ctx)
}

func (s *Store) ExportRows(ctx context.Context, userID, organizationID string, filter work.TaskFilter, emit func(work.ExportRow) error) error {
	started := time.Now()
	defer s.slow(started, "task_export")
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, userID, organizationID)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return work.ErrForbidden
	}
	query := `SELECT t.organization_id,o.name,t.id,t.title,t.description,t.status,t.priority,coalesce(t.due_on::text,''),
        (t.status IN ('Open','InProgress') AND t.due_on<(now() AT TIME ZONE o.timezone)::date),
        coalesce(c.display_name,''),coalesce(a.display_name,''),
        coalesce(string_agg(wu.display_name,'|' ORDER BY lower(wu.display_name),wu.id) FILTER (WHERE wu.id IS NOT NULL),''),
        t.created_at,t.updated_at,t.status_changed_at,t.completed_at
        FROM tasks t JOIN organizations o ON o.id=t.organization_id JOIN users c ON c.id=t.creator_user_id
        LEFT JOIN users a ON a.id=t.assignee_user_id LEFT JOIN task_watchers tw ON tw.organization_id=t.organization_id AND tw.task_id=t.id
        LEFT JOIN users wu ON wu.id=tw.user_id` + exportWhere[len(` FROM tasks t JOIN organizations o ON o.id=t.organization_id`):] + `
        GROUP BY t.id,o.name,o.timezone,c.display_name,a.display_name ORDER BY t.created_at DESC,t.id DESC`
	rows, err := tx.Query(ctx, query, exportArgs(organizationID, filter)...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var row work.ExportRow
		if err := rows.Scan(&row.OrganizationID, &row.OrganizationName, &row.TaskID, &row.Title, &row.Description, &row.Status, &row.Priority,
			&row.DueDate, &row.IsOverdue, &row.CreatorName, &row.AssigneeName, &row.WatcherNames, &row.CreatedAt, &row.UpdatedAt,
			&row.StatusChangedAt, &row.CompletedAt); err != nil {
			return err
		}
		if err := emit(row); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) AuditExport(ctx context.Context, request work.ExportRequest, lifecycle string, rows, bytes int64, reason string) error {
	tx, err := s.organizationTx(ctx, request.ActorUserID, request.OrganizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, request.ActorUserID, request.OrganizationID)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return work.ErrForbidden
	}
	filters, _ := json.Marshal(request.Filters)
	if lifecycle == "requested" {
		_, err = tx.Exec(ctx, `INSERT INTO export_jobs
            (id,organization_id,requester_user_id,filters,mode,status,requested_at,started_at,authorized_row_count,data_as_of)
            VALUES ($1,$2,$3,$4,'sync','Running',$5,$5,0,$5)`, request.ID, request.OrganizationID, request.ActorUserID, filters, request.Now)
	} else {
		status := "Failed"
		if lifecycle == "completed" {
			status = "Completed"
		}
		_, err = tx.Exec(ctx, `UPDATE export_jobs SET status=$3,authorized_row_count=$4,produced_byte_count=$5,
            completed_at=CASE WHEN $3='Completed' THEN $6 END,failed_at=CASE WHEN $3='Failed' THEN $6 END,failure_code=nullif($7,'')
            WHERE organization_id=$1 AND id=$2`, request.OrganizationID, request.ID, status, rows, bytes, request.Now, reason)
	}
	if err != nil {
		return err
	}
	metadata := map[string]any{"filters": request.Filters, "row_count": rows, "byte_count": bytes}
	encoded, _ := json.Marshal(metadata)
	_, err = tx.Exec(ctx, `INSERT INTO audit_events
        (organization_id,actor_user_id,event_type,target_type,target_id,request_id,outcome,reason_code,metadata,occurred_at)
        VALUES ($1,$2,$3,'export',$4,$5,$6,nullif($7,''),$8,$9)`, request.OrganizationID, request.ActorUserID,
		"export."+lifecycle, request.ID, request.RequestID, lifecycle, reason, encoded, request.Now)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) QueueExport(ctx context.Context, request work.ExportRequest, count int64) (work.ExportJob, error) {
	tx, err := s.organizationTx(ctx, request.ActorUserID, request.OrganizationID)
	if err != nil {
		return work.ExportJob{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, request.ActorUserID, request.OrganizationID)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return work.ExportJob{}, work.ErrForbidden
	}
	command, err := tx.Exec(ctx, `UPDATE export_jobs SET mode='background',status='Queued',started_at=NULL,data_as_of=NULL,
        authorized_row_count=$3 WHERE organization_id=$1 AND id=$2 AND requester_user_id=$4`, request.OrganizationID, request.ID, count, request.ActorUserID)
	if err != nil {
		return work.ExportJob{}, err
	}
	if command.RowsAffected() != 1 {
		return work.ExportJob{}, work.ErrNotFound
	}
	job := work.ExportJob{ID: request.ID, OrganizationID: request.OrganizationID, RequesterUserID: request.ActorUserID, Status: "Queued", Mode: "background", RowCount: count, RequestedAt: request.Now}
	return job, tx.Commit(ctx)
}

func (s *Store) ListExports(ctx context.Context, userID, organizationID, cursor string, limit int) (work.ExportPage, error) {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return work.ExportPage{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, userID, organizationID)
	if err != nil || role != tenant.Owner && role != tenant.Admin {
		return work.ExportPage{}, work.ErrForbidden
	}
	cursorTime, cursorID, err := decodeCursor(cursor)
	if err != nil {
		return work.ExportPage{}, work.ErrConflict
	}
	rows, err := tx.Query(ctx, `SELECT id,organization_id,requester_user_id,status,mode,coalesce(failure_code,''),
        authorized_row_count,coalesce(produced_byte_count,0),requested_at,started_at,completed_at,object_expires_at
        FROM export_jobs WHERE organization_id=$1 AND ($2::timestamptz IS NULL OR (requested_at,id)<($2,$3::uuid))
        ORDER BY requested_at DESC,id DESC LIMIT $4`, organizationID, cursorTime, nilIfEmpty(cursorID), limit+1)
	if err != nil {
		return work.ExportPage{}, err
	}
	defer rows.Close()
	items := make([]work.ExportJob, 0, limit)
	for rows.Next() {
		var item work.ExportJob
		if err := rows.Scan(&item.ID, &item.OrganizationID, &item.RequesterUserID, &item.Status, &item.Mode, &item.FailureCode,
			&item.RowCount, &item.ByteCount, &item.RequestedAt, &item.StartedAt, &item.CompletedAt, &item.ExpiresAt); err != nil {
			return work.ExportPage{}, err
		}
		items = append(items, item)
	}
	page := work.ExportPage{Exports: items}
	if len(items) > limit {
		last := items[limit-1]
		page.Exports = items[:limit]
		page.NextCursor = encodeCursor(last.RequestedAt, last.ID)
	}
	return page, tx.Commit(ctx)
}
