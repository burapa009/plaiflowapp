-- Roll back the application flag first. This migration deliberately refuses to discard saved corrections.
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM document_review_draft_history LIMIT 1) OR EXISTS (SELECT 1 FROM document_review_drafts LIMIT 1)
        OR EXISTS (SELECT 1 FROM document_review_tasks LIMIT 1) OR EXISTS (SELECT 1 FROM document_review_returns LIMIT 1)
        OR EXISTS (SELECT 1 FROM document_export_snapshots LIMIT 1) THEN
        RAISE EXCEPTION 'phase 9 review data exists; retain additive schema';
    END IF;
END $$;
DROP TRIGGER document_review_values_purge ON documents;
DROP FUNCTION purge_document_review_values();
DROP TABLE document_review_draft_history;
DROP TABLE document_review_drafts;
DROP TABLE document_review_tasks;
DROP TABLE document_review_returns;
DROP TABLE document_export_snapshot_rows;
DROP TABLE document_export_snapshots;
DROP INDEX document_review_queue;
ALTER TABLE document_extraction_reviews DROP COLUMN draft_revision;
DROP POLICY document_extraction_read ON document_extraction_reviews;
CREATE POLICY document_extraction_read ON document_extraction_reviews FOR SELECT USING
    (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
      AND EXISTS (SELECT 1 FROM documents d WHERE d.organization_id=document_extraction_reviews.organization_id
        AND d.id=document_extraction_reviews.document_id AND d.status IN ('Available','Archived')));
DROP POLICY accounting_suggestions_read ON accounting_suggestions;
CREATE POLICY accounting_suggestions_read ON accounting_suggestions FOR SELECT USING
    (organization_id=app_organization_id() AND is_organization_member(app_user_id(),organization_id)
      AND EXISTS (SELECT 1 FROM documents d WHERE d.organization_id=accounting_suggestions.organization_id
        AND d.id=accounting_suggestions.document_id AND d.status IN ('Available','Archived')));
