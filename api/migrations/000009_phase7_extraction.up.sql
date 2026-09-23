-- Expand-only: Phase 7 endpoints remain unavailable until API configuration is deployed.
CREATE TABLE document_extraction_reviews (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL,
    document_id uuid NOT NULL,
    ocr_job_id uuid NOT NULL REFERENCES document_ocr_runs(job_id),
    revision integer NOT NULL CHECK (revision > 0),
    object_key text NOT NULL UNIQUE,
    confirmed_by uuid NOT NULL,
    confirmed_at timestamptz NOT NULL,
    superseded_at timestamptz,
    FOREIGN KEY (organization_id,document_id) REFERENCES documents(organization_id,id),
    FOREIGN KEY (organization_id,confirmed_by) REFERENCES memberships(organization_id,user_id),
    UNIQUE (organization_id,document_id,revision)
);
CREATE UNIQUE INDEX document_extraction_current ON document_extraction_reviews (organization_id,document_id)
    WHERE superseded_at IS NULL;
CREATE INDEX document_extraction_export ON document_extraction_reviews (organization_id,confirmed_at DESC,id DESC)
    WHERE superseded_at IS NULL;

ALTER TABLE document_extraction_reviews ENABLE ROW LEVEL SECURITY;
CREATE POLICY document_extraction_read ON document_extraction_reviews FOR SELECT
    USING (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
        AND EXISTS (SELECT 1 FROM documents d WHERE d.organization_id=document_extraction_reviews.organization_id
            AND d.id=document_extraction_reviews.document_id AND d.status IN ('Available','Archived')));
CREATE POLICY document_extraction_write ON document_extraction_reviews FOR INSERT
    WITH CHECK (organization_id=app_organization_id()
        AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));
CREATE POLICY document_extraction_supersede ON document_extraction_reviews FOR UPDATE
    USING (organization_id=app_organization_id()
        AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))
    WITH CHECK (organization_id=app_organization_id()
        AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));
