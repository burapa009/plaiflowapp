package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"plaiflow/api/internal/work"
)

func (s *Store) ScheduleReminders(ctx context.Context, now time.Time) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	command, err := tx.Exec(ctx, `WITH recipients AS (
        SELECT t.organization_id,t.id task_id,t.due_on,t.assignee_user_id user_id,o.timezone
        FROM tasks t JOIN organizations o ON o.id=t.organization_id
        WHERE t.status IN ('Open','InProgress') AND t.due_on IS NOT NULL AND t.assignee_user_id IS NOT NULL
        UNION
        SELECT t.organization_id,t.id,t.due_on,tw.user_id,o.timezone
        FROM tasks t JOIN organizations o ON o.id=t.organization_id
        JOIN task_watchers tw ON tw.organization_id=t.organization_id AND tw.task_id=t.id
        WHERE t.status IN ('Open','InProgress') AND t.due_on IS NOT NULL
    ), candidates AS (
        SELECT r.*,m.milestone,(r.due_on+m.day_offset+time '09:00') AT TIME ZONE r.timezone scheduled_for
        FROM recipients r CROSS JOIN (VALUES ('day_before',-1),('due_day',0),('day_after',1)) m(milestone,day_offset)
    )
    INSERT INTO reminders (id,organization_id,task_id,recipient_user_id,due_on,milestone,scheduled_for,created_at)
    SELECT md5(organization_id::text||task_id::text||user_id::text||due_on::text||milestone)::uuid,
           organization_id,task_id,user_id,due_on,milestone,scheduled_for,$1
    FROM candidates WHERE scheduled_for<=$1
    ON CONFLICT (organization_id,task_id,recipient_user_id,due_on,milestone) DO NOTHING`, now)
	if err != nil {
		return 0, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO notifications
        (id,organization_id,recipient_user_id,category,logical_key,title,deep_link,delivery_mode,created_at)
        SELECT md5('notification:'||r.id::text)::uuid,r.organization_id,r.recipient_user_id,'DueReminder',
               'reminder:'||r.id::text,'ถึงกำหนดงาน','/o/'||r.organization_id::text||'/tasks/'||r.task_id::text,
               coalesce(p.web_mode,'Immediate'),$1
        FROM reminders r LEFT JOIN notification_preferences p
          ON p.organization_id=r.organization_id AND p.user_id=r.recipient_user_id AND p.category='DueReminder'
        WHERE r.scheduled_for<=$1 AND (coalesce(p.web_mode,'Immediate')<>'Off' OR coalesce(p.line_mode,'Off')<>'Off')
        ON CONFLICT (organization_id,recipient_user_id,logical_key) DO NOTHING`, now)
	if err != nil {
		return 0, err
	}
	if err := queueImmediateLINE(ctx, tx, "reminder:%", now); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return int(command.RowsAffected()), nil
}

func (s *Store) ProcessDomainEvents(ctx context.Context, limit int, lease time.Duration) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `WITH eligible AS (
        SELECT id FROM domain_events WHERE (processing_state='Pending' AND available_at<=now())
          OR (processing_state='Processing' AND lease_expires_at<=now())
        ORDER BY available_at,id FOR UPDATE SKIP LOCKED LIMIT $1
    ) UPDATE domain_events e SET processing_state='Processing',lease_expires_at=now()+$2*interval '1 second'
      FROM eligible WHERE e.id=eligible.id RETURNING e.id`, limit, int(lease.Seconds()))
	if err != nil {
		return 0, err
	}
	ids, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (int64, error) {
		var id int64
		return id, row.Scan(&id)
	})
	if err != nil || len(ids) == 0 {
		if err == nil {
			err = tx.Commit(ctx)
		}
		return 0, err
	}
	_, err = tx.Exec(ctx, `WITH recipients AS (
        SELECT e.id event_id,e.organization_id,e.aggregate_id task_id,e.actor_user_id,
               CASE WHEN e.event_type IN ('task.created','task.assigned') THEN nullif(e.payload->>'assignee_user_id','')::uuid
                    WHEN e.event_type='task.watcher_added' THEN nullif(e.payload->>'watcher_user_id','')::uuid END recipient_user_id,
               CASE WHEN e.event_type IN ('task.created','task.assigned') THEN 'Assignment' ELSE 'TaskChange' END category
        FROM domain_events e WHERE e.id=ANY($1) AND e.event_type IN ('task.created','task.assigned','task.watcher_added')
        UNION
		SELECT e.id,e.organization_id,e.aggregate_id,e.actor_user_id,
		       nullif(e.payload->>'previous_assignee_user_id','')::uuid,'TaskChange'
		FROM domain_events e WHERE e.id=ANY($1) AND e.event_type IN ('task.assigned','task.unassigned')
		UNION
		SELECT e.id,e.organization_id,e.aggregate_id,e.actor_user_id,tw.user_id,'TaskChange'
		FROM domain_events e JOIN task_watchers tw ON tw.organization_id=e.organization_id AND tw.task_id=e.aggregate_id
		WHERE e.id=ANY($1) AND e.event_type IN ('task.assigned','task.unassigned')
		UNION
        SELECT e.id,e.organization_id,e.aggregate_id,e.actor_user_id,r.user_id,'TaskChange'
        FROM domain_events e JOIN LATERAL (
            SELECT t.assignee_user_id user_id FROM tasks t WHERE t.organization_id=e.organization_id AND t.id=e.aggregate_id
            UNION SELECT tw.user_id FROM task_watchers tw WHERE tw.organization_id=e.organization_id AND tw.task_id=e.aggregate_id
        ) r ON true
		WHERE e.id=ANY($1) AND e.event_type IN ('task.details_changed','task.due_changed','task.priority_changed','task.status_changed')
    )
    INSERT INTO notifications (id,organization_id,recipient_user_id,category,logical_key,title,deep_link,delivery_mode,created_at)
    SELECT md5('notification:'||event_id::text||':'||recipient_user_id::text)::uuid,organization_id,recipient_user_id,category,
           'event:'||event_id::text||':'||recipient_user_id::text,
           CASE WHEN category='Assignment' THEN 'คุณได้รับมอบหมายงาน' ELSE 'งานที่คุณติดตามมีการเปลี่ยนแปลง' END,
           '/o/'||organization_id::text||'/tasks/'||task_id::text,
		   coalesce(p.web_mode,CASE WHEN category='Assignment' THEN 'Immediate' ELSE 'Digest' END),now()
	FROM recipients r LEFT JOIN notification_preferences p
	  ON p.organization_id=r.organization_id AND p.user_id=r.recipient_user_id AND p.category=r.category
    WHERE recipient_user_id IS NOT NULL AND recipient_user_id<>actor_user_id
      AND (coalesce(p.web_mode,CASE WHEN category='Assignment' THEN 'Immediate' ELSE 'Digest' END)<>'Off' OR coalesce(p.line_mode,'Off')<>'Off')
      AND EXISTS (SELECT 1 FROM memberships m WHERE m.organization_id=r.organization_id AND m.user_id=r.recipient_user_id)
    ON CONFLICT (organization_id,recipient_user_id,logical_key) DO NOTHING`, ids)
	if err != nil {
		return 0, err
	}
	if err := queueImmediateLINE(ctx, tx, "event:%", time.Now().UTC()); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `UPDATE domain_events SET processing_state='Processed',processed_at=now(),lease_expires_at=NULL
        WHERE id=ANY($1)`, ids); err != nil {
		return 0, err
	}
	return len(ids), tx.Commit(ctx)
}

func queueImmediateLINE(ctx context.Context, tx pgx.Tx, pattern string, now time.Time) error {
	_, err := tx.Exec(ctx, `INSERT INTO notification_deliveries
        (id,organization_id,notification_id,channel,logical_key,status,available_at)
        SELECT md5('line:'||n.logical_key)::uuid,n.organization_id,n.id,'LINE','line:'||n.logical_key,'Pending',$2
        FROM notifications n JOIN notification_preferences p
          ON p.organization_id=n.organization_id AND p.user_id=n.recipient_user_id AND p.category=n.category
        WHERE n.logical_key LIKE $1 AND p.line_mode='Immediate' AND n.created_at>=p.updated_at
          AND EXISTS (SELECT 1 FROM auth_identities a WHERE a.user_id=n.recipient_user_id AND a.provider='line' AND a.disabled_at IS NULL)
        ON CONFLICT (organization_id,channel,logical_key) DO NOTHING`, pattern, now)
	return err
}

func (s *Store) ProcessDigests(ctx context.Context, now time.Time) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	command, err := tx.Exec(ctx, `WITH candidates AS (
        SELECT DISTINCT ON (n.organization_id,n.recipient_user_id,(n.created_at AT TIME ZONE o.timezone)::date)
               n.id,n.organization_id,n.recipient_user_id,(n.created_at AT TIME ZONE o.timezone)::date digest_date
        FROM notifications n JOIN organizations o ON o.id=n.organization_id
        JOIN notification_preferences p ON p.organization_id=n.organization_id AND p.user_id=n.recipient_user_id AND p.category=n.category
        WHERE p.line_mode='Digest' AND n.created_at>=p.updated_at AND n.digested_at IS NULL
          AND ($1 AT TIME ZONE o.timezone)::time>=time '09:00'
          AND EXISTS (SELECT 1 FROM auth_identities a WHERE a.user_id=n.recipient_user_id AND a.provider='line' AND a.disabled_at IS NULL)
        ORDER BY n.organization_id,n.recipient_user_id,(n.created_at AT TIME ZONE o.timezone)::date,n.created_at,n.id
    ) INSERT INTO notification_deliveries
        (id,organization_id,notification_id,channel,logical_key,status,available_at)
        SELECT md5('line-digest:'||organization_id::text||':'||recipient_user_id::text||':'||digest_date::text)::uuid,
               organization_id,id,'LINE','digest:'||recipient_user_id::text||':'||digest_date::text,'Pending',$1 FROM candidates
        ON CONFLICT (organization_id,channel,logical_key) DO NOTHING`, now)
	if err != nil {
		return 0, err
	}
	_, err = tx.Exec(ctx, `UPDATE notifications n SET digested_at=$1 FROM organizations o,notification_preferences p
        WHERE o.id=n.organization_id AND p.organization_id=n.organization_id AND p.user_id=n.recipient_user_id AND p.category=n.category
          AND n.digested_at IS NULL AND n.created_at>=p.updated_at AND (p.web_mode='Digest' OR p.line_mode='Digest')
          AND ($1 AT TIME ZONE o.timezone)::time>=time '09:00'`, now)
	if err != nil {
		return 0, err
	}
	return int(command.RowsAffected()), tx.Commit(ctx)
}

func (s *Store) ClaimLINEDeliveries(ctx context.Context, limit int, lease time.Duration) ([]work.LINEDelivery, error) {
	rows, err := s.pool.Query(ctx, `WITH eligible AS (
        SELECT id FROM notification_deliveries WHERE (status IN ('Pending','Retryable') AND available_at<=now())
          OR (status='Processing' AND lease_expires_at<=now())
        ORDER BY available_at,id FOR UPDATE SKIP LOCKED LIMIT $1
    ), claimed AS (
        UPDATE notification_deliveries d SET status='Processing',lease_expires_at=now()+$2*interval '1 second'
        FROM eligible WHERE d.id=eligible.id RETURNING d.*
    ) SELECT c.id,c.organization_id,a.subject,o.name,n.deep_link,c.attempt_count
      FROM claimed c JOIN notifications n ON n.organization_id=c.organization_id AND n.id=c.notification_id
      JOIN organizations o ON o.id=c.organization_id JOIN auth_identities a ON a.user_id=n.recipient_user_id
      WHERE a.provider='line' AND a.disabled_at IS NULL`, limit, int(lease.Seconds()))
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (work.LINEDelivery, error) {
		var delivery work.LINEDelivery
		err := row.Scan(&delivery.ID, &delivery.OrganizationID, &delivery.To, &delivery.OrganizationName, &delivery.DeepLink, &delivery.AttemptCount)
		return delivery, err
	})
}

func (s *Store) CompleteLINEDelivery(ctx context.Context, id string, now time.Time) error {
	_, err := s.pool.Exec(ctx, `UPDATE notification_deliveries SET status='Delivered',delivered_at=$2,lease_expires_at=NULL WHERE id=$1`, id, now)
	return err
}

func (s *Store) FailLINEDelivery(ctx context.Context, delivery work.LINEDelivery, code string, delay time.Duration, retry bool) error {
	status := "Failed"
	if retry {
		status = "Retryable"
	}
	_, err := s.pool.Exec(ctx, `UPDATE notification_deliveries SET status=$2,attempt_count=attempt_count+1,
        available_at=now()+$3*interval '1 second',lease_expires_at=NULL,failure_code=$4 WHERE id=$1`,
		delivery.ID, status, int(delay.Seconds()), code)
	return err
}

func (s *Store) WorkMetrics(ctx context.Context) (map[string]int64, error) {
	var pendingEvents, pendingLINE, failedLINE, queuedExports int64
	err := s.pool.QueryRow(ctx, `SELECT
        (SELECT count(*) FROM domain_events WHERE processing_state IN ('Pending','Processing')),
        (SELECT count(*) FROM notification_deliveries WHERE status IN ('Pending','Retryable','Processing')),
        (SELECT count(*) FROM notification_deliveries WHERE status='Failed'),
        (SELECT count(*) FROM export_jobs WHERE status IN ('Queued','Running'))`).Scan(&pendingEvents, &pendingLINE, &failedLINE, &queuedExports)
	if err != nil {
		return nil, fmt.Errorf("work metrics: %w", err)
	}
	return map[string]int64{"pending_events": pendingEvents, "pending_line": pendingLINE, "failed_line": failedLINE, "queued_exports": queuedExports}, nil
}
