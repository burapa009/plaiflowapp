# Phase 13 company pilot status — 2026-09-28

## Scope

The first company pilot adds an in-app, per-user daily briefing of authorized Tasks and human-confirmed Routine Suggestions. Follow-up and meeting action labels come from human-entered Task details. The missing-documents intent reports that its source is unconfigured. Firm briefing, outbound reminders, external wording providers, and financial advice are outside this pilot.

The API and jobworker require `SECRETARY_ENABLED=true`; the web entry point requires `NEXT_PUBLIC_SECRETARY_ENABLED=true`. API and web also require the Organization ID in `SECRETARY_PILOT_ORGANIZATION_IDS` (comma-separated); an empty list denies every Organization. The flags are enabled only for the existing company pilot Organization `bb87fea6-ff54-4f1e-bf58-dcf117381cce`. No other Organization is enabled.

## Evidence and open gates

| Gate | Current evidence | Remaining action |
| --- | --- | --- |
| Local code | `8b3d610` passed clean-snapshot Go tests and server/jobworker build; the isolated PostgreSQL migration/behavior suite passed in 134 seconds. `8fba415` passed clean-snapshot web tests, lint, typecheck, and build; the unchanged web source also built on Vercel | GitHub Actions did not run for this branch; require CI for wider release |
| Migration and backup | Before migration, a private local `pg_dump -Fc` restored 2 Organizations, 7 Documents, 1 Task, and version 15 into a disposable database. Migrations 16–18 passed there, then applied to staging with clean version 18. A post-release dump restored version 18 and counts 2/7/1/1 (last count: Secretary Briefings) | Railway PITR remains disabled; establish managed, off-host recurring backup and RPO/RTO evidence for wider release |
| Rights | Isolated test covers two Organizations, Member/Owner role changes, source access, unseen-item revocation, and Membership deletion. Live unauthenticated pilot route returns 401, excluded Organization returns 404, and the browser showed 404 outside the allowlist | Non-owner runtime-role RLS matrix and live Owner/Member cross-tenant smoke |
| Briefing and reminders | Authenticated browser showed Queued then Ready with one existing undated Open Task, a correct reason, and a working Task deep link. Missing-documents intent explicitly says its source is unconfigured. Task count stayed 1; no Routine Template or Suggestion was fabricated. Isolated test confirms a due suggestion creates a Task only after human confirmation | Live routine confirm/skip, failure/refresh, Member view, and narrow viewport still need a controlled test |
| Operations | The live generation logged 1 candidate, 6,308 ms queue, 16 ms gather, 1 attempt, and zero provider tokens/cost. API `/readyz`, web home/pricing returned 200; no API 5xx in the latest 10-minute window | Representative cached/fresh p95 and sustained queue/error review; investigate the pre-existing `document_scan_bypassed` warning before broader release |
| Rollback | Flags-off behavior was verified before activation. Disable API/jobworker `SECRETARY_ENABLED` and web `NEXT_PUBLIC_SECRETARY_ENABLED`, then redeploy; restore prior app deployments if needed. Keep additive migration 18 after pilot data exists | Rehearse post-activation flag rollback and forward repair; do not run migration 18 down with pilot data |

The pre-migration backup is `.scratch/phase-13/backups/staging-pre-phase13-20260928T075526Z.dump` (SHA-256 `cd7305ae1c0f302ceced26272bfb1e78093c9864d80a17d87f41c1564a7e9717`). The post-release backup is `.scratch/phase-13/backups/staging-phase13-enabled-20260928T081630Z.dump` (SHA-256 `78651b7190881f0944c6edd123d3abd9b7bbdadd3aeaa6ae69a77142d4ee54eb`). Both files are outside Git in a directory restricted to the local Windows user. Each restored successfully into a disposable database on the same Postgres service, which was removed after verification. This proves logical restore, not off-host disaster recovery.

The earlier exposed staging database credential was rotated. Postgres, API, and worker variables were synchronized without printing the new value; all three redeployments succeeded, private-host authentication passed, and API `/readyz` returned 200. The existing registered staging SSH key was used; the public database URL remains absent.

## Release and rollback evidence

The branch `feat/business-management` was pushed through `8b3d610`. Railway API deployment `844e1889-24ab-4bf9-8e3f-b639673a3fde` reached `SUCCESS`; jobworker deployment `c9733398-f115-4d46-b655-7887d704ff13` reached `SUCCESS`. Vercel production deployment `dpl_E6uBjSsy1Pqxq48sgC1kEu4ydhCF` reached `READY` at the existing `plaiflowapp.vercel.app` alias. The web source is unchanged from `8fba415`; only the API needed the later `8b3d610` task-selection fix. The production Vercel alias still points to the staging API under the existing single-site topology.

At 2026-09-28 08:15–08:17 UTC, browser smoke used an existing authenticated Owner session. It observed the real Task in the briefing and opened its Task detail; the source had no due date and no assignee. The missing-document intent returned the unconfigured-source message. The excluded Organization route displayed 404. Jobworker completed the generation; no new Secretary error was observed after the transient `claim status 400` warnings during flag activation. The API also logged the existing `document_scan_bypassed` warning at startup. Neither a one-item pilot nor these smoke checks establish production readiness, representative latency, routine UI completion, or full tenant/RLS proof.
