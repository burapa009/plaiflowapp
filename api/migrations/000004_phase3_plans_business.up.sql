CREATE TABLE organization_plan_periods (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    plan_key text NOT NULL CHECK (plan_key IN ('Starter','Business')),
    source text NOT NULL CHECK (source IN ('trial','manual','billing')),
    external_reference text,
    starts_at timestamptz NOT NULL,
    ends_at timestamptz NOT NULL CHECK (ends_at > starts_at),
    created_at timestamptz NOT NULL,
    UNIQUE (organization_id,source,external_reference)
);

CREATE INDEX organization_plan_periods_effective_idx
    ON organization_plan_periods (organization_id,starts_at DESC,ends_at DESC);

CREATE FUNCTION effective_organization_plan(candidate_organization_id uuid)
RETURNS text
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
    SELECT coalesce((
        SELECT plan_key FROM organization_plan_periods
        WHERE organization_id=candidate_organization_id AND starts_at<=now() AND ends_at>now()
        ORDER BY starts_at DESC,created_at DESC LIMIT 1
    ),'Free')
$$;

CREATE TABLE business_contacts (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    display_name text NOT NULL CHECK (length(btrim(display_name)) BETWEEN 1 AND 240),
    normalized_name text NOT NULL CHECK (length(normalized_name) BETWEEN 1 AND 240),
    contact_code text NOT NULL DEFAULT '' CHECK (length(contact_code) <= 64),
    country text NOT NULL DEFAULT 'TH' CHECK (country ~ '^[A-Z]{2}$'),
    tax_id text NOT NULL DEFAULT '' CHECK (length(tax_id) <= 64),
    branch_code text NOT NULL DEFAULT '' CHECK (length(branch_code) <= 32),
    is_customer boolean NOT NULL DEFAULT false,
    is_vendor boolean NOT NULL DEFAULT false,
    import_preview_id uuid,
    archived_at timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CHECK (is_customer OR is_vendor),
    UNIQUE (organization_id,id)
);

CREATE UNIQUE INDEX business_contacts_tax_branch_unique
    ON business_contacts (organization_id,country,tax_id,branch_code) WHERE tax_id <> '';
CREATE UNIQUE INDEX business_contacts_code_unique
    ON business_contacts (organization_id,lower(contact_code)) WHERE contact_code <> '';
CREATE INDEX business_contacts_vendor_search_idx
    ON business_contacts (organization_id,is_vendor,normalized_name text_pattern_ops,id) WHERE archived_at IS NULL;

CREATE TABLE business_import_previews (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    requester_user_id uuid NOT NULL REFERENCES users(id),
    rows jsonb NOT NULL CHECK (jsonb_typeof(rows)='array'),
    ready_count integer NOT NULL CHECK (ready_count >= 0),
    duplicate_count integer NOT NULL CHECK (duplicate_count >= 0),
    warning_count integer NOT NULL DEFAULT 0 CHECK (warning_count >= 0),
    error_count integer NOT NULL DEFAULT 0 CHECK (error_count >= 0),
    status text NOT NULL CHECK (status IN ('Preview','Committed','Expired')),
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    committed_at timestamptz,
    UNIQUE (organization_id,id)
);

ALTER TABLE business_contacts
    ADD CONSTRAINT business_contacts_import_preview_fk
    FOREIGN KEY (organization_id,import_preview_id)
    REFERENCES business_import_previews(organization_id,id);

CREATE INDEX business_import_previews_expiry_idx
    ON business_import_previews (status,expires_at,id) WHERE status='Preview';

ALTER TABLE organization_plan_periods ENABLE ROW LEVEL SECURITY;
ALTER TABLE business_contacts ENABLE ROW LEVEL SECURITY;
ALTER TABLE business_import_previews ENABLE ROW LEVEL SECURITY;

CREATE POLICY organization_plan_periods_member_read ON organization_plan_periods FOR SELECT
    USING (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id));
CREATE POLICY business_contacts_member_read ON business_contacts FOR SELECT
    USING (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id));
CREATE POLICY business_contacts_manager_write ON business_contacts FOR ALL
    USING (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))
    WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));
CREATE POLICY business_import_previews_manager ON business_import_previews FOR ALL
    USING (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))
    WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));
