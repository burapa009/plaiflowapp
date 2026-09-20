# Staging deployment and rollback

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
