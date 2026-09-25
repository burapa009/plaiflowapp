-- Safe only before any Phase 10 grant has been created. Live rollback uses the feature flag.
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM firm_access_grants) THEN
        RAISE EXCEPTION 'Phase 10 grants exist; disable the feature instead of dropping client access history';
    END IF;
END $$;
DROP POLICY IF EXISTS documents_firm_read ON documents;
DROP FUNCTION IF EXISTS firm_grant_partner_name(uuid);
DROP FUNCTION IF EXISTS firm_grant_allows(uuid,uuid,uuid,text);
DROP TABLE IF EXISTS firm_client_assignments;
DROP TABLE IF EXISTS firm_access_grants;
DROP POLICY IF EXISTS drive_oauth_attempts_manager_policy ON drive_oauth_attempts;
CREATE POLICY drive_oauth_attempts_owner_policy ON drive_oauth_attempts FOR ALL
    USING (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id)='Owner')
    WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id)='Owner');
DROP POLICY IF EXISTS drive_connections_manager_write ON drive_connections;
CREATE POLICY drive_connections_owner_write ON drive_connections FOR ALL
    USING (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id)='Owner')
    WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id)='Owner');
ALTER TABLE organization_plan_periods DROP CONSTRAINT IF EXISTS accounting_firm_manual_period;
ALTER TABLE organization_plan_periods DROP CONSTRAINT IF EXISTS organization_plan_periods_plan_key_check;
ALTER TABLE organization_plan_periods ADD CONSTRAINT organization_plan_periods_plan_key_check
    CHECK (plan_key IN ('Starter','Business'));
