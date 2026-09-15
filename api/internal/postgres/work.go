package postgres

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"plaiflow/api/internal/tenant"
	"plaiflow/api/internal/work"
)

const taskColumns = `t.id,t.organization_id,t.title,t.description,t.creator_user_id,
    coalesce(t.assignee_user_id::text,''),coalesce(a.display_name,''),t.status,t.priority,coalesce(t.due_on::text,''),
    (t.status IN ('Open','InProgress') AND t.due_on < (now() AT TIME ZONE o.timezone)::date),
    t.created_at,t.updated_at,t.status_changed_at,t.completed_at,
    coalesce(array_agg(wu.display_name ORDER BY lower(wu.display_name),wu.id) FILTER (WHERE wu.id IS NOT NULL),ARRAY[]::text[])`

const taskJoins = ` FROM tasks t
    JOIN organizations o ON o.id=t.organization_id
    LEFT JOIN users a ON a.id=t.assignee_user_id
    LEFT JOIN task_watchers tw ON tw.organization_id=t.organization_id AND tw.task_id=t.id
    LEFT JOIN users wu ON wu.id=tw.user_id `

const taskGroup = ` GROUP BY t.id,o.timezone,a.display_name `

func scanTask(row pgx.Row) (work.Task, error) {
	var task work.Task
	err := row.Scan(&task.ID, &task.OrganizationID, &task.Title, &task.Description, &task.CreatorUserID,
		&task.AssigneeUserID, &task.AssigneeName, &task.Status, &task.Priority, &task.DueOn, &task.IsOverdue,
		&task.CreatedAt, &task.UpdatedAt, &task.StatusChangedAt, &task.CompletedAt, &task.WatcherNames)
	return task, err
}

