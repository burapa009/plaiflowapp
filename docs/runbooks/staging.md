# Staging deployment and rollback

## Phase 7 release gate (not yet executed)

`EXTRACTION_ENABLED` defaults to `false`. Do not enable it or apply migration 9 to the current Railway `staging` database while the production Vercel site still points to that API. First isolate the public site from staging, or obtain an explicit maintenance/cutover decision with a recoverable database snapshot.

After isolation: record API/web deployment IDs and migration version/dirty state; take and verify a restorable database snapshot. Apply the additive `000009_phase7_extraction` migration, confirm version 9 and `dirty=false`, deploy API/web with the flag still off, then enable it only for staging. Smoke one disposable Thai tax invoice through completed OCR, raw/normalized review, a low-confidence warning, explicit human confirmation, and exactly one CSV and XLSX row. Check another Organization cannot read/confirm/export it; check spreadsheet formula escaping, request duration, export size, application errors, DB slow queries, and service CPU/memory. Synchronous export rejects more than 100 reviewed rows; larger exports require a durable job. The extractor has zero external provider calls; its provider cost is logged as zero, while compute cost is explicitly marked `not_metered`.

Rollback: set `EXTRACTION_ENABLED=false` and restore the prior API/web deployments. Leave the additive version-9 table and encrypted review artifacts in place if any review has been saved. Only after proving the table empty, taking a fresh snapshot, and confirming artifact cleanup should `migrate down 1` be considered. Recheck `/readyz`, OCR reads, document access, and existing exports.

## Required separation

- Use staging-only Railway project/services, PostgreSQL role/database, LINE test channel, and Vercel project/environment.
- PostgreSQL must use a private or TLS connection. Do not copy production values.
- Set Railway server/worker variables from `api/.env.example`; set Vercel server variables from `web/.env.example`.
- Enable Vercel Deployment Protection before sending synthetic webhook data.
- `DASHBOARD_API_TOKEN` may contain `current,previous` on Go during rotation; Vercel uses only `current`.

## Deploy order

1. Record the current API, worker, and web deployment identifiers.
2. Run `migrate ... up` as a one-off pre-deploy command. Stop if it fails or reports a dirty version.
3. Deploy `api` with `api/railway.server.toml`; wait for `/readyz` to return `200`.
4. Deploy the worker with `api/railway.worker.toml`; confirm a fresh heartbeat.
5. Deploy `web/` to Vercel with `API_BASE_URL=https://...` and the server-only token.
6. Run `scripts/smoke-staging.ps1` with synthetic data, then inspect request duration/error logs and platform CPU/memory metrics.

## Phase 5 durable jobs

1. Record the API, legacy worker, web and database deployment IDs plus a recoverable database snapshot. Confirm migration state is version 6 and clean.
2. Create a staging-only private Railway Bucket in `sin`. Configure the API with its `EXPORT_BUCKET`, `EXPORT_REGION`, `EXPORT_ENDPOINT`, `EXPORT_ACCESS_KEY_ID`, `EXPORT_SECRET_ACCESS_KEY`, a separate base64 32-byte `EXPORT_ENCRYPTION_KEY`, and `EXPORT_PATH_STYLE`. Add the same base64 32-byte `JOB_WORKER_AUTH_KEY` to the API and the new Durable Job worker. Add a separate base64 32-byte `EXPORT_DOWNLOAD_SIGNING_KEY` only to the API. Set only `APP_ENV`, `JOB_API_URL`, `JOB_WORKER_ID` and `JOB_WORKER_AUTH_KEY` on that worker; do not set `DATABASE_URL`, either export key or object-store credentials.
3. Apply migration `000007_phase5_durable_jobs` before deploying the API. Confirm version 7, `dirty=false`, migrated active export counts match, and `/readyz` returns 200.
4. Deploy the API, then deploy the new worker with `api/railway.jobworker.toml`. Keep the legacy worker running for inbound/LINE/document queues; it must not process Durable Jobs.
5. Claim two synthetic jobs concurrently and verify distinct IDs, renew one lease, reclaim one expired lease, and confirm a stale completion returns conflict. Run one export above 5,000 rows and verify Completed status, formula-safe CSV, tenant-scoped 15-minute download and 24-hour artifact expiry.
6. Check `/v1/dashboard` job queue wait, retry, stale reclaim, terminal failure, export row/byte and heartbeat-age metrics. Inspect logs for sustained errors, secrets, hot polling, slow queries and memory pressure.

Rollback: stop the Durable Job worker first, restore the previous API and web deployments, and leave the additive version-7 tables in place if any Durable Job or artifact exists. Existing `export_jobs` history remains readable. Only when all three new tables are empty, after a fresh snapshot, may `migrate down 1` remove version 7. Re-run `/readyz`, legacy worker heartbeat and export history checks.

