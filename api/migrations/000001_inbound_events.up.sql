CREATE TABLE inbound_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    provider text NOT NULL,
    channel text NOT NULL,
    provider_event_id text NOT NULL,
    event_type text NOT NULL,
    payload jsonb,
    status text NOT NULL DEFAULT 'Received'
        CHECK (status IN ('Received', 'Processing', 'Processed', 'Retryable', 'Failed', 'Ignored')),
    attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    available_at timestamptz NOT NULL DEFAULT now(),
    lease_expires_at timestamptz,
    occurred_at timestamptz NOT NULL,
    received_at timestamptz NOT NULL DEFAULT now(),
    processed_at timestamptz,
    failure_code text,
    failure_message text,
    UNIQUE (provider, channel, provider_event_id)
);

CREATE INDEX inbound_events_claim_idx
    ON inbound_events (available_at, received_at)
    WHERE status IN ('Received', 'Retryable', 'Processing');

CREATE TABLE worker_heartbeats (
    worker_name text PRIMARY KEY,
    heartbeat_at timestamptz NOT NULL
);
