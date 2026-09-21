package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"plaiflow/api/internal/job"
	"plaiflow/api/internal/work"
)

func (s *Store) ClaimJobs(ctx context.Context, command job.ClaimCommand) ([]job.Claimed, error) {
	if command.WorkerID == "" || command.Limit < 1 || command.Limit > 10 || command.Lease <= 0 || len(command.Kinds) == 0 {
		return nil, errors.New("invalid job claim")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	now := command.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	_, err = tx.Exec(ctx, `WITH exhausted AS (
		UPDATE durable_jobs SET status='Failed',failed_at=$1,failure_code='attempts_exhausted',
			current_attempt_id=NULL,lease_token_hash=NULL,lease_expires_at=NULL,worker_id=NULL
		WHERE status='Running' AND lease_expires_at<=$1 AND attempt_count>=max_attempts
		RETURNING id,organization_id,legacy_export_job_id
	), legacy AS (
		UPDATE export_jobs e SET status='Failed',failed_at=$1,failure_code='attempts_exhausted'
		FROM exhausted x WHERE e.id=x.legacy_export_job_id RETURNING e.id
	)
	INSERT INTO audit_events (organization_id,event_type,target_type,target_id,outcome,reason_code,occurred_at)
	SELECT organization_id,'job.failed','durable_job',id,'failed','attempts_exhausted',$1 FROM exhausted`, now)
	if err != nil {
		return nil, err
	}
	kinds := make([]string, len(command.Kinds))
	for index, kind := range command.Kinds {
		kinds[index] = string(kind)
	}
	rows, err := tx.Query(ctx, `SELECT id,organization_id,coalesce(requester_user_id::text,''),kind,status,payload,
        attempt_count,created_at FROM durable_jobs
        WHERE kind=ANY($1) AND attempt_count<max_attempts AND
          ((status='Queued' AND available_at<=$2) OR (status='Running' AND lease_expires_at<=$2))
        ORDER BY available_at,created_at,id FOR UPDATE SKIP LOCKED LIMIT $3`, kinds, now, command.Limit)
	if err != nil {
		return nil, err
	}
	type candidate struct {
		job      job.Job
		wasStale bool
	}
	candidates, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (candidate, error) {
		var item candidate
		var payload []byte
		var previous job.Status
		err := row.Scan(&item.job.ID, &item.job.OrganizationID, &item.job.RequesterUserID, &item.job.Kind, &previous,
			&payload, &item.job.AttemptCount, &item.job.CreatedAt)
		item.job.Payload = json.RawMessage(payload)
		item.wasStale = previous == job.Running
		return item, err
	})
	if err != nil {
		return nil, err
	}
	claimed := make([]job.Claimed, 0, len(candidates))
	for _, candidate := range candidates {
		attemptID, err := randomUUID()
		if err != nil {
			return nil, err
		}
		leaseToken, leaseHash, err := randomLeaseToken()
		if err != nil {
			return nil, err
		}
		expires := now.Add(command.Lease)
		commandTag, err := tx.Exec(ctx, `UPDATE durable_jobs SET status='Running',attempt_count=attempt_count+1,
            current_attempt_id=$2,lease_token_hash=$3,lease_expires_at=$4,worker_id=$5,
            started_at=coalesce(started_at,$1),stale_reclaim_count=stale_reclaim_count+$6
            WHERE id=$7 AND organization_id=$8`, now, attemptID, leaseHash, expires, command.WorkerID, boolInt(candidate.wasStale), candidate.job.ID, candidate.job.OrganizationID)
		if err != nil || commandTag.RowsAffected() != 1 {
			return nil, errors.New("job claim update failed")
		}
		candidate.job.Status = job.Running
		candidate.job.AttemptID = attemptID
		candidate.job.AttemptCount++
		claimed = append(claimed, job.Claimed{Job: candidate.job, LeaseToken: leaseToken, LeaseExpiresAt: expires})
		eventType := "job.claimed"
		if candidate.wasStale {
			eventType = "job.reclaimed"
		}
		metadata, _ := json.Marshal(map[string]any{"worker_id": command.WorkerID, "attempt_id": attemptID, "attempt_count": candidate.job.AttemptCount})
		if _, err := tx.Exec(ctx, `INSERT INTO audit_events
            (organization_id,event_type,target_type,target_id,outcome,metadata,occurred_at)
            VALUES ($1,$2,'durable_job',$3,'succeeded',$4,$5)`, candidate.job.OrganizationID, eventType, candidate.job.ID, metadata, now); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO durable_job_worker_heartbeats (worker_id,heartbeat_at,environment,supported_kinds)
		VALUES ($1,$2,$3,$4) ON CONFLICT (worker_id) DO UPDATE SET heartbeat_at=excluded.heartbeat_at,environment=excluded.environment,supported_kinds=excluded.supported_kinds`, command.WorkerID, now, command.Environment, kinds); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return claimed, nil
}

func (s *Store) HeartbeatJob(ctx context.Context, command job.LeaseCommand, lease time.Duration) (time.Time, error) {
	now := command.Now.UTC()
	expires := now.Add(lease)
	tag, err := s.pool.Exec(ctx, `UPDATE durable_jobs SET lease_expires_at=$1 WHERE id=$2 AND status='Running'
		AND current_attempt_id=$3 AND lease_token_hash=$4 AND lease_expires_at>$5`, expires, command.JobID, command.AttemptID, leaseHash(command.LeaseToken), now)
	if err != nil {
		return time.Time{}, err
	}
	if tag.RowsAffected() != 1 {
		return time.Time{}, job.ErrLeaseLost
	}
	return expires, nil
}

func (s *Store) FailJob(ctx context.Context, command job.FailureCommand) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var organizationID string
	var attemptCount, maxAttempts int
	err = tx.QueryRow(ctx, `SELECT organization_id,attempt_count,max_attempts FROM durable_jobs WHERE id=$1 AND status='Running'
		AND current_attempt_id=$2 AND lease_token_hash=$3 AND lease_expires_at>$4 FOR UPDATE`, command.JobID, command.AttemptID, leaseHash(command.LeaseToken), command.Now.UTC()).Scan(&organizationID, &attemptCount, &maxAttempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return job.ErrLeaseLost
	}
	if err != nil {
		return err
	}
	transient := command.Code == "temporary_upstream" || command.Code == "rate_limited" || command.Code == "artifact_upload_failed"
	status := job.Failed
	available := command.Now.UTC()
	if transient && attemptCount < maxAttempts {
		status = job.Queued
		backoff := time.Duration(1<<min(attemptCount, 8))*time.Second + retryJitter()
		if command.RetryAfter > backoff {
			backoff = command.RetryAfter
		}
		available = available.Add(backoff)
	}
	_, err = tx.Exec(ctx, `UPDATE durable_jobs SET status=$1,available_at=$2,failure_code=$3,
		failed_at=CASE WHEN $1='Failed' THEN $4 ELSE NULL END,current_attempt_id=NULL,lease_token_hash=NULL,lease_expires_at=NULL,worker_id=NULL
		WHERE id=$5`, status, available, command.Code, command.Now.UTC(), command.JobID)
	if err != nil {
		return err
	}
	if status == job.Failed {
		_, err = tx.Exec(ctx, `UPDATE export_jobs SET status='Failed',failure_code=$1,failed_at=$2 WHERE id=(SELECT legacy_export_job_id FROM durable_jobs WHERE id=$3)`, command.Code, command.Now.UTC(), command.JobID)
		if err != nil {
			return err
		}
	}
	metadata, _ := json.Marshal(map[string]any{"attempt_id": command.AttemptID, "retry": status == job.Queued})
	_, err = tx.Exec(ctx, `INSERT INTO audit_events (organization_id,event_type,target_type,target_id,outcome,reason_code,metadata,occurred_at)
		VALUES ($1,$2,'durable_job',$3,$4,$5,$6,$7)`, organizationID, map[bool]string{true: "job.retry_scheduled", false: "job.failed"}[status == job.Queued], command.JobID, strings.ToLower(string(status)), command.Code, metadata, command.Now.UTC())
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type exportPayload struct {
	Format   string          `json:"format"`
	Filters  work.TaskFilter `json:"filters"`
	RowCount int64           `json:"row_count"`
	DataAsOf time.Time       `json:"data_as_of"`
}

func (s *Store) ReadExportPage(ctx context.Context, command job.LeaseCommand, cursor string, limit int) (job.ExportPage, error) {
	if limit < 1 || limit > 500 {
		return job.ExportPage{}, errors.New("invalid export page limit")
	}
	var organizationID string
	var payloadBytes []byte
	err := s.pool.QueryRow(ctx, `SELECT organization_id,payload FROM durable_jobs WHERE id=$1 AND kind='export' AND status='Running'
		AND current_attempt_id=$2 AND lease_token_hash=$3 AND lease_expires_at>$4`, command.JobID, command.AttemptID, leaseHash(command.LeaseToken), command.Now.UTC()).Scan(&organizationID, &payloadBytes)
	if errors.Is(err, pgx.ErrNoRows) {
		return job.ExportPage{}, job.ErrLeaseLost
	}
	if err != nil {
		return job.ExportPage{}, err
	}
	var payload exportPayload
	if json.Unmarshal(payloadBytes, &payload) != nil || payload.Format != "csv" && payload.Format != "xlsx" || payload.DataAsOf.IsZero() {
		return job.ExportPage{}, errors.New("invalid export payload")
	}
	args := exportArgs(organizationID, payload.Filters)
	args = append(args, payload.DataAsOf, nilIfEmpty(cursor), limit+1)
	query := `SELECT t.organization_id,o.name,t.id,t.title,t.description,t.status,t.priority,coalesce(t.due_on::text,''),
		coalesce(t.status IN ('Open','InProgress') AND t.due_on<(($12 AT TIME ZONE o.timezone)::date),false),coalesce(c.display_name,''),coalesce(a.display_name,''),
		coalesce(string_agg(wu.display_name,'|' ORDER BY lower(wu.display_name),wu.id) FILTER (WHERE wu.id IS NOT NULL),''),
		t.created_at,t.updated_at,t.status_changed_at,t.completed_at
		FROM tasks t JOIN organizations o ON o.id=t.organization_id JOIN users c ON c.id=t.creator_user_id
		LEFT JOIN users a ON a.id=t.assignee_user_id LEFT JOIN task_watchers tw ON tw.organization_id=t.organization_id AND tw.task_id=t.id
		LEFT JOIN users wu ON wu.id=tw.user_id` + exportWhere[len(` FROM tasks t JOIN organizations o ON o.id=t.organization_id`):] + `
		AND t.created_at<=$12 AND t.updated_at<=$12 AND ($13::uuid IS NULL OR t.id>$13::uuid)
		GROUP BY t.id,o.name,o.timezone,c.display_name,a.display_name ORDER BY t.id LIMIT $14`
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return job.ExportPage{}, err
	}
	defer rows.Close()
	items := make([]work.ExportRow, 0, limit+1)
	for rows.Next() {
		var row work.ExportRow
		if err := rows.Scan(&row.OrganizationID, &row.OrganizationName, &row.TaskID, &row.Title, &row.Description, &row.Status, &row.Priority, &row.DueDate, &row.IsOverdue, &row.CreatorName, &row.AssigneeName, &row.WatcherNames, &row.CreatedAt, &row.UpdatedAt, &row.StatusChangedAt, &row.CompletedAt); err != nil {
			return job.ExportPage{}, err
		}
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return job.ExportPage{}, err
	}
	page := job.ExportPage{Rows: items, Done: len(items) <= limit}
	if len(items) > limit {
		page.Rows = items[:limit]
		page.NextCursor = page.Rows[len(page.Rows)-1].TaskID
	}
	return page, nil
}

func (s *Store) CompleteExport(ctx context.Context, command job.CompleteExportCommand) (job.ExportArtifact, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return job.ExportArtifact{}, err
	}
	defer tx.Rollback(ctx)
	var organizationID string
	var payloadBytes, resultBytes []byte
	var status job.Status
	err = tx.QueryRow(ctx, `SELECT organization_id,payload,status,coalesce(result,'{}'::jsonb) FROM durable_jobs WHERE id=$1 FOR UPDATE`, command.JobID).Scan(&organizationID, &payloadBytes, &status, &resultBytes)
	if errors.Is(err, pgx.ErrNoRows) {
		return job.ExportArtifact{}, job.ErrLeaseLost
	}
	if err != nil {
		return job.ExportArtifact{}, err
	}
	if status == job.Completed {
		var result struct {
			AttemptID string `json:"attempt_id"`
		}
		_ = json.Unmarshal(resultBytes, &result)
		if result.AttemptID != command.AttemptID {
			return job.ExportArtifact{}, job.ErrLeaseLost
		}
		return artifactByJob(ctx, tx, command.JobID)
	}
	var payload exportPayload
	maxBytes := int64(256 << 20)
	if command.Format == "xlsx" {
		maxBytes = 128 << 20
	}
	if json.Unmarshal(payloadBytes, &payload) != nil || payload.Format != command.Format || command.RowCount > payload.RowCount || command.ByteCount < 1 || command.ByteCount > maxBytes || command.ExpiresAt.Sub(command.Now.UTC()) > 24*time.Hour+time.Minute {
		return job.ExportArtifact{}, errors.New("artifact metadata mismatch")
	}
	var valid bool
	err = tx.QueryRow(ctx, `SELECT status='Running' AND current_attempt_id=$2 AND lease_token_hash=$3 AND lease_expires_at>$4 FROM durable_jobs WHERE id=$1`, command.JobID, command.AttemptID, leaseHash(command.LeaseToken), command.Now.UTC()).Scan(&valid)
	if err != nil || !valid {
		return job.ExportArtifact{}, job.ErrLeaseLost
	}
	hash, err := hex.DecodeString(command.SHA256)
	if err != nil || len(hash) != 32 {
		return job.ExportArtifact{}, errors.New("invalid artifact hash")
	}
	_, err = tx.Exec(ctx, `INSERT INTO durable_job_artifacts (job_id,organization_id,object_key,format,row_count,byte_count,sha256,created_at,expires_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, command.JobID, organizationID, command.ObjectKey, command.Format, command.RowCount, command.ByteCount, hash, command.Now.UTC(), command.ExpiresAt)
	if err != nil {
		return job.ExportArtifact{}, err
	}
	result, _ := json.Marshal(map[string]any{"attempt_id": command.AttemptID, "format": command.Format, "row_count": command.RowCount, "byte_count": command.ByteCount})
	_, err = tx.Exec(ctx, `UPDATE durable_jobs SET status='Completed',result=$1,completed_at=$2,current_attempt_id=NULL,lease_token_hash=NULL,lease_expires_at=NULL,worker_id=NULL WHERE id=$3`, result, command.Now.UTC(), command.JobID)
	if err != nil {
		return job.ExportArtifact{}, err
	}
	_, err = tx.Exec(ctx, `UPDATE export_jobs SET status='Completed',produced_byte_count=$1,completed_at=$2,object_expires_at=$3 WHERE id=(SELECT legacy_export_job_id FROM durable_jobs WHERE id=$4)`, command.ByteCount, command.Now.UTC(), command.ExpiresAt, command.JobID)
	if err != nil {
		return job.ExportArtifact{}, err
	}
	metadata, _ := json.Marshal(map[string]any{"attempt_id": command.AttemptID, "format": command.Format, "row_count": command.RowCount, "byte_count": command.ByteCount})
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events (organization_id,event_type,target_type,target_id,outcome,metadata,occurred_at)
		VALUES ($1,'job.completed','durable_job',$2,'completed',$3,$4)`, organizationID, command.JobID, metadata, command.Now.UTC()); err != nil {
		return job.ExportArtifact{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return job.ExportArtifact{}, err
	}
	return job.ExportArtifact{JobID: command.JobID, OrganizationID: organizationID, ObjectKey: command.ObjectKey, Format: command.Format, SHA256: command.SHA256, RowCount: command.RowCount, ByteCount: command.ByteCount, ExpiresAt: command.ExpiresAt}, nil
}

func (s *Store) AuthorizeExportArtifact(ctx context.Context, userID, organizationID, jobID string) (job.ExportArtifact, error) {
	var artifact job.ExportArtifact
	var hash []byte
	err := s.pool.QueryRow(ctx, `SELECT a.job_id,a.organization_id,a.object_key,a.format,a.row_count,a.byte_count,a.sha256,a.expires_at FROM durable_job_artifacts a JOIN durable_jobs j ON j.id=a.job_id JOIN memberships m ON m.organization_id=j.organization_id AND m.user_id=$1 WHERE a.job_id=$2 AND a.organization_id=$3 AND a.deleted_at IS NULL AND a.expires_at>now() AND (j.requester_user_id=$1 OR m.role IN ('Owner','Admin'))`, userID, jobID, organizationID).Scan(&artifact.JobID, &artifact.OrganizationID, &artifact.ObjectKey, &artifact.Format, &artifact.RowCount, &artifact.ByteCount, &hash, &artifact.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return job.ExportArtifact{}, work.ErrNotFound
	}
	if err != nil {
		return job.ExportArtifact{}, err
	}
	artifact.SHA256 = hex.EncodeToString(hash)
	return artifact, nil
}

func (s *Store) RecordArtifactDownload(ctx context.Context, userID, organizationID, jobID string, now time.Time) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO audit_events (organization_id,actor_user_id,event_type,target_type,target_id,outcome,occurred_at)
		VALUES ($1,$2,'export.downloaded','durable_job',$3,'succeeded',$4)`, organizationID, userID, jobID, now)
	return err
}

