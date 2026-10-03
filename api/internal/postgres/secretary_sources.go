package postgres

import (
	"context"
	"errors"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"plaiflow/api/internal/secretary"
	"plaiflow/api/internal/tenant"
)

func (s *Store) Intent(ctx context.Context, userID, organizationID, intent string, now time.Time) (secretary.IntentAnswer, error) {
	answer := secretary.IntentAnswer{Intent: intent, Items: []secretary.Item{}}
	if intent == "missing-documents" {
		answer.State = "unconfigured"
		answer.Explanation = "ยังไม่ได้ตั้งค่ารายการเอกสารที่คาดว่าจะได้รับ"
		return answer, nil
	}
	if intent == "today" || intent == "why-first" {
		briefing, err := s.Open(ctx, userID, organizationID, now)
		if err != nil {
			return answer, err
		}
		answer.State = briefing.Status
		if briefing.Status != "Ready" {
			return answer, nil
		}
		answer.Items = briefing.Items
		if intent == "why-first" {
			if len(answer.Items) == 0 {
				answer.State = "empty"
				return answer, nil
			}
			answer.Items = answer.Items[:1]
			answer.Explanation = answer.Items[0].Reason
		}
		return answer, nil
	}
	if intent != "follow-up" && intent != "meeting-actions" {
		return answer, tenant.ErrNotFound
	}
	category := "FollowUp"
	if intent == "meeting-actions" {
		category = "MeetingAction"
	}
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return answer, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, userID, organizationID)
	if err != nil {
		return answer, tenant.ErrNotFound
	}
	rows, err := tx.Query(ctx, `SELECT t.id,t.title,coalesce(t.due_on::text,''),t.priority,
		coalesce(t.follow_up_with,''),coalesce(t.meeting_name,''),coalesce(t.meeting_on::text,'')
		FROM tasks t WHERE t.organization_id=$1 AND t.status IN ('Open','InProgress')
		AND t.secretary_category=$3 AND ($4 IN ('Owner','Admin') OR t.assignee_user_id=$2 OR EXISTS
		(SELECT 1 FROM task_watchers w WHERE w.organization_id=t.organization_id AND w.task_id=t.id AND w.user_id=$2))
		ORDER BY t.due_on NULLS LAST,CASE t.priority WHEN 'Urgent' THEN 0 WHEN 'High' THEN 1 ELSE 2 END,t.id LIMIT 20`,
		organizationID, userID, category, role)
	if err != nil {
		return answer, err
	}
	defer rows.Close()
	for rows.Next() {
		var item secretary.Item
		var followUpWith, meetingName, meetingOn string
		if err = rows.Scan(&item.SourceID, &item.Title, &item.DueOn, &item.Priority, &followUpWith, &meetingName, &meetingOn); err != nil {
			return answer, err
		}
		item.Category = category
		item.URL = "/o/" + url.PathEscape(organizationID) + "/tasks/" + url.PathEscape(item.SourceID)
		if category == "FollowUp" {
			item.Reason = "ติดตาม: " + followUpWith
		} else {
			item.Reason = "จากประชุม: " + meetingName + " · " + meetingOn
		}
		answer.Items = append(answer.Items, item)
	}
	if err = rows.Err(); err != nil {
		return answer, err
	}
	if err = tx.Commit(ctx); err != nil {
		return answer, err
	}
	answer.State = "Ready"
	if len(answer.Items) == 0 {
		answer.State = "empty"
	}
	return answer, nil
}

func (s *Store) GetTaskSource(ctx context.Context, userID, organizationID, taskID string) (secretary.TaskSource, error) {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return secretary.TaskSource{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, userID, organizationID)
	if err != nil {
		return secretary.TaskSource{}, tenant.ErrNotFound
	}
	var source secretary.TaskSource
	err = tx.QueryRow(ctx, `SELECT id,secretary_category,follow_up_with,meeting_name,coalesce(meeting_on::text,'')
		FROM tasks WHERE organization_id=$1 AND id=$2 AND ($3 IN ('Owner','Admin') OR creator_user_id=$4 OR assignee_user_id=$4
		OR EXISTS(SELECT 1 FROM task_watchers w WHERE w.organization_id=tasks.organization_id
		AND w.task_id=tasks.id AND w.user_id=$4))`, organizationID, taskID, role, userID).Scan(
		&source.TaskID, &source.Category, &source.FollowUpWith, &source.MeetingName, &source.MeetingOn)
	if errors.Is(err, pgx.ErrNoRows) {
		return secretary.TaskSource{}, tenant.ErrNotFound
	}
	if err != nil {
		return secretary.TaskSource{}, err
	}
	return source, tx.Commit(ctx)
}

func (s *Store) SetTaskSource(ctx context.Context, command secretary.SourceCommand) (secretary.TaskSource, error) {
	tx, err := s.organizationTx(ctx, command.UserID, command.OrganizationID)
	if err != nil {
		return secretary.TaskSource{}, err
	}
	defer tx.Rollback(ctx)
	role, err := currentRole(ctx, tx, command.UserID, command.OrganizationID)
	if err != nil {
		return secretary.TaskSource{}, tenant.ErrNotFound
	}
	var creator string
	err = tx.QueryRow(ctx, `SELECT creator_user_id FROM tasks WHERE organization_id=$1 AND id=$2 FOR UPDATE`,
		command.OrganizationID, command.TaskID).Scan(&creator)
	if errors.Is(err, pgx.ErrNoRows) {
		return secretary.TaskSource{}, tenant.ErrNotFound
	}
	if err != nil {
		return secretary.TaskSource{}, err
	}
	if role != tenant.Owner && role != tenant.Admin && creator != command.UserID {
		return secretary.TaskSource{}, tenant.ErrForbidden
	}
	_, err = tx.Exec(ctx, `UPDATE tasks SET secretary_category=$3,follow_up_with=$4,meeting_name=$5,
		meeting_on=nullif($6,'')::date,updated_at=$7 WHERE organization_id=$1 AND id=$2`,
		command.OrganizationID, command.TaskID, command.Category, command.FollowUpWith, command.MeetingName, command.MeetingOn, command.Now.UTC())
	if err != nil {
		return secretary.TaskSource{}, err
	}
	if err = insertDomainEvent(ctx, tx, command.OrganizationID, command.TaskID, command.UserID,
		"task.details_changed", map[string]any{"changed_fields": []string{"secretary_source"}}, command.Now.UTC()); err != nil {
		return secretary.TaskSource{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return secretary.TaskSource{}, err
	}
	return command.TaskSource, nil
}
