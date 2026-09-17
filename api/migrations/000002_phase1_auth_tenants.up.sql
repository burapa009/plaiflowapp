CREATE TABLE users (
    id uuid PRIMARY KEY,
    display_name text,
    picture_url text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    closed_at timestamptz
);

CREATE TABLE auth_identities (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    provider text NOT NULL CHECK (provider IN ('line', 'google')),
    issuer text NOT NULL,
    subject text,
    email text,
    email_verified boolean NOT NULL DEFAULT false,
    display_name text,
    picture_url text,
    linked_at timestamptz NOT NULL DEFAULT now(),
    disabled_at timestamptz,
    cooldown_until timestamptz,
    CHECK ((disabled_at IS NULL AND subject IS NOT NULL AND cooldown_until IS NULL)
        OR (disabled_at IS NOT NULL AND cooldown_until IS NOT NULL))
);

CREATE UNIQUE INDEX auth_identities_issuer_subject_uidx
    ON auth_identities (issuer, subject) WHERE subject IS NOT NULL;
CREATE UNIQUE INDEX auth_identities_user_provider_active_uidx
    ON auth_identities (user_id, provider) WHERE disabled_at IS NULL;
CREATE INDEX auth_identities_user_active_idx
    ON auth_identities (user_id, provider) WHERE disabled_at IS NULL;

CREATE TABLE sessions (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    csrf_hash bytea NOT NULL CHECK (octet_length(csrf_hash) = 32),
    created_at timestamptz NOT NULL,
    authenticated_at timestamptz NOT NULL,
    authenticated_provider text NOT NULL CHECK (authenticated_provider IN ('line', 'google')),
    last_seen_at timestamptz NOT NULL,
    idle_expires_at timestamptz NOT NULL,
    absolute_expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    CHECK (created_at <= authenticated_at AND authenticated_at <= absolute_expires_at),
    CHECK (created_at <= last_seen_at AND idle_expires_at <= absolute_expires_at)
);

CREATE INDEX sessions_user_active_idx
    ON sessions (user_id, absolute_expires_at) WHERE revoked_at IS NULL;
CREATE INDEX sessions_expiry_idx
    ON sessions (absolute_expires_at, idle_expires_at) WHERE revoked_at IS NULL;

CREATE TABLE auth_transactions (
    state_hash bytea PRIMARY KEY CHECK (octet_length(state_hash) = 32),
    browser_hash bytea NOT NULL CHECK (octet_length(browser_hash) = 32),
    provider text NOT NULL CHECK (provider IN ('line', 'google')),
    issuer text NOT NULL,
    purpose text NOT NULL CHECK (purpose IN ('login', 'link', 'reauth')),
    nonce text NOT NULL,
    code_verifier text NOT NULL,
    return_to text NOT NULL,
    initiating_user_id uuid REFERENCES users(id),
    initiating_session_id uuid REFERENCES sessions(id),
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    CHECK (expires_at > created_at),
    CHECK ((purpose = 'login' AND initiating_user_id IS NULL AND initiating_session_id IS NULL)
        OR (purpose IN ('link', 'reauth') AND initiating_user_id IS NOT NULL AND initiating_session_id IS NOT NULL))
);

CREATE INDEX auth_transactions_initiator_idx
    ON auth_transactions (initiating_user_id, initiating_session_id, created_at DESC)
    WHERE consumed_at IS NULL;
CREATE INDEX auth_transactions_expiry_idx
    ON auth_transactions (expires_at) WHERE consumed_at IS NULL;

CREATE TABLE organizations (
    id uuid PRIMARY KEY,
    name text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 160),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE memberships (
    organization_id uuid NOT NULL REFERENCES organizations(id),
    user_id uuid NOT NULL REFERENCES users(id),
    role text NOT NULL CHECK (role IN ('Owner', 'Admin', 'Member')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, user_id)
);

CREATE UNIQUE INDEX memberships_one_owner_uidx
    ON memberships (organization_id) WHERE role = 'Owner';
CREATE INDEX memberships_user_organization_idx
    ON memberships (user_id, organization_id) INCLUDE (role);
CREATE INDEX memberships_organization_role_user_idx
    ON memberships (organization_id, role, user_id);

CREATE TABLE invitations (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    inviter_user_id uuid NOT NULL,
    role text NOT NULL DEFAULT 'Member' CHECK (role = 'Member'),
    token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    accepted_by_user_id uuid REFERENCES users(id),
    accepted_at timestamptz,
    FOREIGN KEY (organization_id, inviter_user_id) REFERENCES memberships (organization_id, user_id),
    CHECK (expires_at > created_at),
    CHECK ((accepted_by_user_id IS NULL) = (accepted_at IS NULL))
);

