DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM document_classifications) THEN
    RAISE EXCEPTION 'Preserve document classification and user corrections';
  END IF;
END $$;
DROP TABLE document_classifications;
