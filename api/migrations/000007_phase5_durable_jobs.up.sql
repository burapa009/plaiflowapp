CREATE TABLE durable_jobs (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    requester_user_id uuid,
    kind text NOT NULL CHECK (kind IN ('export','ocr')),
    status text NOT NULL CHECK (status IN ('Queued','Running','Completed','Failed','Cancelled')),
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload)='object'),
    result jsonb CHECK (result IS NULL OR jsonb_typeof(result)='object'),
    idempotency_key text NOT NULL CHECK (length(idempotency_key) BETWEEN 1 AND 200),
    request_id text,
    attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count BETWEEN 0 AND 5),
    max_attempts integer NOT NULL DEFAULT 5 CHECK (max_attempts BETWEEN 1 AND 5),
    current_attempt_id uuid,
    lease_token_hash bytea CHECK (lease_token_hash IS NULL OR octet_length(lease_token_hash)=32),
    lease_expires_at timestamptz,
    worker_id text,
    available_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    started_at timestamptz,
    completed_at timestamptz,
    failed_at timestamptz,
    cancelled_at timestamptz,
    failure_code text,
    stale_reclaim_count integer NOT NULL DEFAULT 0 CHECK (stale_reclaim_count>=0),
    legacy_export_job_id uuid UNIQUE REFERENCES export_jobs(id),
    UNIQUE (organization_id,id),
    UNIQUE (organization_id,kind,idempotency_key),
    FOREIGN KEY (organization_id,requester_user_id) REFERENCES memberships(organization_id,user_id) ON DELETE SET NULL (requester_user_id),
    CHECK ((status='Running') = (current_attempt_id IS NOT NULL AND lease_token_hash IS NOT NULL AND lease_expires_at IS NOT NULL AND worker_id IS NOT NULL)),
    CHECK ((status='Completed') = (completed_at IS NOT NULL)),
    CHECK ((status='Failed') = (failed_at IS NOT NULL)),
    CHECK ((status='Cancelled') = (cancelled_at IS NOT NULL))
);

CREATE INDEX durable_jobs_claim_idx ON durable_jobs (available_at,created_at,id)
    WHERE status IN ('Queued','Running');
CREATE INDEX durable_jobs_kind_claim_idx ON durable_jobs (kind,available_at,created_at,id)
    WHERE status IN ('Queued','Running');
CREATE INDEX durable_jobs_organization_history_idx ON durable_jobs (organization_id,created_at DESC,id DESC);
CREATE INDEX durable_jobs_retention_idx ON durable_jobs (completed_at,failed_at,cancelled_at)
    WHERE status IN ('Completed','Failed','Cancelled');

CREATE TABLE durable_job_artifacts (
    job_id uuid PRIMARY KEY REFERENCES durable_jobs(id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    object_key text NOT NULL UNIQUE,
    format text NOT NULL CHECK (format IN ('csv','xlsx')),
    row_count bigint NOT NULL CHECK (row_count>=0),
    byte_count bigint NOT NULL CHECK (byte_count>=0),
    sha256 bytea NOT NULL CHECK (octet_length(sha256)=32),
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    deleted_at timestamptz,
    FOREIGN KEY (organization_id,job_id) REFERENCES durable_jobs(organization_id,id),
    CHECK (expires_at>created_at)
);
CREATE INDEX durable_job_artifacts_expiry_idx ON durable_job_artifacts (expires_at,job_id)
    WHERE deleted_at IS NULL;

CREATE TABLE durable_job_worker_heartbeats (
    worker_id text PRIMARY KEY,
    heartbeat_at timestamptz NOT NULL,
    environment text NOT NULL,
    supported_kinds text[] NOT NULL
);

INSERT INTO durable_jobs
    (id,organization_id,requester_user_id,kind,status,payload,idempotency_key,request_id,
     attempt_count,max_attempts,available_at,created_at,legacy_export_job_id)
SELECT id,organization_id,requester_user_id,'export','Queued',
       jsonb_build_object('export_type','tasks','format','csv','format_version',format_version,
           'filters',filters,'row_count',authorized_row_count,'data_as_of',coalesce(data_as_of,requested_at)),
       id::text,null,0,5,now(),requested_at,id
FROM export_jobs
WHERE mode='background' AND status IN ('Queued','Running')
ON CONFLICT (id) DO NOTHING;