CREATE INDEX invitations_organization_pending_idx
    ON invitations (organization_id, created_at DESC)
    WHERE revoked_at IS NULL AND accepted_at IS NULL;
CREATE INDEX invitations_expiry_idx
    ON invitations (expires_at) WHERE revoked_at IS NULL AND accepted_at IS NULL;

CREATE TABLE invite_handoffs (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    invitation_id uuid NOT NULL REFERENCES invitations(id),
    return_to text NOT NULL,
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    CHECK (expires_at > created_at AND expires_at <= created_at + interval '5 minutes')
);

CREATE INDEX invite_handoffs_expiry_idx
    ON invite_handoffs (expires_at) WHERE consumed_at IS NULL;

CREATE TABLE line_group_connections (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    messaging_channel text NOT NULL,
    group_id text NOT NULL,
    status text NOT NULL CHECK (status IN ('connected', 'disconnected')),
    connected_by_user_id uuid NOT NULL,
    connected_at timestamptz NOT NULL,
    disconnected_by_user_id uuid,
    disconnected_at timestamptz,
    FOREIGN KEY (organization_id, connected_by_user_id) REFERENCES memberships (organization_id, user_id),
    FOREIGN KEY (organization_id, disconnected_by_user_id) REFERENCES memberships (organization_id, user_id),
    CHECK ((status = 'connected' AND disconnected_at IS NULL)
        OR (status = 'disconnected' AND disconnected_at IS NOT NULL))
);

CREATE UNIQUE INDEX line_group_connections_active_group_uidx
    ON line_group_connections (messaging_channel, group_id) WHERE status = 'connected';
CREATE INDEX line_group_connections_organization_status_idx
    ON line_group_connections (organization_id, status, connected_at DESC);

CREATE TABLE line_link_codes (
    id uuid PRIMARY KEY,
    code_hash bytea NOT NULL UNIQUE CHECK (octet_length(code_hash) = 32),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    initiating_user_id uuid NOT NULL,
    expected_line_subject text NOT NULL,
    messaging_channel text NOT NULL,
    purpose text NOT NULL DEFAULT 'group_link' CHECK (purpose = 'group_link'),
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    consumed_at timestamptz,
    FOREIGN KEY (organization_id, initiating_user_id) REFERENCES memberships (organization_id, user_id),
    CHECK (expires_at > created_at AND expires_at <= created_at + interval '10 minutes')
);

CREATE INDEX line_link_codes_user_organization_pending_idx
    ON line_link_codes (initiating_user_id, organization_id, created_at DESC)
    WHERE revoked_at IS NULL AND consumed_at IS NULL;
CREATE INDEX line_link_codes_expiry_idx
    ON line_link_codes (expires_at) WHERE revoked_at IS NULL AND consumed_at IS NULL;

CREATE TABLE audit_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    organization_id uuid REFERENCES organizations(id),
    actor_user_id uuid REFERENCES users(id),
    event_type text NOT NULL,
    target_type text,
    target_id text,
    request_id text,
    outcome text NOT NULL,
    reason_code text,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE INDEX audit_events_organization_time_idx
    ON audit_events (organization_id, occurred_at DESC, id DESC) WHERE organization_id IS NOT NULL;
CREATE INDEX audit_events_actor_time_idx
    ON audit_events (actor_user_id, occurred_at DESC, id DESC) WHERE actor_user_id IS NOT NULL;

CREATE FUNCTION reject_audit_event_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit events are append-only';
END
$$;

CREATE TRIGGER audit_events_append_only
    BEFORE UPDATE OR DELETE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION reject_audit_event_mutation();

ALTER TABLE inbound_events
    ADD COLUMN source_type text,
    ADD COLUMN source_group_id text,
    ADD COLUMN source_user_id text,
    ADD COLUMN link_code_hash bytea CHECK (link_code_hash IS NULL OR octet_length(link_code_hash) = 32);

CREATE INDEX inbound_events_link_attempt_idx
    ON inbound_events (link_code_hash, available_at) WHERE link_code_hash IS NOT NULL;

CREATE FUNCTION app_user_id() RETURNS uuid
LANGUAGE sql STABLE PARALLEL SAFE AS $$
    SELECT nullif(current_setting('app.user_id', true), '')::uuid
$$;

CREATE FUNCTION app_organization_id() RETURNS uuid
LANGUAGE sql STABLE PARALLEL SAFE AS $$
    SELECT nullif(current_setting('app.organization_id', true), '')::uuid
$$;

