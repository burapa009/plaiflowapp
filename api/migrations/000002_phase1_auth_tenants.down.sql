DROP POLICY line_link_codes_manager_policy ON line_link_codes;
DROP POLICY line_group_connections_manager_policy ON line_group_connections;
DROP POLICY invitations_member_policy ON invitations;
DROP POLICY memberships_delete_policy ON memberships;
DROP POLICY memberships_update_policy ON memberships;
DROP POLICY memberships_insert_policy ON memberships;
DROP POLICY memberships_select_policy ON memberships;
DROP POLICY organizations_member_policy ON organizations;

DROP TRIGGER memberships_retain_owner ON memberships;
DROP FUNCTION enforce_organization_owner();
DROP FUNCTION transfer_organization_ownership(uuid, uuid, uuid, timestamptz);
DROP FUNCTION accept_member_invitation(bytea, uuid, timestamptz);
DROP FUNCTION claim_member_invitation(bytea, bytea, uuid, text, timestamptz, timestamptz);
DROP FUNCTION list_user_organizations(uuid);
DROP FUNCTION create_user_organization(uuid, uuid, text, timestamptz);
DROP FUNCTION organization_role(uuid, uuid);
DROP FUNCTION is_organization_member(uuid, uuid);
DROP FUNCTION app_organization_id();
DROP FUNCTION app_user_id();

DROP INDEX inbound_events_link_attempt_idx;
ALTER TABLE inbound_events
    DROP COLUMN link_code_hash,
    DROP COLUMN source_user_id,
    DROP COLUMN source_group_id,
    DROP COLUMN source_type;

DROP TRIGGER audit_events_append_only ON audit_events;
DROP FUNCTION reject_audit_event_mutation();
DROP TABLE audit_events;
DROP TABLE line_link_codes;
DROP TABLE line_group_connections;
DROP TABLE invite_handoffs;
DROP TABLE invitations;
DROP TABLE memberships;
DROP TABLE organizations;
DROP TABLE auth_transactions;
DROP TABLE sessions;
DROP TABLE auth_identities;
DROP TABLE users;
