-- Deliberately refuse a destructive rollback after a pilot has stored work.
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM secretary_briefings)
       OR EXISTS (SELECT 1 FROM routine_templates)
       OR EXISTS (SELECT 1 FROM routine_suggestions)
       OR EXISTS (SELECT 1 FROM secretary_preferences)
       OR EXISTS (SELECT 1 FROM durable_jobs WHERE kind='secretary')
       OR EXISTS (SELECT 1 FROM tasks WHERE secretary_category<>'Task') THEN
        RAISE EXCEPTION 'Phase 13 contains data; disable the flag and use a forward repair';
    END IF;
END $$;

DROP TABLE routine_suggestions;
DROP TABLE routine_templates;
ALTER TABLE tasks DROP CONSTRAINT task_secretary_source_consistent;
ALTER TABLE tasks DROP COLUMN meeting_on;
ALTER TABLE tasks DROP COLUMN meeting_name;
ALTER TABLE tasks DROP COLUMN follow_up_with;
ALTER TABLE tasks DROP COLUMN secretary_category;
DROP TABLE secretary_preferences;
DROP TABLE secretary_briefings;
ALTER TABLE durable_jobs DROP CONSTRAINT durable_jobs_kind_check;
ALTER TABLE durable_jobs ADD CONSTRAINT durable_jobs_kind_check CHECK (kind IN ('export','ocr'));
