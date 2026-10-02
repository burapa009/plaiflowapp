CREATE TABLE expense_document_numbers (
    organization_id uuid NOT NULL REFERENCES organizations(id),
    document_type text NOT NULL CHECK (document_type IN ('receipt_substitute','cash_receipt','payment_voucher')),
    year integer NOT NULL,
    last_number integer NOT NULL CHECK (last_number > 0),
    PRIMARY KEY (organization_id,document_type,year)
);

CREATE TABLE expense_generated_documents (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL,
    document_id uuid NOT NULL,
    document_type text NOT NULL CHECK (document_type IN ('receipt_substitute','cash_receipt','payment_voucher')),
    document_number text NOT NULL DEFAULT '',
    status text NOT NULL CHECK (status IN ('incomplete','pending_signature','generating','generated','generation_failed')),
    form_data jsonb NOT NULL DEFAULT '{}'::jsonb,
    snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
    object_key text NOT NULL DEFAULT '',
    review_id uuid,
    created_by uuid NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    generated_at timestamptz,
    FOREIGN KEY (organization_id,document_id) REFERENCES documents(organization_id,id),
    FOREIGN KEY (organization_id,created_by) REFERENCES memberships(organization_id,user_id),
    UNIQUE (organization_id,document_id,document_type)
);
CREATE INDEX expense_generated_documents_document_idx ON expense_generated_documents(organization_id,document_id);
CREATE UNIQUE INDEX expense_generated_documents_number_idx ON expense_generated_documents(organization_id,document_type,document_number) WHERE document_number <> '';

ALTER TABLE expense_document_numbers ENABLE ROW LEVEL SECURITY;
CREATE POLICY expense_document_numbers_manage ON expense_document_numbers FOR ALL
    USING (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))
    WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));

ALTER TABLE expense_generated_documents ENABLE ROW LEVEL SECURITY;
CREATE POLICY expense_generated_documents_read ON expense_generated_documents FOR SELECT
    USING (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
        AND EXISTS (SELECT 1 FROM documents d WHERE d.organization_id=expense_generated_documents.organization_id
            AND d.id=expense_generated_documents.document_id AND d.status IN ('Available','Archived')));
CREATE POLICY expense_generated_documents_write ON expense_generated_documents FOR ALL
    USING (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))
    WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));
