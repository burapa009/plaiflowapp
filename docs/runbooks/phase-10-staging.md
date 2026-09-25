# Phase 10 staging and rollback

Phase 10 remains disabled until its client authorization, Phase 9 dependency, database, and performance gates pass. The current Railway `staging` database also serves the public site; verify a restorable snapshot and the public routing before applying migration 13 or changing flags.

## Deploy order

1. Record API, worker, web, database deployment IDs; check `schema_migrations.version`, `dirty=false`, and a recoverable database snapshot.
2. Apply `000013_phase10_firm` on an isolated test database first. Run the PostgreSQL integration tests, including request, approve, accept, assign, foreign tenant denial, and immediate revoke. Check migration 13 with `dirty=false` and previous endpoints before deploying code.
3. Deploy API and web with `FIRM_ENABLED=false` and `NEXT_PUBLIC_FIRM_ENABLED=false`. Verify `/readyz`, existing auth/Drive/Document routes, logs, and metrics. The Drive Owner/Admin policy change is included in migration 13 and must be checked for client-only authorization and recent authentication.
4. Activate only when the feature's full security and performance proof is recorded. Phase 9-dependent firm review and export require their separate Phase 9 gates first. A passing build or deployment state is not activation evidence.

## Rollback

Set `FIRM_ENABLED=false` on the API and web and `NEXT_PUBLIC_FIRM_ENABLED=false` on the web, then restore the preceding API/web deployments. Confirm `/readyz`, direct client access, Drive, tasks, and existing export. Keep the additive tables and append-only audit history once a grant exists; the down migration refuses to delete grants. Revoke active grants through the client or firm relationship control if needed. Do not force a dirty migration version or drop grant history to roll back application code.

## Evidence still required

- PostgreSQL migration dry run and integration tests on a disposable database.
- Authenticated two-client tenant isolation and grant/assignment revoke races.
- Portfolio and queue P95 at 20 clients/5 staff, query count, CPU/RAM, and freshness.
- 50,000-row XLSX through the durable job path within 10 minutes and 128 MiB; authorization during generation and download, artifact expiry and deletion.
- Staging HTTP smoke, Railway/Vercel terminal deployment states, error logs, metrics, and rollback rehearsal.
