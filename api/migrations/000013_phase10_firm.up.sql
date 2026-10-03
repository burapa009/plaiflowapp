-- Expand-only. Application rollback disables Phase 10 routes; it does not drop client grants.
ALTER TABLE organization_plan_periods DROP CONSTRAINT organization_plan_periods_plan_key_check;
ALTER TABLE organization_plan_periods ADD CONSTRAINT organization_plan_periods_plan_key_check
    CHECK (plan_key IN ('Starter','Business','AccountingFirm'));
ALTER TABLE organization_plan_periods ADD CONSTRAINT accounting_firm_manual_period
    CHECK (plan_key <> 'AccountingFirm' OR source = 'manual');

DROP POLICY drive_oauth_attempts_owner_policy ON drive_oauth_attempts;
CREATE POLICY drive_oauth_attempts_manager_policy ON drive_oauth_attempts FOR ALL
    USING (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))
    WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));
DROP POLICY drive_connections_owner_write ON drive_connections;
CREATE POLICY drive_connections_manager_write ON drive_connections FOR ALL
    USING (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))
    WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));

CREATE TABLE firm_access_grants (
    id uuid PRIMARY KEY,
    firm_organization_id uuid NOT NULL REFERENCES organizations(id),
    client_organization_id uuid NOT NULL REFERENCES organizations(id),
    status text NOT NULL CHECK (status IN ('Requested','ClientApproved','Active','Cancelled','Revoked','Expired')),
    scopes text[] NOT NULL DEFAULT ARRAY['read']::text[],
    revision integer NOT NULL DEFAULT 1 CHECK (revision > 0),
    requested_by uuid NOT NULL REFERENCES users(id),
    approved_by uuid REFERENCES users(id),
    accepted_by uuid REFERENCES users(id),
    ended_by uuid REFERENCES users(id),
    requested_at timestamptz NOT NULL,
    request_expires_at timestamptz NOT NULL,
    approved_at timestamptz,
    accepted_at timestamptz,
    active_expires_at timestamptz,
    ended_at timestamptz,
    CHECK (firm_organization_id <> client_organization_id),
    CHECK (request_expires_at > requested_at),
    CHECK (status <> 'Active' OR (accepted_at IS NOT NULL AND active_expires_at > accepted_at)),
    UNIQUE (id,firm_organization_id,client_organization_id)
);
CREATE UNIQUE INDEX firm_access_grants_one_open_pair ON firm_access_grants
    (firm_organization_id,client_organization_id)
    WHERE status IN ('Requested','ClientApproved','Active');
CREATE INDEX firm_access_grants_firm_page ON firm_access_grants
    (firm_organization_id,status,client_organization_id,id);
CREATE INDEX firm_access_grants_client_page ON firm_access_grants
    (client_organization_id,status,firm_organization_id,id);
CREATE INDEX firm_access_grants_expiry ON firm_access_grants
    (status,request_expires_at,active_expires_at,id)
    WHERE status IN ('Requested','ClientApproved','Active');

CREATE TABLE firm_client_assignments (
    id uuid PRIMARY KEY,
    grant_id uuid NOT NULL,
    firm_organization_id uuid NOT NULL,
    client_organization_id uuid NOT NULL,
    staff_user_id uuid NOT NULL REFERENCES users(id),
    assigned_by uuid NOT NULL REFERENCES users(id),
    assigned_at timestamptz NOT NULL,
    ended_at timestamptz,
    FOREIGN KEY (grant_id,firm_organization_id,client_organization_id)
        REFERENCES firm_access_grants(id,firm_organization_id,client_organization_id)
);
CREATE UNIQUE INDEX firm_client_assignments_one_active ON firm_client_assignments (grant_id,staff_user_id)
    WHERE ended_at IS NULL;
CREATE INDEX firm_client_assignments_staff ON firm_client_assignments
    (firm_organization_id,staff_user_id,client_organization_id,grant_id)
    WHERE ended_at IS NULL;

