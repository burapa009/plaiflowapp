-- Additive storage for type-specific forms; OCR artifacts and old reviews remain intact.
ALTER TABLE document_extraction_reviews ADD COLUMN form_revision integer NOT NULL DEFAULT 0;
ALTER TABLE document_extraction_reviews ADD COLUMN form_invalidated_at timestamptz;
CREATE TABLE document_forms (
 organization_id uuid NOT NULL,
 document_id uuid NOT NULL,
 ocr_job_id uuid NOT NULL,
 revision integer NOT NULL CHECK (revision > 0),
 data jsonb NOT NULL CHECK (jsonb_typeof(data)='object'),
 status text NOT NULL CHECK (status IN ('Draft','Confirmed')),
 updated_by uuid NOT NULL,
 updated_at timestamptz NOT NULL,
 PRIMARY KEY (organization_id,document_id),
 FOREIGN KEY (organization_id,document_id) REFERENCES documents(organization_id,id),
 FOREIGN KEY (organization_id,updated_by) REFERENCES memberships(organization_id,user_id)
);
CREATE TABLE document_form_history (LIKE document_forms INCLUDING CONSTRAINTS);
ALTER TABLE document_form_history ADD PRIMARY KEY (organization_id,document_id,revision);
ALTER TABLE document_form_history ADD FOREIGN KEY (organization_id,document_id) REFERENCES documents(organization_id,id);
ALTER TABLE document_forms ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_forms FORCE ROW LEVEL SECURITY;
ALTER TABLE document_form_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_form_history FORCE ROW LEVEL SECURITY;
-- A permitted reviewer may invalidate a confirmation by editing, but cannot write reviews.
-- The trigger changes only the invalidation timestamp for this exact document.
CREATE FUNCTION invalidate_document_form_review() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,public AS $$
BEGIN
 IF NEW.status='Draft' THEN
  UPDATE public.document_extraction_reviews SET form_invalidated_at=NEW.updated_at
   WHERE organization_id=NEW.organization_id AND document_id=NEW.document_id AND superseded_at IS NULL;
 END IF;
 RETURN NEW;
END;
$$;
REVOKE ALL ON FUNCTION invalidate_document_form_review() FROM PUBLIC;
CREATE TRIGGER document_form_invalidates_review AFTER INSERT OR UPDATE ON document_forms
FOR EACH ROW EXECUTE FUNCTION invalidate_document_form_review();
CREATE POLICY document_forms_read ON document_forms FOR SELECT USING (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
 AND EXISTS (SELECT 1 FROM documents d WHERE d.organization_id=document_forms.organization_id AND d.id=document_forms.document_id
 AND d.status IN ('Available','Archived') AND (
  organization_role(app_user_id(),d.organization_id) IN ('Owner','Admin')
  OR (d.submitted_by_user_id=app_user_id() AND NOT d.group_restricted)
  OR d.assignee_user_id=app_user_id()
  OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id
   AND ds.document_id=d.id AND ds.submitted_by_user_id=app_user_id() AND NOT ds.group_source))));
CREATE POLICY document_forms_insert ON document_forms FOR INSERT WITH CHECK (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
 AND EXISTS (SELECT 1 FROM documents d WHERE d.organization_id=document_forms.organization_id AND d.id=document_forms.document_id
 AND d.status IN ('Available','Archived') AND (
  organization_role(app_user_id(),d.organization_id) IN ('Owner','Admin')
  OR (d.submitted_by_user_id=app_user_id() AND NOT d.group_restricted)
  OR d.assignee_user_id=app_user_id()
  OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id
   AND ds.document_id=d.id AND ds.submitted_by_user_id=app_user_id() AND NOT ds.group_source))) AND updated_by=app_user_id()
 AND (status='Draft' OR organization_role(app_user_id(),organization_id) IN ('Owner','Admin')));
CREATE POLICY document_forms_update ON document_forms FOR UPDATE USING (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
 AND EXISTS (SELECT 1 FROM documents d WHERE d.organization_id=document_forms.organization_id AND d.id=document_forms.document_id
 AND d.status IN ('Available','Archived') AND (
  organization_role(app_user_id(),d.organization_id) IN ('Owner','Admin')
  OR (d.submitted_by_user_id=app_user_id() AND NOT d.group_restricted)
  OR d.assignee_user_id=app_user_id()
  OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id
   AND ds.document_id=d.id AND ds.submitted_by_user_id=app_user_id() AND NOT ds.group_source)))) WITH CHECK (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
 AND EXISTS (SELECT 1 FROM documents d WHERE d.organization_id=document_forms.organization_id AND d.id=document_forms.document_id
 AND d.status IN ('Available','Archived') AND (
  organization_role(app_user_id(),d.organization_id) IN ('Owner','Admin')
  OR (d.submitted_by_user_id=app_user_id() AND NOT d.group_restricted)
  OR d.assignee_user_id=app_user_id()
  OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id
   AND ds.document_id=d.id AND ds.submitted_by_user_id=app_user_id() AND NOT ds.group_source))) AND updated_by=app_user_id()
 AND (status='Draft' OR organization_role(app_user_id(),organization_id) IN ('Owner','Admin')));
CREATE POLICY document_form_history_read ON document_form_history FOR SELECT USING (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
 AND EXISTS (SELECT 1 FROM documents d WHERE d.organization_id=document_form_history.organization_id AND d.id=document_form_history.document_id
 AND d.status IN ('Available','Archived') AND (
  organization_role(app_user_id(),d.organization_id) IN ('Owner','Admin')
  OR (d.submitted_by_user_id=app_user_id() AND NOT d.group_restricted)
  OR d.assignee_user_id=app_user_id()
  OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id
   AND ds.document_id=d.id AND ds.submitted_by_user_id=app_user_id() AND NOT ds.group_source))));
CREATE POLICY document_form_history_insert ON document_form_history FOR INSERT WITH CHECK (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
 AND EXISTS (SELECT 1 FROM documents d WHERE d.organization_id=document_form_history.organization_id AND d.id=document_form_history.document_id
 AND d.status IN ('Available','Archived') AND (
  organization_role(app_user_id(),d.organization_id) IN ('Owner','Admin')
  OR (d.submitted_by_user_id=app_user_id() AND NOT d.group_restricted)
  OR d.assignee_user_id=app_user_id()
  OR EXISTS (SELECT 1 FROM document_sources ds WHERE ds.organization_id=d.organization_id
   AND ds.document_id=d.id AND ds.submitted_by_user_id=app_user_id() AND NOT ds.group_source))) AND updated_by=app_user_id()
 AND (status='Draft' OR organization_role(app_user_id(),organization_id) IN ('Owner','Admin')));