Phase 3 requires schema version `5`; `/readyz` intentionally remains unavailable on older schema. Migrations `000004` and `000005` are additive. Before applying them, record the current migration version and take the provider's point-in-time backup/snapshot. Verify `version=5` and `dirty=false` before deploying the API.

Publish the Rich Menu only to the staging LINE channel after the web preview URL is stable. Record the previous default Rich Menu ID, then run `scripts/build-rich-menu.ps1` and `go run ./cmd/richmenu` from `api/` with staging-only `LINE_CHANNEL_ACCESS_TOKEN`, `WEB_BASE_URL`, and `RICH_MENU_IMAGE=../web/public/rich-menu.png`.

The Go JSON logs expose `duration_ms` for HTTP requests, `slow_query` above 200 ms, and `queue_age_high` above 60 seconds. Use platform histograms to verify webhook p95 below 500 ms and dashboard API p95 below 300 ms.

## Rollback

1. Stop new traffic by disabling the LINE webhook or restoring the previous server deployment.
2. Restore the previous worker and web deployments.
3. If the release has written no data requiring the new schema, run `migrate ... down 1`; otherwise leave the additive schema in place and roll application code back only.
4. Re-run `/readyz`, forged-signature, signed webhook, worker-state, and dashboard smoke checks.
5. If migration state is dirty, do not force a version blindly; inspect the applied SQL and repair it in a transaction first.

For Phase 3, prefer application rollback while leaving the additive version-5 schema in place. If and only if no Phase 3 Plan, Business Contact, Import Preview, Drive Connection, or reconnect-task data must be retained, run `migrate ... down 2` to remove `000005` then `000004`. Restore the previous default Rich Menu ID separately; deleting the new menu is optional and must happen only after the old default is active.

Migration `000001` is reversible but its down migration deletes all Phase 0 event data. CI tests down/up only against an isolated disposable database.

## Phase 4 document release gate

This section is a procedure, not deployment evidence. Do not release the document routes to staging until the remaining Phase 4 acceptance items are implemented and the checks below have recorded results.

1. Create a staging-only private object bucket and a private ClamAV `clamd` service in the Singapore Railway region. Keep production resources separate. Confirm that the bucket credentials report their API region and endpoint; do not assume the deployment region is the S3 API region.
2. Configure API and worker with the same `DOCUMENT_BUCKET`, `DOCUMENT_REGION`, `DOCUMENT_ENDPOINT`, `DOCUMENT_ACCESS_KEY_ID`, `DOCUMENT_SECRET_ACCESS_KEY`, `DOCUMENT_ENCRYPTION_KEY` (base64-encoded 32-byte key), `DOCUMENT_PATH_STYLE`, and `CLAMD_ADDR`. Keep credentials server-side. The API and worker reject staging startup if storage or scanner configuration is missing.
3. Confirm private bucket access and `clamd` connectivity from both services using a disposable, non-sensitive file. Confirm encrypted bytes in the bucket and that rejection removes quarantine bytes. Do not use a production document for this check.
4. Record deployment IDs, current database migration version, and a recoverable database snapshot. Apply `000006_phase4_documents` before deploying the new API. Require `version=6`, `dirty=false`, and `/readyz=200` before starting the new worker or web release. The migration has not been proven on this checkout without PostgreSQL.
5. Deploy API, worker, then web. Run the existing `scripts/smoke-staging.ps1`, followed by one authorized PDF/JPEG/PNG from Web, LINE direct, LINE group, and a selected Drive revision. Check Organization isolation, original retrieval permissions, Drive provenance, repeated webhook/revision idempotency, invalid-file rejection, quota behavior, status actions, filtered CSV/XLSX export, and revoked Drive access. Use a staging account and disposable files.
6. Record webhook response p95, document intake duration/error counts by channel and rejection code, scanner failures, Drive rate-limit/reconnect errors, worker queue age, bucket failures, API/worker CPU and memory, and database slow queries. Investigate any sustained error or backlog before calling the release healthy.

Rollback: restore the previous web, worker, and API deployments together. Leave the additive version-6 schema in place once any Document, Source, Usage, or Attempt row exists; its down migration drops those tables and loses data. If no Phase 4 row has ever been written, verify that fact against the staging database and take a fresh snapshot before considering `down 1`. Re-run readiness, webhook, worker, and authenticated document-access checks after rollback. Bucket originals need separate inventory and retention review; a database rollback does not delete them.

## Retention residual risk

Payload is cleared after 30 days and terminal metadata is deleted after 90 days. A provider replay after the 90-day idempotency window can create a new event; the accepted mitigation is the staging-only data boundary and no outbound Phase 0 side effect.
