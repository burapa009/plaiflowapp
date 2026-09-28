-- Expand only. Existing OCR artifacts and jobs retain the railway provider.
ALTER TABLE document_ocr_runs
  ADD COLUMN provider text NOT NULL DEFAULT 'railway' CHECK (provider IN ('railway','runpod')),
  ADD COLUMN provider_job_id text,
  ADD COLUMN provider_submitted_at timestamptz,
  ADD COLUMN processing_ms bigint CHECK (processing_ms IS NULL OR processing_ms BETWEEN 0 AND 900000),
  ADD COLUMN provider_execution_ms bigint CHECK (provider_execution_ms IS NULL OR provider_execution_ms >= 0),
  ADD COLUMN provider_queue_ms bigint CHECK (provider_queue_ms IS NULL OR provider_queue_ms >= 0),
  ADD COLUMN gpu_class text,
  ADD COLUMN provider_cost_microusd bigint CHECK (provider_cost_microusd IS NULL OR provider_cost_microusd >= 0);
CREATE INDEX document_ocr_provider_job ON document_ocr_runs (provider,provider_job_id)
  WHERE provider_job_id IS NOT NULL;

-- Only human-confirmed fields enter the bounded document matching index.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
ALTER TABLE document_extraction_reviews ADD CONSTRAINT document_extraction_reviews_org_id_unique
  UNIQUE (organization_id,id);
CREATE TABLE document_match_index (
  organization_id uuid NOT NULL,
  document_id uuid NOT NULL,
  review_id uuid NOT NULL,
  document_type text NOT NULL,
  issue_date date,
  total_amount numeric(18,2),
  document_number text NOT NULL DEFAULT '',
  seller_tax_id text NOT NULL DEFAULT '',
  seller_name text NOT NULL DEFAULT '',
  PRIMARY KEY (organization_id,document_id),
  FOREIGN KEY (organization_id,document_id) REFERENCES documents(organization_id,id),
  FOREIGN KEY (organization_id,review_id) REFERENCES document_extraction_reviews(organization_id,id)
);
CREATE INDEX document_match_amount_date ON document_match_index (organization_id,total_amount,issue_date)
  WHERE total_amount IS NOT NULL AND issue_date IS NOT NULL;
ALTER TABLE document_match_index ENABLE ROW LEVEL SECURITY;
CREATE POLICY document_match_read ON document_match_index FOR SELECT USING
  (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
    AND EXISTS (SELECT 1 FROM documents d WHERE d.organization_id=document_match_index.organization_id
      AND d.id=document_match_index.document_id AND d.status IN ('Available','Archived')
      AND (organization_role(app_user_id(),d.organization_id) IN ('Owner','Admin')
        OR (d.submitted_by_user_id=app_user_id() AND NOT d.group_restricted) OR d.assignee_user_id=app_user_id()
        OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id
          AND ds.document_id=d.id AND ds.submitted_by_user_id=app_user_id() AND NOT ds.group_source))));
CREATE POLICY document_match_write ON document_match_index FOR INSERT WITH CHECK
  (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));
CREATE POLICY document_match_update ON document_match_index FOR UPDATE USING
  (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'))
  WITH CHECK (organization_id=app_organization_id() AND organization_role(app_user_id(),organization_id) IN ('Owner','Admin'));
