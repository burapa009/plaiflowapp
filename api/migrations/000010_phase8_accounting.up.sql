-- Expand-only and dormant until ACCOUNTING_ENABLED=true. Historical Phase 7 rows are untouched.
CREATE TABLE expense_categories (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    name text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 120),
    archived_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id,id)
);
CREATE UNIQUE INDEX expense_categories_active_name ON expense_categories (organization_id,lower(name)) WHERE archived_at IS NULL;

-- Editable starter labels only; these are not ledger or tax codes.
INSERT INTO expense_categories(id,organization_id,name)
SELECT gen_random_uuid(),o.id,seed.name FROM organizations o
CROSS JOIN (VALUES ('ค่าเดินทาง'),('ค่าอาหาร'),('วัสดุสำนักงาน')) AS seed(name);

CREATE TABLE accounting_rule_sets (
    organization_id uuid PRIMARY KEY REFERENCES organizations(id),
    version integer NOT NULL DEFAULT 0 CHECK (version >= 0)
);

CREATE TABLE accounting_mapping_rules (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL,
    vendor_id uuid NOT NULL,
    document_type text NOT NULL DEFAULT '' CHECK (document_type IN ('','receipt','invoice','tax_invoice')),
    category_id uuid NOT NULL,
    version integer NOT NULL CHECK (version > 0),
    approved_by uuid NOT NULL,
    approved_at timestamptz NOT NULL,
    retired_at timestamptz,
    FOREIGN KEY (organization_id,vendor_id) REFERENCES business_contacts(organization_id,id),
    FOREIGN KEY (organization_id,category_id) REFERENCES expense_categories(organization_id,id),
    FOREIGN KEY (organization_id,approved_by) REFERENCES memberships(organization_id,user_id),
    UNIQUE (organization_id,id)
);
CREATE INDEX accounting_rules_current ON accounting_mapping_rules (organization_id,vendor_id,document_type)
    WHERE retired_at IS NULL;

CREATE UNIQUE INDEX document_extraction_reviews_org_id ON document_extraction_reviews (organization_id,id);

CREATE TABLE accounting_suggestions (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL,
    document_id uuid NOT NULL,
    review_id uuid NOT NULL,
    review_revision integer NOT NULL CHECK (review_revision > 0),
    revision integer NOT NULL DEFAULT 1 CHECK (revision > 0),
    rule_set_version integer NOT NULL CHECK (rule_set_version >= 0),
    category_id uuid NOT NULL,
    category_name text NOT NULL CHECK (length(category_name) BETWEEN 1 AND 120),
    vendor_id uuid,
    vendor_contact_code text NOT NULL DEFAULT '' CHECK (length(vendor_contact_code) <= 64),
    unmatched_vendor_reason text NOT NULL DEFAULT '' CHECK (length(unmatched_vendor_reason) <= 240),
    suggestion_basis text NOT NULL CHECK (length(suggestion_basis) <= 64),
    rule_version integer,
    approved_by uuid NOT NULL,
    approved_at timestamptz NOT NULL,
    superseded_at timestamptz,
    FOREIGN KEY (organization_id,document_id) REFERENCES documents(organization_id,id),
    FOREIGN KEY (organization_id,review_id) REFERENCES document_extraction_reviews(organization_id,id),
    FOREIGN KEY (organization_id,category_id) REFERENCES expense_categories(organization_id,id),
    FOREIGN KEY (organization_id,vendor_id) REFERENCES business_contacts(organization_id,id),
    FOREIGN KEY (organization_id,approved_by) REFERENCES memberships(organization_id,user_id),
    UNIQUE (organization_id,document_id,revision)
);
CREATE UNIQUE INDEX accounting_suggestions_current ON accounting_suggestions (organization_id,document_id)
    WHERE superseded_at IS NULL;
CREATE INDEX accounting_suggestions_export ON accounting_suggestions (organization_id,approved_at DESC,id DESC)
    WHERE superseded_at IS NULL;

ALTER TABLE expense_categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE accounting_rule_sets ENABLE ROW LEVEL SECURITY;
ALTER TABLE accounting_mapping_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE accounting_suggestions ENABLE ROW LEVEL SECURITY;

CREATE POLICY expense_categories_read ON expense_categories FOR SELECT USING
    (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id));
CREATE POLICY expense_categories_write ON expense_categories FOR ALL USING
    (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))
    WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));
CREATE POLICY accounting_rule_sets_read ON accounting_rule_sets FOR SELECT USING
    (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id));
CREATE POLICY accounting_rule_sets_write ON accounting_rule_sets FOR ALL USING
    (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))
    WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));
CREATE POLICY accounting_rules_read ON accounting_mapping_rules FOR SELECT USING
    (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id));
CREATE POLICY accounting_rules_write ON accounting_mapping_rules FOR ALL USING
    (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))
    WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));
CREATE POLICY accounting_suggestions_read ON accounting_suggestions FOR SELECT USING
    (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
        AND EXISTS (SELECT 1 FROM documents d WHERE d.organization_id=accounting_suggestions.organization_id
            AND d.id=accounting_suggestions.document_id AND d.status IN ('Available','Archived')));
CREATE POLICY accounting_suggestions_write ON accounting_suggestions FOR ALL USING
    (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))
    WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));
