-- Expand only; scheduling remains disabled until the runtime is verified.
CREATE TABLE ocr_settings (
 singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
 enabled boolean NOT NULL DEFAULT false,
 model_version text NOT NULL,
 preprocessing_version text NOT NULL
);
INSERT INTO ocr_settings VALUES (true,false,'paddleocr-3.7.0-ppocrv5-th-v1','v1');

CREATE TABLE document_ocr_runs (
 job_id uuid PRIMARY KEY REFERENCES durable_jobs(id),
 organization_id uuid NOT NULL,
 document_id uuid NOT NULL,
 fingerprint text NOT NULL,
 retry_of uuid REFERENCES durable_jobs(id),
 object_key text UNIQUE,
 result_sha256 text,
 result_bytes bigint CHECK (result_bytes BETWEEN 1 AND 8388608),
 page_count integer CHECK (page_count BETWEEN 1 AND 20),
 published_at timestamptz,
 superseded_at timestamptz,
 deleted_at timestamptz,
 FOREIGN KEY (organization_id,document_id) REFERENCES documents(organization_id,id),
 FOREIGN KEY (organization_id,job_id) REFERENCES durable_jobs(organization_id,id)
);
CREATE INDEX document_ocr_history ON document_ocr_runs (organization_id,document_id,published_at DESC);
CREATE UNIQUE INDEX document_ocr_published ON document_ocr_runs (organization_id,fingerprint) WHERE published_at IS NOT NULL AND deleted_at IS NULL;

CREATE FUNCTION enqueue_document_ocr() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE cfg ocr_settings; jid uuid; fp text;
BEGIN
 SELECT * INTO cfg FROM ocr_settings WHERE singleton AND enabled;
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
CREATE TRIGGER document_ocr_accepted AFTER INSERT ON documents FOR EACH ROW EXECUTE FUNCTION enqueue_document_ocr();

CREATE FUNCTION stop_removed_document_ocr() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.status IN ('Trash','Purged') AND OLD.status IS DISTINCT FROM NEW.status THEN
  UPDATE durable_jobs SET status='Cancelled',cancelled_at=now()
  WHERE id IN (SELECT job_id FROM document_ocr_runs WHERE document_id=NEW.id AND organization_id=NEW.organization_id) AND status='Queued';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER document_ocr_removed AFTER UPDATE OF status ON documents FOR EACH ROW EXECUTE FUNCTION stop_removed_document_ocr();
