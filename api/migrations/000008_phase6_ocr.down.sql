-- Refuse destructive rollback while OCR history exists; operational rollback disables scheduling.
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM document_ocr_runs) THEN
  RAISE EXCEPTION 'Preserve OCR history: disable ocr_settings.enabled and drain workers instead';
 END IF;
END $$;
DROP TRIGGER document_ocr_removed ON documents;
DROP FUNCTION stop_removed_document_ocr();
DROP TRIGGER document_ocr_accepted ON documents;
DROP FUNCTION enqueue_document_ocr();
DROP TABLE document_ocr_runs;
DROP TABLE ocr_settings;
