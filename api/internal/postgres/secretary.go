package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"plaiflow/api/internal/job"
	"plaiflow/api/internal/secretary"
	"plaiflow/api/internal/tenant"
)

type secretaryRow struct {
	briefing  secretary.Briefing
	role      tenant.Role
	requested time.Time
}

func (s *Store) Open(ctx context.Context, userID, organizationID string, now time.Time) (secretary.Briefing, error) {
	return s.openSecretary(ctx, userID, organizationID, now, false)
}

func (s *Store) Refresh(ctx context.Context, userID, organizationID string, now time.Time) (secretary.Briefing, error) {
	return s.openSecretary(ctx, userID, organizationID, now, true)
}

func (s *Store) openSecretary(ctx context.Context, userID, organizationID string, now time.Time, refresh bool) (secretary.Briefing, error) {
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return secretary.Briefing{}, err
	}
	defer tx.Rollback(ctx)
	var role tenant.Role
	var timezone string
	// Serializes first opens and refreshes for one Membership, including role changes.
	err = tx.QueryRow(ctx, `SELECT m.role,o.timezone FROM memberships m JOIN organizations o ON o.id=m.organization_id
		WHERE m.organization_id=$1 AND m.user_id=$2 FOR UPDATE OF m`, organizationID, userID).Scan(&role, &timezone)
	if errors.Is(err, pgx.ErrNoRows) {
		return secretary.Briefing{}, tenant.ErrNotFound
	}
	if err != nil {
		return secretary.Briefing{}, err
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return secretary.Briefing{}, err
	}
	local := now.In(location)
	day := local.Format("2006-01-02")
	expires := time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, location).UTC()
	row, err := readSecretaryRow(ctx, tx, organizationID, userID, day)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return secretary.Briefing{}, err
	}
	missing := errors.Is(err, pgx.ErrNoRows)
	invalid := !missing && (row.role != role || !s.secretarySourcesAuthorized(ctx, tx, organizationID, userID, role, row.briefing.Items))
	if !missing && !invalid && row.briefing.Status == "Ready" {
		currentCount, countErr := countSecretaryCandidates(ctx, tx, organizationID, userID, day, role)
		if countErr != nil {
			return secretary.Briefing{}, countErr
		}
		invalid = currentCount != len(row.briefing.Items)+row.briefing.Remaining
	}
	if !missing && !invalid && row.briefing.Status == "Queued" {
		var jobStatus job.Status
		if err = tx.QueryRow(ctx, `SELECT status FROM durable_jobs WHERE id=$1`, row.briefing.JobID).Scan(&jobStatus); err != nil {
			return secretary.Briefing{}, err
		}
		if jobStatus == job.Failed {
			row.briefing.Status = "Failed"
		}
	}
	if refresh && !missing && !invalid && row.briefing.Status != "Failed" && now.Sub(row.requested) < 15*time.Minute {
		return secretary.Briefing{}, secretary.ErrTooSoon
	}
	if missing || invalid || refresh {
		row, err = queueSecretary(ctx, tx, userID, organizationID, day, role, expires, now, row.briefing.JobID)
		if err != nil {
			return secretary.Briefing{}, err
		}
	}
	if row.briefing.Status == "Ready" && row.briefing.GeneratedAt != nil {
		var stale bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tasks t WHERE t.organization_id=$1
			AND t.updated_at>$3 AND ($4 IN ('Owner','Admin') OR t.assignee_user_id=$2 OR EXISTS
			(SELECT 1 FROM task_watchers w WHERE w.organization_id=t.organization_id AND w.task_id=t.id AND w.user_id=$2)))`,
			organizationID, userID, *row.briefing.GeneratedAt, role).Scan(&stale)
		if err != nil {
			return secretary.Briefing{}, err
		}
		row.briefing.PossiblyStale = stale
	}
	if err = tx.Commit(ctx); err != nil {
		return secretary.Briefing{}, err
	}
	return row.briefing, nil
}

func readSecretaryRow(ctx context.Context, tx pgx.Tx, organizationID, userID, day string) (secretaryRow, error) {
	var row secretaryRow
	var items []byte
	err := tx.QueryRow(ctx, `SELECT status,local_day::text,job_id,role,items,remaining_count,requested_at,generated_at
		FROM secretary_briefings WHERE organization_id=$1 AND user_id=$2 AND local_day=$3::date`, organizationID, userID, day).Scan(
		&row.briefing.Status, &row.briefing.LocalDay, &row.briefing.JobID, &row.role, &items,
		&row.briefing.Remaining, &row.requested, &row.briefing.GeneratedAt)
	if err != nil {
		return secretaryRow{}, err
	}
	if err = json.Unmarshal(items, &row.briefing.Items); err != nil {
		return secretaryRow{}, err
	}
	return row, nil
}

func (s *Store) secretarySourcesAuthorized(ctx context.Context, tx pgx.Tx, organizationID, userID string, role tenant.Role, items []secretary.Item) bool {
	for _, item := range items {
		var allowed bool
		query := `SELECT EXISTS(SELECT 1 FROM tasks t WHERE t.organization_id=$1 AND t.id=$3::uuid
			AND ($4 IN ('Owner','Admin') OR t.assignee_user_id=$2 OR EXISTS
			(SELECT 1 FROM task_watchers w WHERE w.organization_id=t.organization_id AND w.task_id=t.id AND w.user_id=$2)))`
		if item.Category == "Routine" {
			query = `SELECT EXISTS(SELECT 1 FROM routine_suggestions s JOIN routine_templates t
				ON t.organization_id=s.organization_id AND t.id=s.template_id
				WHERE s.organization_id=$1 AND s.id=$3::uuid AND s.status='Pending' AND t.active AND
				($4 IN ('Owner','Admin') OR t.responsible_user_id=$2))`
		}
		err := tx.QueryRow(ctx, query, organizationID, userID, item.SourceID, role).Scan(&allowed)
		if err != nil || !allowed {
			return false
		}
	}
	return true
}

func queueSecretary(ctx context.Context, tx pgx.Tx, userID, organizationID, day string, role tenant.Role, expires, now time.Time, oldJobID string) (secretaryRow, error) {
	if oldJobID != "" {
		_, err := tx.Exec(ctx, `UPDATE durable_jobs SET
			status=CASE WHEN status='Queued' THEN 'Cancelled' ELSE 'Failed' END,
			cancelled_at=CASE WHEN status='Queued' THEN $2 ELSE cancelled_at END,
			failed_at=CASE WHEN status='Running' THEN $2 ELSE failed_at END,
			failure_code=CASE WHEN status='Running' THEN 'superseded' ELSE failure_code END,
			current_attempt_id=NULL,lease_token_hash=NULL,lease_expires_at=NULL,worker_id=NULL
			WHERE id=$1 AND status IN ('Queued','Running')`, oldJobID, now.UTC())
		if err != nil {
			return secretaryRow{}, err
		}
	}
	jobID, err := randomUUID()
	if err != nil {
		return secretaryRow{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO durable_jobs
		(id,organization_id,requester_user_id,kind,status,payload,idempotency_key,
		 max_attempts,available_at,created_at)
		VALUES ($1,$2,$3,'secretary','Queued','{}'::jsonb,$5,3,$4,$4)`, jobID, organizationID, userID, now.UTC(), jobID)
	if err != nil {
		return secretaryRow{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO secretary_briefings
		(organization_id,user_id,local_day,job_id,status,role,items,remaining_count,requested_at,expires_at)
		VALUES ($1,$2,$3::date,$4,'Queued',$5,'[]'::jsonb,0,$6,$7)
		ON CONFLICT (organization_id,user_id,local_day) DO UPDATE SET job_id=excluded.job_id,
		status='Queued',role=excluded.role,items='[]'::jsonb,remaining_count=0,
		requested_at=excluded.requested_at,generated_at=NULL,expires_at=excluded.expires_at`,
		organizationID, userID, day, jobID, role, now.UTC(), expires)
	if err != nil {
		return secretaryRow{}, err
	}
	return secretaryRow{briefing: secretary.Briefing{Status: "Queued", LocalDay: day, JobID: jobID, Items: []secretary.Item{}}, role: role, requested: now.UTC()}, nil
}

func countSecretaryCandidates(ctx context.Context, tx pgx.Tx, organizationID, userID, day string, role tenant.Role) (int, error) {
	var tasks, routines int
	err := tx.QueryRow(ctx, `SELECT count(*) FROM tasks t
		LEFT JOIN secretary_preferences p ON p.organization_id=t.organization_id AND p.user_id=$2
		WHERE t.organization_id=$1 AND t.status IN ('Open','InProgress') AND (t.due_on<=$3::date OR t.due_on IS NULL)
		AND NOT (t.secretary_category=ANY(coalesce(p.hidden_categories,'{}'::text[])))
		AND ($4 IN ('Owner','Admin') OR t.assignee_user_id=$2 OR EXISTS
		(SELECT 1 FROM task_watchers w WHERE w.organization_id=t.organization_id AND w.task_id=t.id AND w.user_id=$2))`,
		organizationID, userID, day, role).Scan(&tasks)
	if err != nil {
		return 0, err
	}
	err = tx.QueryRow(ctx, `SELECT count(*) FROM routine_suggestions s JOIN routine_templates t
		ON t.organization_id=s.organization_id AND t.id=s.template_id
		LEFT JOIN secretary_preferences p ON p.organization_id=s.organization_id AND p.user_id=$2
		WHERE s.organization_id=$1 AND s.status='Pending' AND s.due_on<=$3::date AND t.active
		AND NOT ('Routine'=ANY(coalesce(p.hidden_categories,'{}'::text[])))
		AND ($4 IN ('Owner','Admin') OR t.responsible_user_id=$2)`, organizationID, userID, day, role).Scan(&routines)
	return tasks + routines, err
}

func (s *Store) Generate(ctx context.Context, command job.LeaseCommand) error {
	started := time.Now()
	var userID, organizationID string
	var created, claimed time.Time
	var attempts int
	err := s.pool.QueryRow(ctx, `SELECT coalesce(requester_user_id::text,''),organization_id,created_at,started_at,attempt_count
		FROM durable_jobs WHERE id=$1 AND kind='secretary'`, command.JobID).Scan(&userID, &organizationID, &created, &claimed, &attempts)
	if errors.Is(err, pgx.ErrNoRows) || userID == "" {
		return job.ErrLeaseLost
	}
	if err != nil {
		return err
	}
	tx, err := s.organizationTx(ctx, userID, organizationID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var role tenant.Role
	var timezone string
	err = tx.QueryRow(ctx, `SELECT m.role,o.timezone FROM memberships m JOIN organizations o ON o.id=m.organization_id
		WHERE m.organization_id=$1 AND m.user_id=$2 FOR SHARE OF m`, organizationID, userID).Scan(&role, &timezone)
	if errors.Is(err, pgx.ErrNoRows) {
		return job.ErrLeaseLost
	}
	if err != nil {
		return err
	}
	var valid bool
	err = tx.QueryRow(ctx, `SELECT status='Running' AND current_attempt_id=$2 AND lease_token_hash=$3 AND lease_expires_at>$4
		FROM durable_jobs WHERE id=$1 FOR UPDATE`, command.JobID, command.AttemptID, leaseHash(command.LeaseToken), command.Now.UTC()).Scan(&valid)
	if err != nil || !valid {
		return job.ErrLeaseLost
	}
	var day string
	var roleAtRequest tenant.Role
	err = tx.QueryRow(ctx, `SELECT local_day::text,role FROM secretary_briefings WHERE job_id=$1 FOR UPDATE`, command.JobID).Scan(&day, &roleAtRequest)
	if errors.Is(err, pgx.ErrNoRows) {
		return job.ErrLeaseLost
	}
	if err != nil {
		return err
	}
	if role != roleAtRequest {
		return job.ErrLeaseLost
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return err
	}
	if command.Now.In(location).Format("2006-01-02") != day {
		return job.ErrLeaseLost
	}
	if err = syncRoutineSuggestions(ctx, tx, userID, organizationID, role, day, command.Now.UTC()); err != nil {
		return err
	}
	count, err := countSecretaryCandidates(ctx, tx, organizationID, userID, day, role)
	if err != nil {
		return err
	}
	condition := `t.organization_id=$1 AND t.status IN ('Open','InProgress') AND (t.due_on<=$3::date OR t.due_on IS NULL)
		AND NOT (t.secretary_category=ANY(coalesce(p.hidden_categories,'{}'::text[])))
		AND ($4 IN ('Owner','Admin') OR t.assignee_user_id=$2 OR EXISTS
		(SELECT 1 FROM task_watchers w WHERE w.organization_id=t.organization_id AND w.task_id=t.id AND w.user_id=$2))`
	prefJoin := ` LEFT JOIN secretary_preferences p ON p.organization_id=t.organization_id AND p.user_id=$2 `
	rows, err := tx.Query(ctx, `SELECT t.id,t.title,coalesce(t.due_on::text,''),t.priority,t.secretary_category,
		CASE WHEN t.assignee_user_id=$2 OR EXISTS
		(SELECT 1 FROM task_watchers w WHERE w.organization_id=t.organization_id AND w.task_id=t.id AND w.user_id=$2)
		THEN 'personal' ELSE 'team' END,
		t.id=ANY(coalesce(p.pinned_task_ids,'{}'::uuid[]))
		FROM tasks t `+prefJoin+` WHERE `+condition+`
		ORDER BY CASE WHEN t.due_on<$3::date THEN 0 WHEN t.due_on=$3::date THEN 1 ELSE 2 END,
		CASE WHEN t.id=ANY(coalesce(p.pinned_task_ids,'{}'::uuid[])) THEN 0 ELSE 1 END,
		CASE t.priority WHEN 'Urgent' THEN 0 WHEN 'High' THEN 1 ELSE 2 END,
		t.due_on,t.id LIMIT 5`, organizationID, userID, day, role)
	if err != nil {
		return err
	}
	items := make([]secretary.Item, 0, 5)
	for rows.Next() {
		var item secretary.Item
		if err = rows.Scan(&item.SourceID, &item.Title, &item.DueOn, &item.Priority, &item.Category, &item.Scope, &item.Pinned); err != nil {
			rows.Close()
			return err
		}
		item.URL = "/o/" + url.PathEscape(organizationID) + "/tasks/" + url.PathEscape(item.SourceID)
		if item.DueOn == "" {
			item.Reason = "ไม่กำหนดวันครบกำหนด · ความสำคัญ " + item.Priority
		} else if item.DueOn < day {
			item.Reason = "เกินกำหนด · ความสำคัญ " + item.Priority
		} else {
			item.Reason = "ครบกำหนดวันนี้ · ความสำคัญ " + item.Priority
		}
		if item.Pinned {
			item.Reason = "ปักหมุดในกลุ่มวันครบกำหนดเดียวกัน · " + item.Reason
		}
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	routineRows, err := tx.Query(ctx, `SELECT s.id,t.title,s.due_on::text,t.responsible_user_id
		FROM routine_suggestions s JOIN routine_templates t ON t.organization_id=s.organization_id AND t.id=s.template_id
		LEFT JOIN secretary_preferences p ON p.organization_id=s.organization_id AND p.user_id=$2
		WHERE s.organization_id=$1 AND s.status='Pending' AND s.due_on<=$3::date AND t.active
		AND NOT ('Routine'=ANY(coalesce(p.hidden_categories,'{}'::text[])))
		AND ($4 IN ('Owner','Admin') OR t.responsible_user_id=$2)
		ORDER BY s.due_on,s.id LIMIT 5`, organizationID, userID, day, role)
	if err != nil {
		return err
	}
	for routineRows.Next() {
		var item secretary.Item
		var responsible string
		if err = routineRows.Scan(&item.SourceID, &item.Title, &item.DueOn, &responsible); err != nil {
			routineRows.Close()
			return err
		}
		item.Category = "Routine"
		item.Priority = "Normal"
		item.Scope = "team"
		if responsible == userID {
			item.Scope = "personal"
		}
		item.Reason = "งานประจำถึงรอบ · รอยืนยันก่อนสร้าง Task"
		item.URL = "/o/" + url.PathEscape(organizationID) + "/secretary#routine-" + url.PathEscape(item.SourceID)
		items = append(items, item)
	}
	err = routineRows.Err()
	routineRows.Close()
	if err != nil {
		return err
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		group := func(due string) int {
			if due == "" {
				return 2
			}
			if due < day {
				return 0
			}
			return 1
		}
		if group(a.DueOn) != group(b.DueOn) {
			return group(a.DueOn) < group(b.DueOn)
		}
		if a.Pinned != b.Pinned {
			return a.Pinned
		}
		priority := map[string]int{"Urgent": 0, "High": 1, "Normal": 2}
		if priority[a.Priority] != priority[b.Priority] {
			return priority[a.Priority] < priority[b.Priority]
		}
		if a.DueOn != b.DueOn {
			return a.DueOn < b.DueOn
		}
		return a.SourceID < b.SourceID
	})
	if len(items) > 5 {
		items = items[:5]
	}
	// The membership lock prevents role removal until commit; source links reauthorize on open.
	encoded, err := json.Marshal(items)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE secretary_briefings SET status='Ready',items=$2,remaining_count=$3,generated_at=$4
		WHERE job_id=$1`, command.JobID, encoded, max(0, count-len(items)), command.Now.UTC())
	if err != nil {
		return err
	}
	result, _ := json.Marshal(map[string]any{"candidate_count": count, "attempt_id": command.AttemptID})
	_, err = tx.Exec(ctx, `UPDATE durable_jobs SET status='Completed',result=$2,completed_at=$3,
		current_attempt_id=NULL,lease_token_hash=NULL,lease_expires_at=NULL,worker_id=NULL WHERE id=$1`, command.JobID, result, command.Now.UTC())
	if err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	s.logger.Info("secretary_briefing_generated", "job_id", command.JobID,
		"candidate_count", count, "queue_ms", claimed.Sub(created).Milliseconds(),
		"gather_ms", time.Since(started).Milliseconds(), "attempts", attempts,
		"provider_tokens", 0, "provider_cost", 0)
	return nil
}

func (s *Store) ReadySecretary(ctx context.Context) error {
	var present bool
	err := s.pool.QueryRow(ctx, `SELECT to_regclass('public.secretary_briefings') IS NOT NULL`).Scan(&present)
	if err != nil {
		return err
	}
	if !present {
		return fmt.Errorf("secretary schema missing")
	}
	return nil
}
