CREATE TABLE document_usage_periods (
    organization_id uuid NOT NULL REFERENCES organizations(id),
    starts_at timestamptz NOT NULL,
    ends_at timestamptz NOT NULL CHECK (ends_at > starts_at),
    timezone text NOT NULL,
    PRIMARY KEY (organization_id, starts_at)
);

CREATE TABLE documents (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    content_sha256 bytea NOT NULL CHECK (octet_length(content_sha256) = 32),
    storage_key text,
    display_filename text NOT NULL CHECK (length(display_filename) BETWEEN 1 AND 240),
    detected_mime text NOT NULL CHECK (detected_mime IN ('application/pdf','image/jpeg','image/png')),
    byte_size bigint NOT NULL CHECK (byte_size BETWEEN 1 AND 20971520),
    status text NOT NULL CHECK (status IN ('Available','Archived','Trash','Purged')),
    submitted_by_user_id uuid,
    assignee_user_id uuid,
    accepted_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    trashed_at timestamptz,
    purged_at timestamptz,
    usage_period_start timestamptz NOT NULL,
    UNIQUE (organization_id,id),
    FOREIGN KEY (organization_id,submitted_by_user_id) REFERENCES memberships(organization_id,user_id),
    FOREIGN KEY (organization_id,assignee_user_id) REFERENCES memberships(organization_id,user_id),
    FOREIGN KEY (organization_id,usage_period_start) REFERENCES document_usage_periods(organization_id,starts_at),
    CHECK ((status='Purged') = (purged_at IS NOT NULL)),
    CHECK ((status='Purged') = (storage_key IS NULL))
);
CREATE UNIQUE INDEX documents_active_content_idx ON documents (organization_id,content_sha256)
    WHERE status <> 'Purged';
CREATE INDEX documents_list_idx ON documents (organization_id,status,accepted_at DESC,id DESC);
CREATE INDEX documents_assignee_idx ON documents (organization_id,assignee_user_id,accepted_at DESC,id DESC);

CREATE TABLE document_sources (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL,
    document_id uuid NOT NULL,
    channel text NOT NULL CHECK (channel IN ('Web','LINE','Drive')),
    origin_key text NOT NULL,
    submitted_by_user_id uuid,
    drive_connection_id uuid,
    drive_file_id text,
    drive_revision text,
    created_at timestamptz NOT NULL,
    UNIQUE (organization_id,channel,origin_key),
    FOREIGN KEY (organization_id,document_id) REFERENCES documents(organization_id,id),
    FOREIGN KEY (organization_id,submitted_by_user_id) REFERENCES memberships(organization_id,user_id),
    FOREIGN KEY (drive_connection_id) REFERENCES drive_connections(organization_id),
    CHECK ((channel='Drive' AND drive_connection_id IS NOT NULL AND drive_file_id IS NOT NULL AND drive_revision IS NOT NULL)
        OR (channel<>'Drive' AND drive_connection_id IS NULL AND drive_file_id IS NULL AND drive_revision IS NULL))
);
CREATE INDEX document_sources_document_idx ON document_sources (organization_id,document_id,created_at);

CREATE TABLE document_usage_charges (
    organization_id uuid NOT NULL,
    document_id uuid NOT NULL,
    period_start timestamptz NOT NULL,
    accepted_at timestamptz NOT NULL,
    PRIMARY KEY (organization_id,document_id),
    FOREIGN KEY (organization_id,document_id) REFERENCES documents(organization_id,id),
    FOREIGN KEY (organization_id,period_start) REFERENCES document_usage_periods(organization_id,starts_at)
);
CREATE INDEX document_usage_charges_period_idx ON document_usage_charges (organization_id,period_start);
CREATE INDEX document_usage_charges_trial_idx ON document_usage_charges (organization_id,accepted_at);

CREATE TABLE document_intake_attempts (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    actor_user_id uuid,
    channel text NOT NULL CHECK (channel IN ('Web','LINE','Drive')),
    origin_key text NOT NULL,
    status text NOT NULL CHECK (status IN ('Receiving','Checking','Accepted','Rejected')),
    rejection_code text,
    document_id uuid,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    UNIQUE (organization_id,channel,origin_key),
    FOREIGN KEY (organization_id,document_id) REFERENCES documents(organization_id,id),
    FOREIGN KEY (organization_id,actor_user_id) REFERENCES memberships(organization_id,user_id)
);
CREATE INDEX document_intake_attempts_inbox_idx ON document_intake_attempts (organization_id,status,created_at DESC,id DESC);
CREATE INDEX document_intake_attempts_expiry_idx ON document_intake_attempts (expires_at) WHERE status='Rejected';

ALTER TABLE document_usage_periods ENABLE ROW LEVEL SECURITY;
ALTER TABLE documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_sources ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_usage_charges ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_intake_attempts ENABLE ROW LEVEL SECURITY;

CREATE POLICY document_usage_periods_member_read ON document_usage_periods FOR SELECT
    USING (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id));
CREATE POLICY document_usage_periods_member_insert ON document_usage_periods FOR INSERT
    WITH CHECK (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id));
CREATE POLICY documents_member_read ON documents FOR SELECT
    USING (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
       AND (organization_role(app_user_id(),organization_id) IN ('Owner','Admin')
            OR submitted_by_user_id=app_user_id() OR assignee_user_id=app_user_id()
            OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=documents.organization_id
                AND ds.document_id=documents.id AND ds.submitted_by_user_id=app_user_id())));
CREATE POLICY documents_member_insert ON documents FOR INSERT
    WITH CHECK (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
        AND submitted_by_user_id=app_user_id());
CREATE POLICY documents_manager_update ON documents FOR UPDATE
    USING (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))
    WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));
CREATE POLICY document_sources_member_read ON document_sources FOR SELECT
    USING (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
        AND (organization_role(app_user_id(),organization_id) IN ('Owner','Admin') OR submitted_by_user_id=app_user_id()));
CREATE POLICY document_sources_member_insert ON document_sources FOR INSERT
    WITH CHECK (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
        AND submitted_by_user_id=app_user_id());
CREATE POLICY document_usage_charges_member_read ON document_usage_charges FOR SELECT
    USING (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id));
CREATE POLICY document_usage_charges_member_insert ON document_usage_charges FOR INSERT
    WITH CHECK (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id));
CREATE POLICY document_intake_attempts_member_read ON document_intake_attempts FOR SELECT
    USING (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
        AND (actor_user_id=app_user_id() OR organization_role(app_user_id(),organization_id) IN ('Owner','Admin')));
CREATE POLICY document_intake_attempts_member_insert ON document_intake_attempts FOR INSERT
    WITH CHECK (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
        AND actor_user_id=app_user_id());
CREATE POLICY document_intake_attempts_member_update ON document_intake_attempts FOR UPDATE
    USING (organization_id=app_organization_id() AND actor_user_id=app_user_id())
    WITH CHECK (organization_id=app_organization_id() AND actor_user_id=app_user_id());

CREATE FUNCTION lookup_document_for_intake(candidate_organization_id uuid, candidate_sha256 bytea)
RETURNS TABLE(document_id uuid, document_status text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
    SELECT d.id,d.status FROM documents d
    WHERE d.organization_id=candidate_organization_id AND d.content_sha256=candidate_sha256
      AND d.status<>'Purged' AND candidate_organization_id=app_organization_id()
      AND is_organization_member(app_user_id(),candidate_organization_id)
    LIMIT 1
$$;