CREATE FUNCTION is_organization_member(candidate_user_id uuid, candidate_organization_id uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
    SELECT EXISTS (
        SELECT 1 FROM memberships
        WHERE user_id = candidate_user_id AND organization_id = candidate_organization_id
    )
$$;

CREATE FUNCTION organization_role(candidate_user_id uuid, candidate_organization_id uuid) RETURNS text
LANGUAGE sql STABLE SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
    SELECT role FROM memberships
    WHERE user_id = candidate_user_id AND organization_id = candidate_organization_id
$$;

CREATE FUNCTION create_user_organization(candidate_user_id uuid, new_organization_id uuid, organization_name text, created_time timestamptz)
RETURNS TABLE (id uuid, name text, role text)
LANGUAGE plpgsql SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    IF app_user_id() IS DISTINCT FROM candidate_user_id THEN
        RAISE EXCEPTION 'user context mismatch';
    END IF;
    INSERT INTO organizations (id,name,created_at,updated_at)
        VALUES (new_organization_id,btrim(organization_name),created_time,created_time);
    INSERT INTO memberships (organization_id,user_id,role,created_at,updated_at)
        VALUES (new_organization_id,candidate_user_id,'Owner',created_time,created_time);
    RETURN QUERY SELECT new_organization_id,btrim(organization_name),'Owner'::text;
END
$$;

CREATE FUNCTION list_user_organizations(candidate_user_id uuid)
RETURNS TABLE (id uuid, name text, role text)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    IF app_user_id() IS DISTINCT FROM candidate_user_id THEN
        RAISE EXCEPTION 'user context mismatch';
    END IF;
    RETURN QUERY
        SELECT o.id,o.name,m.role
        FROM memberships m JOIN organizations o ON o.id=m.organization_id
        WHERE m.user_id=candidate_user_id
        ORDER BY o.name,o.id;
END
$$;

CREATE FUNCTION claim_member_invitation(candidate_token_hash bytea, new_handoff_hash bytea, new_handoff_id uuid,
    safe_return_to text, at_time timestamptz, handoff_expiry timestamptz)
RETURNS TABLE (invitation_id uuid, organization_id uuid, organization_name text, inviter_name text, role text, expires_at timestamptz)
LANGUAGE plpgsql SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
DECLARE
    selected invitations%ROWTYPE;
BEGIN
    SELECT i.* INTO selected FROM invitations i
    WHERE i.token_hash=candidate_token_hash AND i.revoked_at IS NULL AND i.accepted_at IS NULL AND i.expires_at>at_time
      AND organization_role(i.inviter_user_id,i.organization_id) IN ('Owner','Admin')
    FOR UPDATE;
    IF NOT FOUND OR handoff_expiry>at_time+interval '5 minutes' THEN
        RETURN;
    END IF;
    INSERT INTO invite_handoffs (token_hash,invitation_id,return_to,created_at,expires_at)
        VALUES (new_handoff_hash,selected.id,safe_return_to,at_time,handoff_expiry);
    RETURN QUERY
        SELECT selected.id,o.id,o.name,coalesce(u.display_name,'สมาชิก PlaiFlow'),selected.role,selected.expires_at
        FROM organizations o JOIN users u ON u.id=selected.inviter_user_id WHERE o.id=selected.organization_id;
END
$$;

CREATE FUNCTION accept_member_invitation(candidate_handoff_hash bytea, candidate_user_id uuid, at_time timestamptz)
RETURNS TABLE (id uuid, name text, role text)
LANGUAGE plpgsql SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
DECLARE
    selected_handoff invite_handoffs%ROWTYPE;
    selected_invitation invitations%ROWTYPE;
BEGIN
    IF app_user_id() IS DISTINCT FROM candidate_user_id THEN
        RAISE EXCEPTION 'user context mismatch';
    END IF;
    SELECT h.* INTO selected_handoff FROM invite_handoffs h
    WHERE h.token_hash=candidate_handoff_hash AND h.consumed_at IS NULL AND h.expires_at>at_time
    FOR UPDATE;
    IF NOT FOUND THEN RETURN; END IF;
    SELECT i.* INTO selected_invitation FROM invitations i
    WHERE i.id=selected_handoff.invitation_id AND i.revoked_at IS NULL AND i.accepted_at IS NULL AND i.expires_at>at_time
      AND organization_role(i.inviter_user_id,i.organization_id) IN ('Owner','Admin')
    FOR UPDATE;
    IF NOT FOUND THEN RETURN; END IF;
    INSERT INTO memberships (organization_id,user_id,role,created_at,updated_at)
        VALUES (selected_invitation.organization_id,candidate_user_id,'Member',at_time,at_time)
        ON CONFLICT (organization_id,user_id) DO NOTHING;
    UPDATE invitations SET accepted_by_user_id=candidate_user_id,accepted_at=at_time WHERE invitations.id=selected_invitation.id;
    UPDATE invite_handoffs SET consumed_at=at_time WHERE token_hash=candidate_handoff_hash;
    INSERT INTO audit_events (organization_id,actor_user_id,event_type,target_type,target_id,outcome,occurred_at)
        VALUES (selected_invitation.organization_id,candidate_user_id,'invitation.redeem','invitation',selected_invitation.id::text,'success',at_time);
    RETURN QUERY SELECT o.id,o.name,'Member'::text FROM organizations o WHERE o.id=selected_invitation.organization_id;
END
$$;

CREATE FUNCTION transfer_organization_ownership(candidate_organization_id uuid, actor_user_id uuid, target_user_id uuid, at_time timestamptz)
RETURNS void
LANGUAGE plpgsql SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    IF app_user_id() IS DISTINCT FROM actor_user_id OR app_organization_id() IS DISTINCT FROM candidate_organization_id THEN
        RAISE EXCEPTION 'organization context mismatch';
    END IF;
    PERFORM 1 FROM memberships WHERE organization_id=candidate_organization_id AND user_id IN (actor_user_id,target_user_id) FOR UPDATE;
    IF organization_role(actor_user_id,candidate_organization_id)<>'Owner'
       OR NOT is_organization_member(target_user_id,candidate_organization_id)
       OR actor_user_id=target_user_id THEN
        RAISE EXCEPTION 'ownership transfer is unavailable';
    END IF;
    UPDATE memberships SET role='Admin',updated_at=at_time
        WHERE organization_id=candidate_organization_id AND user_id=actor_user_id;
    UPDATE memberships SET role='Owner',updated_at=at_time
        WHERE organization_id=candidate_organization_id AND user_id=target_user_id;
END
$$;

CREATE FUNCTION enforce_organization_owner() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM memberships WHERE organization_id=OLD.organization_id AND role='Owner') THEN
        RAISE EXCEPTION 'organization must retain an owner';
    END IF;
    RETURN NULL;