ALTER TABLE firm_access_grants ENABLE ROW LEVEL SECURITY;
ALTER TABLE firm_client_assignments ENABLE ROW LEVEL SECURITY;
CREATE POLICY firm_grants_member_read ON firm_access_grants FOR SELECT USING (
    (firm_organization_id = app_organization_id() AND is_organization_member(app_user_id(),firm_organization_id))
    OR (client_organization_id = app_organization_id() AND is_organization_member(app_user_id(),client_organization_id)));
CREATE POLICY firm_grants_firm_request ON firm_access_grants FOR INSERT WITH CHECK (
    firm_organization_id = app_organization_id()
    AND organization_role(app_user_id(),firm_organization_id) IN ('Owner','Admin'));
CREATE POLICY firm_grants_manager_update ON firm_access_grants FOR UPDATE USING (
    (firm_organization_id = app_organization_id() AND organization_role(app_user_id(),firm_organization_id) IN ('Owner','Admin'))
    OR (client_organization_id = app_organization_id() AND organization_role(app_user_id(),client_organization_id) IN ('Owner','Admin')))
    WITH CHECK (
    (firm_organization_id = app_organization_id() AND organization_role(app_user_id(),firm_organization_id) IN ('Owner','Admin'))
    OR (client_organization_id = app_organization_id() AND organization_role(app_user_id(),client_organization_id) IN ('Owner','Admin')));
CREATE POLICY firm_assignments_manager ON firm_client_assignments FOR ALL USING (
    firm_organization_id = app_organization_id()
    AND organization_role(app_user_id(),firm_organization_id) IN ('Owner','Admin'))
    WITH CHECK (firm_organization_id = app_organization_id()
    AND organization_role(app_user_id(),firm_organization_id) IN ('Owner','Admin'));
CREATE POLICY firm_assignments_staff_read ON firm_client_assignments FOR SELECT USING (
    firm_organization_id = app_organization_id() AND staff_user_id = app_user_id()
    AND is_organization_member(app_user_id(),firm_organization_id));
CREATE POLICY firm_assignments_client_roster ON firm_client_assignments FOR SELECT USING (
    client_organization_id = app_organization_id()
    AND organization_role(app_user_id(),client_organization_id) IN ('Owner','Admin'));

CREATE FUNCTION firm_grant_allows(candidate_user_id uuid, candidate_firm_id uuid,
    candidate_client_id uuid, needed_scope text) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
    SELECT candidate_user_id = app_user_id() AND candidate_firm_id = app_organization_id()
        AND effective_organization_plan(candidate_firm_id) = 'AccountingFirm'
        AND is_organization_member(candidate_user_id,candidate_firm_id)
        AND EXISTS (
            SELECT 1 FROM firm_access_grants g
            JOIN firm_client_assignments a ON a.grant_id = g.id
            WHERE g.firm_organization_id = candidate_firm_id
              AND g.client_organization_id = candidate_client_id
              AND g.status = 'Active' AND g.active_expires_at > now()
              AND a.staff_user_id = candidate_user_id AND a.ended_at IS NULL
              AND 'read' = ANY(g.scopes) AND needed_scope = ANY(g.scopes)
        )
$$;
CREATE FUNCTION firm_grant_partner_name(candidate_grant_id uuid) RETURNS text
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
    SELECT o.name FROM firm_access_grants g JOIN organizations o ON o.id =
        CASE WHEN g.firm_organization_id = app_organization_id() THEN g.client_organization_id
             ELSE g.firm_organization_id END
    WHERE g.id = candidate_grant_id AND
        (g.client_organization_id = app_organization_id() OR g.approved_by IS NOT NULL) AND
        ((g.firm_organization_id = app_organization_id() AND is_organization_member(app_user_id(),g.firm_organization_id))
         OR (g.client_organization_id = app_organization_id() AND is_organization_member(app_user_id(),g.client_organization_id)))
$$;
CREATE POLICY documents_firm_read ON documents FOR SELECT USING (
    status IN ('Available','Archived')
    AND NOT group_restricted
    AND firm_grant_allows(app_user_id(),app_organization_id(),organization_id,'read'));
