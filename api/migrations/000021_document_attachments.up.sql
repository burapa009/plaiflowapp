CREATE TABLE document_attachments (
    organization_id uuid NOT NULL,
    parent_document_id uuid NOT NULL,
    document_id uuid NOT NULL,
    attached_by_user_id uuid NOT NULL,
    attached_at timestamptz NOT NULL,
    PRIMARY KEY (organization_id,parent_document_id,document_id),
    FOREIGN KEY (organization_id,parent_document_id) REFERENCES documents(organization_id,id),
    FOREIGN KEY (organization_id,document_id) REFERENCES documents(organization_id,id),
    FOREIGN KEY (organization_id,attached_by_user_id) REFERENCES memberships(organization_id,user_id),
    CHECK (parent_document_id <> document_id)
);
CREATE INDEX document_attachments_document_idx ON document_attachments (organization_id,document_id);

ALTER TABLE document_attachments ENABLE ROW LEVEL SECURITY;
CREATE POLICY document_attachments_read ON document_attachments FOR SELECT USING (
    organization_id=app_organization_id()
    AND EXISTS (SELECT 1 FROM documents WHERE organization_id=document_attachments.organization_id AND id=parent_document_id)
    AND EXISTS (SELECT 1 FROM documents WHERE organization_id=document_attachments.organization_id AND id=document_id)
);
CREATE POLICY document_attachments_insert ON document_attachments FOR INSERT WITH CHECK (
    organization_id=app_organization_id() AND attached_by_user_id=app_user_id()
    AND EXISTS (SELECT 1 FROM documents WHERE organization_id=document_attachments.organization_id AND id=parent_document_id)
    AND EXISTS (SELECT 1 FROM documents WHERE organization_id=document_attachments.organization_id AND id=document_id)
);
