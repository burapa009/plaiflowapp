CREATE TABLE drive_oauth_attempts (
    state_hash bytea PRIMARY KEY CHECK (octet_length(state_hash)=32),
    browser_hash bytea NOT NULL CHECK (octet_length(browser_hash)=32),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    organization_name text NOT NULL,
    user_id uuid NOT NULL REFERENCES users(id),
    session_id uuid NOT NULL REFERENCES sessions(id),
    code_verifier text NOT NULL,
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL CHECK (expires_at>created_at AND expires_at<=created_at+interval '10 minutes'),
    consumed_at timestamptz
);

CREATE INDEX drive_oauth_attempts_expiry_idx ON drive_oauth_attempts (expires_at)
    WHERE consumed_at IS NULL;

CREATE TABLE drive_connections (
    organization_id uuid PRIMARY KEY REFERENCES organizations(id),
    status text NOT NULL CHECK (status IN ('Connected','Reauthorization Required','Not Connected')),
    google_subject text,
    google_email text,
    folder_id text,
    authorizer_user_id uuid REFERENCES users(id),
    encrypted_refresh_token bytea,
    token_nonce bytea,
    credential_generation bigint NOT NULL DEFAULT 0 CHECK (credential_generation>=0),
    connected_at timestamptz,
    updated_at timestamptz NOT NULL,
    disconnected_at timestamptz,
    CHECK ((status='Connected' AND google_subject IS NOT NULL AND folder_id IS NOT NULL
        AND authorizer_user_id IS NOT NULL AND encrypted_refresh_token IS NOT NULL AND token_nonce IS NOT NULL)
        OR (status<>'Connected' AND encrypted_refresh_token IS NULL AND token_nonce IS NULL))
);

CREATE TABLE drive_reconnect_tasks (
    organization_id uuid NOT NULL REFERENCES organizations(id),
    credential_generation bigint NOT NULL,
    task_id uuid NOT NULL,
    created_at timestamptz NOT NULL,
    resolved_at timestamptz,
    PRIMARY KEY (organization_id,credential_generation),
    FOREIGN KEY (organization_id,task_id) REFERENCES tasks(organization_id,id)
);

CREATE FUNCTION consume_drive_oauth_attempt(candidate_state_hash bytea, current_time timestamptz)
RETURNS SETOF drive_oauth_attempts
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    RETURN QUERY
    UPDATE drive_oauth_attempts SET consumed_at=current_time
    WHERE state_hash=candidate_state_hash AND consumed_at IS NULL AND expires_at>current_time
    RETURNING *;
END
$$;

CREATE FUNCTION mark_drive_reconnect_required(candidate_organization_id uuid, expected_generation bigint, current_time timestamptz, new_task_id uuid)
RETURNS boolean
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
DECLARE
    owner_id uuid;
BEGIN
    UPDATE drive_connections SET status='Reauthorization Required',encrypted_refresh_token=NULL,token_nonce=NULL,updated_at=current_time
    WHERE organization_id=candidate_organization_id AND credential_generation=expected_generation AND status='Connected';
    IF NOT FOUND THEN RETURN false; END IF;

    SELECT user_id INTO STRICT owner_id FROM memberships WHERE organization_id=candidate_organization_id AND role='Owner';
    IF NOT EXISTS (SELECT 1 FROM drive_reconnect_tasks WHERE organization_id=candidate_organization_id AND credential_generation=expected_generation) THEN
        INSERT INTO tasks
            (id,organization_id,title,description,creator_user_id,assignee_user_id,status,priority,created_at,updated_at,status_changed_at)
        VALUES
            (new_task_id,candidate_organization_id,'เชื่อมต่อ Google Drive ใหม่','Google ขอการยืนยันสิทธิ์ใหม่ กรุณาเปิด Connections และเชื่อมต่ออีกครั้ง',owner_id,owner_id,'Open','High',current_time,current_time,current_time);
        INSERT INTO drive_reconnect_tasks (organization_id,credential_generation,task_id,created_at)
        VALUES (candidate_organization_id,expected_generation,new_task_id,current_time);
    END IF;
    INSERT INTO audit_events (organization_id,event_type,target_type,target_id,outcome,reason_code,occurred_at)
    VALUES (candidate_organization_id,'drive.reconnect_required','drive_connection',candidate_organization_id::text,'success','invalid_grant',current_time);
    RETURN true;
END
$$;

ALTER TABLE drive_oauth_attempts ENABLE ROW LEVEL SECURITY;
ALTER TABLE drive_connections ENABLE ROW LEVEL SECURITY;
ALTER TABLE drive_reconnect_tasks ENABLE ROW LEVEL SECURITY;

CREATE POLICY drive_oauth_attempts_owner_policy ON drive_oauth_attempts FOR ALL
    USING (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id)='Owner')
    WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id)='Owner');
CREATE POLICY drive_connections_member_read ON drive_connections FOR SELECT
    USING (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id));
CREATE POLICY drive_connections_owner_write ON drive_connections FOR ALL
    USING (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id)='Owner')
    WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id)='Owner');
CREATE POLICY drive_reconnect_tasks_member_read ON drive_reconnect_tasks FOR SELECT
    USING (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id));
