package postgres

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"plaiflow/api/internal/inbound"
)

type Store struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func New(ctx context.Context, databaseURL string, maxConnections int32, logger *slog.Logger) (*Store, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, errors.New("invalid database configuration")
	}
	config.MaxConns = maxConnections
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, errors.New("database connection failed")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Store{pool: pool, logger: logger}, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) Ready(ctx context.Context) error {
	started := time.Now()
	defer s.slow(started, "ready")
	var version int
	var dirty bool
	if err := s.pool.QueryRow(ctx, "SELECT version, dirty FROM schema_migrations LIMIT 1").Scan(&version, &dirty); err != nil || dirty || version < 1 {
		return errors.New("database migration is not ready")
	}
	return nil
}

func (s *Store) InsertEvents(ctx context.Context, events []inbound.Event) error {
	started := time.Now()
	defer s.slow(started, "insert_events")
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, event := range events {
		_, err = tx.Exec(ctx, `INSERT INTO inbound_events
            (provider, channel, provider_event_id, event_type, payload, occurred_at,source_type,source_group_id,source_user_id,link_code_hash)
			SELECT $1,$2,$3,$4,$5,$6,$7,nullif($8,''),nullif($9,''),$10
			WHERE $7<>'group' OR $10 IS NOT NULL OR EXISTS (
				SELECT 1 FROM line_group_connections WHERE messaging_channel=$2 AND group_id=$8 AND status='connected'
			)
			ON CONFLICT (provider, channel, provider_event_id) DO NOTHING`,
			event.Provider, event.Channel, event.ProviderEventID, event.Type, event.Payload, event.OccurredAt,
			event.SourceType, event.SourceGroupID, event.SourceUserID, emptyBytesToNil(event.LinkCodeHash))
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) Snapshot(ctx context.Context) (inbound.Snapshot, error) {
	started := time.Now()
	defer s.slow(started, "dashboard_snapshot")
	var result inbound.Snapshot
	var workerHealthy bool
	err := s.pool.QueryRow(ctx, `SELECT
        count(*) FILTER (WHERE status = 'Received'),
        count(*) FILTER (WHERE status = 'Processed'),
        count(*) FILTER (WHERE status = 'Ignored'),
        count(*) FILTER (WHERE status = 'Retryable'),
        count(*) FILTER (WHERE status = 'Failed'),
        EXISTS (SELECT 1 FROM worker_heartbeats WHERE heartbeat_at > now() - interval '2 minutes')
        FROM inbound_events WHERE received_at >= now() - interval '24 hours'`).Scan(
		&result.Counts.Received, &result.Counts.Processed, &result.Counts.Ignored,
		&result.Counts.Retryable, &result.Counts.Failed, &workerHealthy)
	if err != nil {
		return inbound.Snapshot{}, err
	}
	result.API, result.Database = "ok", "ok"
	if workerHealthy {
		result.Worker = "ok"
	} else {
		result.Worker = "unhealthy"
	}
	return result, nil
}

func (s *Store) Claim(ctx context.Context, limit int, lease time.Duration) ([]inbound.Event, error) {
	started := time.Now()
	defer s.slow(started, "claim_events")
	rows, err := s.pool.Query(ctx, `WITH eligible AS (
        SELECT id FROM inbound_events
        WHERE (status IN ('Received', 'Retryable') AND available_at <= now())
           OR (status = 'Processing' AND lease_expires_at <= now())
        ORDER BY available_at, id
        FOR UPDATE SKIP LOCKED LIMIT $1
    )
	UPDATE inbound_events e SET status = 'Processing', lease_expires_at = now() + $2 * interval '1 second'
    FROM eligible WHERE e.id = eligible.id
		RETURNING e.id, e.provider, e.channel, e.provider_event_id, e.event_type, e.payload,
		          coalesce(e.source_type,''),coalesce(e.source_group_id,''),coalesce(e.source_user_id,''),e.link_code_hash,
		          e.occurred_at,e.attempt_count`, limit, int(lease.Seconds()))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (inbound.Event, error) {
		var event inbound.Event
		err := row.Scan(&event.ID, &event.Provider, &event.Channel, &event.ProviderEventID, &event.Type, &event.Payload,
			&event.SourceType, &event.SourceGroupID, &event.SourceUserID, &event.LinkCodeHash, &event.OccurredAt, &event.AttemptCount)
		return event, err
	})
	return events, err
}

func emptyBytesToNil(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return value
}

func (s *Store) Complete(ctx context.Context, id int64, status inbound.Status, reason string) error {
	_, err := s.pool.Exec(ctx, `UPDATE inbound_events SET status=$2, processed_at=now(), lease_expires_at=NULL,
        failure_code=CASE WHEN $2='Ignored' THEN 'unsupported_event_type' END,
        failure_message=CASE WHEN $2='Ignored' THEN $3 END WHERE id=$1`, id, status, reason)
	return err
}

func (s *Store) Fail(ctx context.Context, event inbound.Event, code, message string) error {
	attempt := event.AttemptCount + 1
	delay, retry := inbound.RetryDelay(attempt)
	status := inbound.Failed
	if retry {
		status = inbound.Retryable
	}
	_, err := s.pool.Exec(ctx, `UPDATE inbound_events SET status=$2, attempt_count=$3,
		available_at=now()+$4 * interval '1 second', lease_expires_at=NULL, failure_code=$5, failure_message=$6,
        processed_at=CASE WHEN $2='Failed' THEN now() END WHERE id=$1`,
		event.ID, status, attempt, int(delay.Seconds()), code, message)
	return err
}

func (s *Store) Heartbeat(ctx context.Context, name string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO worker_heartbeats (worker_name, heartbeat_at) VALUES ($1, now())
        ON CONFLICT (worker_name) DO UPDATE SET heartbeat_at=excluded.heartbeat_at`, name)
	return err
}

func (s *Store) Cleanup(ctx context.Context) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	terminal := "status IN ('Processed','Ignored','Failed') AND lease_expires_at IS NULL"
	if _, err = tx.Exec(ctx, "UPDATE inbound_events SET payload=NULL WHERE payload IS NOT NULL AND received_at < now()-interval '30 days' AND "+terminal); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "DELETE FROM inbound_events WHERE received_at < now()-interval '90 days' AND "+terminal); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) slow(started time.Time, operation string) {
	duration := time.Since(started)
	if duration > 200*time.Millisecond {
		s.logger.Warn("slow_query", "operation", operation, "duration_ms", duration.Milliseconds())
	}
}