func (s *Store) CreateTask(ctx context.Context, command work.CreateTask) (work.Task, error) {
	tx, err := s.organizationTx(ctx, command.ActorUserID, command.OrganizationID)
	if err != nil {
		return work.Task{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := membershipFor(ctx, tx, command.ActorUserID, command.OrganizationID); err != nil {
		return work.Task{}, work.ErrNotFound
	}
	if command.AssigneeUserID != "" && !memberExists(ctx, tx, command.OrganizationID, command.AssigneeUserID) {
		return work.Task{}, work.ErrNotFound
	}
	_, err = tx.Exec(ctx, `INSERT INTO tasks
        (id,organization_id,title,description,creator_user_id,assignee_user_id,status,priority,due_on,created_at,updated_at,status_changed_at)
        VALUES ($1,$2,$3,$4,$5,nullif($6,'')::uuid,'Open',$7,nullif($8,'')::date,$9,$9,$9)`,
		command.ID, command.OrganizationID, command.Title, command.Description, command.ActorUserID, command.AssigneeUserID, command.Priority, command.DueOn, command.Now)
	if err != nil {
		return work.Task{}, mapWorkError(err)
	}
	payload := map[string]any{"task_id": command.ID}
	if command.AssigneeUserID != "" {
		payload["assignee_user_id"] = command.AssigneeUserID
	}
	if err := insertDomainEvent(ctx, tx, command.OrganizationID, command.ID, command.ActorUserID, "task.created", payload, command.Now); err != nil {
		return work.Task{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return work.Task{}, err
	}
	return s.GetTask(ctx, command.ActorUserID, command.OrganizationID, command.ID)
}

func (s *Store) GetTask(ctx context.Context, userID, organizationID, taskID string) (work.Task, error) {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return work.Task{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := membershipFor(ctx, tx, userID, organizationID); err != nil {
		return work.Task{}, work.ErrNotFound
	}
	task, err := scanTask(tx.QueryRow(ctx, `SELECT `+taskColumns+taskJoins+` WHERE t.organization_id=$1 AND t.id=$2 `+taskGroup, organizationID, taskID))
	if errors.Is(err, pgx.ErrNoRows) {
		return work.Task{}, work.ErrNotFound
	}
	if err != nil {
		return work.Task{}, err
	}
	return task, tx.Commit(ctx)
}

func (s *Store) ListTasks(ctx context.Context, userID, organizationID string, filter work.TaskFilter) (work.TaskPage, error) {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return work.TaskPage{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := membershipFor(ctx, tx, userID, organizationID); err != nil {
		return work.TaskPage{}, work.ErrNotFound
	}
	cursorTime, cursorID, err := decodeCursor(filter.Cursor)
	if err != nil {
		return work.TaskPage{}, work.ErrConflict
	}
	rows, err := tx.Query(ctx, `SELECT `+taskColumns+taskJoins+`
        WHERE t.organization_id=$1
          AND ($2='' OR t.status=$2) AND ($3='' OR t.priority=$3)
          AND ($4='' OR t.assignee_user_id=nullif($4,'')::uuid) AND ($5='' OR t.creator_user_id=nullif($5,'')::uuid)
          AND ($6='' OR t.due_on>=nullif($6,'')::date) AND ($7='' OR t.due_on<=nullif($7,'')::date)
          AND ($8='' OR (t.created_at AT TIME ZONE o.timezone)::date>=nullif($8,'')::date)
          AND ($9='' OR (t.created_at AT TIME ZONE o.timezone)::date<=nullif($9,'')::date)
          AND (NOT $10 OR (t.status IN ('Open','InProgress') AND t.due_on < (now() AT TIME ZONE o.timezone)::date)=$11)
          AND ($12::timestamptz IS NULL OR (t.created_at,t.id)<($12,$13::uuid))
        `+taskGroup+` ORDER BY t.created_at DESC,t.id DESC LIMIT $14`,
		organizationID, filter.Status, filter.Priority, filter.AssigneeUserID, filter.CreatorUserID, filter.DueFrom, filter.DueTo,
		filter.CreatedFrom, filter.CreatedTo, filter.Overdue != nil, boolValue(filter.Overdue), cursorTime, nilIfEmpty(cursorID), filter.Limit+1)
	if err != nil {
		return work.TaskPage{}, err
	}
	defer rows.Close()
	tasks := make([]work.Task, 0, filter.Limit)
	for rows.Next() {
		task, scanErr := scanTask(rows)
		if scanErr != nil {
			return work.TaskPage{}, scanErr
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return work.TaskPage{}, err
	}
	page := work.TaskPage{Tasks: tasks}
	if len(tasks) > filter.Limit {
		last := tasks[filter.Limit-1]
		page.Tasks = tasks[:filter.Limit]
		page.NextCursor = encodeCursor(last.CreatedAt, last.ID)
	}
	return page, tx.Commit(ctx)
}

func (s *Store) UpdateTask(ctx context.Context, command work.UpdateTask) (work.Task, error) {
	tx, err := s.organizationTx(ctx, command.ActorUserID, command.OrganizationID)
	if err != nil {
		return work.Task{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, command.ActorUserID, command.OrganizationID)
	if err != nil {
		return work.Task{}, work.ErrNotFound
	}
	var creator, assignee, title, description, dueOn string
	var status work.Status
	var priority work.Priority
	err = tx.QueryRow(ctx, `SELECT creator_user_id,coalesce(assignee_user_id::text,''),title,description,status,priority,coalesce(due_on::text,'')
        FROM tasks WHERE organization_id=$1 AND id=$2 FOR UPDATE`, command.OrganizationID, command.ID).Scan(
		&creator, &assignee, &title, &description, &status, &priority, &dueOn)
	if errors.Is(err, pgx.ErrNoRows) {
		return work.Task{}, work.ErrNotFound
	}
	if err != nil {
		return work.Task{}, err
	}
	manager := role == tenant.Owner || role == tenant.Admin
	canEdit := manager || creator == command.ActorUserID
	canStatus := manager || assignee == command.ActorUserID
	if !canEdit && (command.TitleSet || command.DescriptionSet || command.AssigneeSet || command.DueOnSet || command.PrioritySet) ||
		command.StatusSet && !canStatus {
		return work.Task{}, work.ErrForbidden
	}
	if !command.TitleSet {
		command.Title = title
	}
	if !command.DescriptionSet {
		command.Description = description
	}
	if !command.AssigneeSet {
		command.AssigneeUserID = assignee
	}
	if !command.DueOnSet {
		command.DueOn = dueOn
	}
	if !command.StatusSet {
		command.Status = status
	}
	if !command.PrioritySet {
		command.Priority = priority
	}
	if command.AssigneeUserID != "" && !memberExists(ctx, tx, command.OrganizationID, command.AssigneeUserID) {
		return work.Task{}, work.ErrNotFound
	}
	completedAt := any(nil)
	completedBy := any(nil)
	if command.Status == work.Done {
		completedAt, completedBy = command.Now, command.ActorUserID
	}
	_, err = tx.Exec(ctx, `UPDATE tasks SET title=$3,description=$4,assignee_user_id=nullif($5,'')::uuid,status=$6,priority=$7,
        due_on=nullif($8,'')::date,updated_at=$9,status_changed_at=CASE WHEN status<>$6 THEN $9 ELSE status_changed_at END,
        completed_at=$10,completed_by_user_id=$11 WHERE organization_id=$1 AND id=$2`,
		command.OrganizationID, command.ID, command.Title, command.Description, command.AssigneeUserID, command.Status, command.Priority, command.DueOn, command.Now, completedAt, completedBy)
	if err != nil {
		return work.Task{}, mapWorkError(err)
	}
	if command.AssigneeUserID != "" {
		if _, err := tx.Exec(ctx, `DELETE FROM task_watchers WHERE organization_id=$1 AND task_id=$2 AND user_id=$3`, command.OrganizationID, command.ID, command.AssigneeUserID); err != nil {
			return work.Task{}, err
		}
	}
	changedFields := make([]string, 0, 2)
	if command.TitleSet && command.Title != title {
		changedFields = append(changedFields, "title")
	}
	if command.DescriptionSet && command.Description != description {
		changedFields = append(changedFields, "description")
	}
	changes := []struct {
		changed  bool
		typeName string
		payload  map[string]any
	}{
		{len(changedFields) > 0, "task.details_changed", map[string]any{"changed_fields": changedFields}},
		{command.AssigneeSet && command.AssigneeUserID != assignee, assignmentEvent(command.AssigneeUserID), map[string]any{"assignee_user_id": nilIfEmpty(command.AssigneeUserID), "previous_assignee_user_id": nilIfEmpty(assignee)}},
		{command.DueOnSet && command.DueOn != dueOn, "task.due_changed", map[string]any{"due_changed": true}},
		{command.PrioritySet && command.Priority != priority, "task.priority_changed", map[string]any{"priority": command.Priority}},
		{command.StatusSet && command.Status != status, "task.status_changed", map[string]any{"status": command.Status, "previous_status": status}},
	}
	for _, change := range changes {
		if change.changed && change.typeName != "" {
			if err := insertDomainEvent(ctx, tx, command.OrganizationID, command.ID, command.ActorUserID, change.typeName, change.payload, command.Now); err != nil {
				return work.Task{}, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return work.Task{}, err
	}
	return s.GetTask(ctx, command.ActorUserID, command.OrganizationID, command.ID)
}

func (s *Store) SetWatcher(ctx context.Context, organizationID, actorUserID, taskID, watcherUserID string, add bool, now time.Time) error {
	tx, err := s.organizationTx(ctx, actorUserID, organizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, actorUserID, organizationID)
	if err != nil {
		return work.ErrNotFound
	}
	var creator string
	if err := tx.QueryRow(ctx, `SELECT creator_user_id FROM tasks WHERE organization_id=$1 AND id=$2`, organizationID, taskID).Scan(&creator); errors.Is(err, pgx.ErrNoRows) {
		return work.ErrNotFound
	} else if err != nil {
		return err
	}
	if role != tenant.Owner && role != tenant.Admin && creator != actorUserID {
		return work.ErrForbidden
	}
	eventType := "task.watcher_removed"
	if add {
		if !memberExists(ctx, tx, organizationID, watcherUserID) {
			return work.ErrNotFound
		}
		command, err := tx.Exec(ctx, `INSERT INTO task_watchers (organization_id,task_id,user_id,created_at)
            VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`, organizationID, taskID, watcherUserID, now)
		if err != nil {
			return mapWorkError(err)
		}
		if command.RowsAffected() == 0 {
			return tx.Commit(ctx)
		}
		eventType = "task.watcher_added"
	} else {
		command, err := tx.Exec(ctx, `DELETE FROM task_watchers WHERE organization_id=$1 AND task_id=$2 AND user_id=$3`, organizationID, taskID, watcherUserID)
		if err != nil {
			return err
		}
		if command.RowsAffected() == 0 {
			return tx.Commit(ctx)
		}
	}
	if err := insertDomainEvent(ctx, tx, organizationID, taskID, actorUserID, eventType, map[string]any{"watcher_user_id": watcherUserID}, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func memberExists(ctx context.Context, tx pgx.Tx, organizationID, userID string) bool {
	var exists bool
	_ = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM memberships WHERE organization_id=$1 AND user_id=$2)`, organizationID, userID).Scan(&exists)
	return exists
}

func insertDomainEvent(ctx context.Context, tx pgx.Tx, organizationID, taskID, actorUserID, eventType string, payload map[string]any, now time.Time) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO domain_events
        (organization_id,event_type,aggregate_id,actor_user_id,payload,occurred_at,available_at)
        VALUES ($1,$2,$3,$4,$5,$6,$6)`, organizationID, eventType, taskID, actorUserID, encoded, now)
	return err
}

func assignmentEvent(assignee string) string {
	if assignee == "" {
		return "task.unassigned"
	}
	return "task.assigned"
}

func mapWorkError(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "violates") || strings.Contains(err.Error(), "assignee cannot") {
		return work.ErrConflict
	}
	return err
}

func encodeCursor(created time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(created.UTC().Format(time.RFC3339Nano) + "|" + id))
}

func decodeCursor(value string) (*time.Time, string, error) {
	if value == "" {
		return nil, "", nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, "", err
	}
	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 || parts[1] == "" {
		return nil, "", errors.New("invalid cursor")
	}
	parsed, err := time.Parse(time.RFC3339Nano, parts[0])
	return &parsed, parts[1], err
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func nilIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
