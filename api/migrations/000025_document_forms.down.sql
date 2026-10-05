-- Refuse to erase human work. Application rollback should keep this additive schema.
DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM document_forms) OR EXISTS (SELECT 1 FROM document_form_history) THEN
  RAISE EXCEPTION 'Document forms contain saved data; retain migration 25 during application rollback';
 END IF;
END $$;
DROP TRIGGER document_form_invalidates_review ON document_forms;
DROP FUNCTION invalidate_document_form_review();
DROP TABLE document_form_history;
DROP TABLE document_forms;
ALTER TABLE document_extraction_reviews DROP COLUMN form_invalidated_at;
ALTER TABLE document_extraction_reviews DROP COLUMN form_revision;
