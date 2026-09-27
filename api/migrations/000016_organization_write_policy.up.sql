DROP POLICY IF EXISTS organizations_member_policy ON organizations;
DROP POLICY IF EXISTS organizations_member_read ON organizations;
DROP POLICY IF EXISTS organizations_manager_update ON organizations;

CREATE POLICY organizations_member_read ON organizations FOR SELECT
    USING (id=app_organization_id() AND is_organization_member(app_user_id(),id));

CREATE POLICY organizations_manager_update ON organizations FOR UPDATE
    USING (id=app_organization_id() AND organization_role(app_user_id(),id) IN ('Owner','Admin'))
    WITH CHECK (id=app_organization_id() AND organization_role(app_user_id(),id) IN ('Owner','Admin'));
