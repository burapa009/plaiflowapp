-- Roll back the application/feature flag, not these historical decisions.
-- Refuse a schema-version downgrade that would leave Phase 8 tables in place.
DO $$ BEGIN RAISE EXCEPTION 'Phase 8 schema is expand-only; set ACCOUNTING_ENABLED=false to roll back'; END $$;
