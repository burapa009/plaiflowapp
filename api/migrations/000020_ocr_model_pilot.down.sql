-- Keep model provenance and completed artifacts. Roll back by removing the
-- operator's pilot override after draining its jobs, then restoring the image.
DO $$ BEGIN RAISE EXCEPTION 'Drain pilot jobs and remove model override; preserve OCR model history'; END $$;
