CREATE TABLE document_classifications (
  ocr_job_id uuid PRIMARY KEY REFERENCES document_ocr_runs(job_id),
  organization_id uuid NOT NULL,
  document_id uuid NOT NULL,
  document_type text NOT NULL,
  result jsonb NOT NULL CHECK (jsonb_typeof(result) = 'object'),
  corrected_type text,
  corrected_by uuid REFERENCES users(id),
  corrected_at timestamptz,
  classified_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (organization_id,document_id) REFERENCES documents(organization_id,id)
);
CREATE INDEX document_classifications_type ON document_classifications (organization_id,document_type,classified_at DESC);
ALTER TABLE document_classifications ENABLE ROW LEVEL SECURITY;
CREATE POLICY document_classifications_read ON document_classifications FOR SELECT USING
  (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
    AND EXISTS (SELECT 1 FROM documents d WHERE d.organization_id=document_classifications.organization_id
      AND d.id=document_classifications.document_id AND d.status IN ('Available','Archived')
      AND (organization_role(app_user_id(),d.organization_id) IN ('Owner','Admin')
        OR (d.submitted_by_user_id=app_user_id() AND NOT d.group_restricted)
        OR d.assignee_user_id=app_user_id()
        OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id
          AND ds.document_id=d.id AND ds.submitted_by_user_id=app_user_id() AND NOT ds.group_source))));
CREATE POLICY document_classifications_correct ON document_classifications FOR UPDATE USING
  (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))
  WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));