END
$$;

CREATE CONSTRAINT TRIGGER memberships_retain_owner
    AFTER UPDATE OR DELETE ON memberships
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW WHEN (OLD.role = 'Owner')
    EXECUTE FUNCTION enforce_organization_owner();

ALTER TABLE organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE memberships ENABLE ROW LEVEL SECURITY;
ALTER TABLE invitations ENABLE ROW LEVEL SECURITY;
ALTER TABLE line_group_connections ENABLE ROW LEVEL SECURITY;
ALTER TABLE line_link_codes ENABLE ROW LEVEL SECURITY;

CREATE POLICY organizations_member_policy ON organizations
    USING (id = app_organization_id() AND is_organization_member(app_user_id(), id))
    WITH CHECK (id = app_organization_id() AND is_organization_member(app_user_id(), id));

CREATE POLICY memberships_select_policy ON memberships FOR SELECT
    USING (organization_id = app_organization_id() AND is_organization_member(app_user_id(), organization_id));
CREATE POLICY memberships_insert_policy ON memberships FOR INSERT
    WITH CHECK (
        organization_id = app_organization_id()
        AND ((organization_role(app_user_id(), organization_id) IN ('Owner', 'Admin') AND role = 'Member')
            OR (organization_role(app_user_id(), organization_id) = 'Owner' AND role = 'Admin'))
    );
CREATE POLICY memberships_update_policy ON memberships FOR UPDATE
    USING (organization_id = app_organization_id() AND organization_role(app_user_id(), organization_id) = 'Owner')
    WITH CHECK (organization_id = app_organization_id() AND organization_role(app_user_id(), organization_id) = 'Owner');
CREATE POLICY memberships_delete_policy ON memberships FOR DELETE
    USING (
        organization_id = app_organization_id()
        AND ((organization_role(app_user_id(), organization_id) IN ('Owner', 'Admin') AND role = 'Member')
            OR (organization_role(app_user_id(), organization_id) = 'Owner' AND role = 'Admin')
            OR user_id = app_user_id())
    );

CREATE POLICY invitations_member_policy ON invitations
    USING (organization_id = app_organization_id() AND organization_role(app_user_id(), organization_id) IN ('Owner', 'Admin'))
    WITH CHECK (organization_id = app_organization_id() AND organization_role(app_user_id(), organization_id) IN ('Owner', 'Admin') AND role = 'Member');

CREATE POLICY line_group_connections_manager_policy ON line_group_connections
    USING (organization_id = app_organization_id() AND organization_role(app_user_id(), organization_id) IN ('Owner', 'Admin'))
    WITH CHECK (organization_id = app_organization_id() AND organization_role(app_user_id(), organization_id) IN ('Owner', 'Admin'));

CREATE POLICY line_link_codes_manager_policy ON line_link_codes
    USING (organization_id = app_organization_id() AND organization_role(app_user_id(), organization_id) IN ('Owner', 'Admin'))
    WITH CHECK (organization_id = app_organization_id() AND organization_role(app_user_id(), organization_id) IN ('Owner', 'Admin'));
