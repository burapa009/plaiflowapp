DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM durable_jobs) OR EXISTS (SELECT 1 FROM durable_job_artifacts) THEN
        RAISE EXCEPTION 'phase 5 rollback refused: durable job data exists';
    END IF;
END
$$;

DROP TABLE durable_job_worker_heartbeats;
DROP TABLE durable_job_artifacts;
DROP TABLE durable_jobs;
