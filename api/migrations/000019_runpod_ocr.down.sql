DROP TABLE IF EXISTS document_match_index;
ALTER TABLE document_extraction_reviews DROP CONSTRAINT IF EXISTS document_extraction_reviews_org_id_unique;
DROP INDEX IF EXISTS document_ocr_provider_job;
ALTER TABLE document_ocr_runs
  DROP COLUMN IF EXISTS provider_cost_microusd,
  DROP COLUMN IF EXISTS gpu_class,
  DROP COLUMN IF EXISTS provider_queue_ms,
  DROP COLUMN IF EXISTS provider_execution_ms,
  DROP COLUMN IF EXISTS processing_ms,
  DROP COLUMN IF EXISTS provider_submitted_at,
  DROP COLUMN IF EXISTS provider_job_id,
  DROP COLUMN IF EXISTS provider;
