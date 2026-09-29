-- Empty by default. Operator-only model selection keeps the pilot tenant scoped.
CREATE TABLE organization_ocr_models (
 organization_id uuid PRIMARY KEY REFERENCES organizations(id),
 model_version text NOT NULL CHECK (model_version='typhoon-ocr1.5-2b-openthai2-q4-v1'),
 preprocessing_version text NOT NULL CHECK (preprocessing_version='markdown-v1')
);
ALTER TABLE organization_ocr_models ENABLE ROW LEVEL SECURITY;
CREATE POLICY organization_ocr_models_read ON organization_ocr_models FOR SELECT USING
 (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id));

CREATE FUNCTION document_ocr_model(org uuid) RETURNS TABLE(model_version text,preprocessing_version text)
 LANGUAGE sql STABLE AS $$
 SELECT coalesce(o.model_version,c.model_version),coalesce(o.preprocessing_version,c.preprocessing_version)
 FROM ocr_settings c LEFT JOIN organization_ocr_models o ON o.organization_id=org
 WHERE c.singleton AND c.enabled
$$;

CREATE OR REPLACE FUNCTION enqueue_document_ocr() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE cfg record; jid uuid; fp text;
BEGIN
 SELECT * INTO cfg FROM document_ocr_model(NEW.organization_id);
 IF NOT FOUND THEN RETURN NEW; END IF;
 fp := NEW.id::text || ':' || encode(NEW.content_sha256,'hex') || ':' || cfg.model_version || ':' || cfg.preprocessing_version;
 jid := gen_random_uuid();
 INSERT INTO durable_jobs(id,organization_id,requester_user_id,kind,status,payload,idempotency_key,max_attempts,available_at,created_at)
 VALUES(jid,NEW.organization_id,NEW.submitted_by_user_id,'ocr','Queued',
 jsonb_build_object('document_id',NEW.id,'sha256',encode(NEW.content_sha256,'hex'),'model_version',cfg.model_version,'preprocessing_version',cfg.preprocessing_version),
 fp,3,now(),now());
 INSERT INTO document_ocr_runs(job_id,organization_id,document_id,fingerprint) VALUES(jid,NEW.organization_id,NEW.id,fp);
 RETURN NEW;
END $$;