func artifactByJob(ctx context.Context, tx pgx.Tx, jobID string) (job.ExportArtifact, error) {
	var a job.ExportArtifact
	var hash []byte
	err := tx.QueryRow(ctx, `SELECT job_id,organization_id,object_key,format,row_count,byte_count,sha256,expires_at FROM durable_job_artifacts WHERE job_id=$1`, jobID).Scan(&a.JobID, &a.OrganizationID, &a.ObjectKey, &a.Format, &a.RowCount, &a.ByteCount, &hash, &a.ExpiresAt)
	a.SHA256 = hex.EncodeToString(hash)
	return a, err
}
func leaseHash(token string) []byte { sum := sha256.Sum256([]byte(token)); return sum[:] }

func retryJitter() time.Duration {
	var value [2]byte
	if _, err := rand.Read(value[:]); err != nil {
		return 0
	}
	return time.Duration((int(value[0])<<8|int(value[1]))%1000) * time.Millisecond
}

func randomUUID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	value[6] = value[6]&0x0f | 0x40
	value[8] = value[8]&0x3f | 0x80
	encoded := hex.EncodeToString(value[:])
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32], nil
}

func randomLeaseToken() (string, []byte, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(value)
	hash := sha256.Sum256([]byte(token))
	return token, hash[:], nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
