# Phase 13 company pilot status — 2026-09-28

## Scope

The first company pilot adds an in-app, per-user daily briefing of authorized Tasks and human-confirmed Routine Suggestions. Follow-up and meeting action labels come from human-entered Task details. The missing-documents intent reports that its source is unconfigured. Firm briefing, outbound reminders, external wording providers, and financial advice are outside this pilot.

The API, jobworker, and web entry point require `SECRETARY_ENABLED=true` and `NEXT_PUBLIC_SECRETARY_ENABLED=true` respectively. API and web also require the Organization ID in `SECRETARY_PILOT_ORGANIZATION_IDS` (comma-separated); an empty list denies every Organization. Both flags stay off in staging until this gate is complete. The worker claims Secretary jobs only while its flag is on.

## Evidence and open gates

| Gate | Current evidence | Remaining action |
| --- | --- | --- |
| Local code | `go test ./...`, API binary build, `npm test`, `npm run lint`, `npm run typecheck`, and `npm run build` passed locally on 2026-09-28 | Re-run on the exact release commit in CI |
| Migration | An earlier Migration 18 up/down/up and company behavior candidate passed on a disposable database through the staging PostgreSQL tunnel; the final cascade and count changes have **not** had a database rerun because the tunnel lost SSH authentication. No live staging schema changed | Restore test access, then restorable live backup, isolated restore, and rehearse 16–18 before applying live |
| Rights | Earlier isolated test exercised two organizations, Member versus Owner work, ownership transfer, source read/edit, and human confirmation; route test checks session, allowlist, and feature flag | Re-run final database candidate, non-owner runtime-role RLS matrix, and deployed cross-tenant smoke |
| Briefing and reminders | Earlier isolated test checked same-day job reuse, overdue priority, unconfigured document source, and routine confirmation creates one Task | Re-run final database candidate, then authenticated browser smoke for queued/ready/failed, refresh, source links, narrow viewport, and real-time latency |
| Operations | Generation logs candidate count, queue and gathering milliseconds, attempts, zero provider tokens/cost; no source text is logged | Staging queue/retry/error review, cached and fresh p95, and observed cost on representative load |
| Rollback | Flags off hides all public routes and stops Secretary claims; migration 18 down succeeds only while pilot tables and jobs contain no data | Rehearse flag rollback on deployed build; after pilot data exists use a forward repair and retained backup |

Read-only staging checks on 2026-09-28 found migration version **15** and PITR `enabled: false`; no current restorable backup has been verified. Do not run migrations 16–18 on the live staging database or enable the pilot until backup and restore proof exists. `/readyz` and a local build alone do not prove the user flow. Preserve the existing single-site Railway/Vercel topology.

The 2026-09-28 final database rerun was interrupted when the Railway SSH tunnel closed and a new tunnel could not authenticate with the previously configured key. The public database URL is absent. Tunnel diagnostic files were removed after the CLI wrote a database credential to them; rotate that staging credential before further external access.

## Activation sequence

1. Identify named test Organizations and a representative workload. Keep billing, firm, and Secretary flags off.
2. Capture a current staging database backup outside the live volume. Restore it to a disposable database and verify tenant and document counts; rehearse migrations 16–18 there.
3. Apply migrations 16–18 to staging in order. Deploy the exact API, jobworker, and web commit with Secretary flags off. Check service health, routes hidden, and unrelated flows.
4. Enable the API and jobworker flag for a controlled pilot, then the web flag. Check a real Owner and Member briefing, refresh, follow-up and meeting intents, routine suggestion confirmation, cross-tenant denial, and the source deep links.
5. Measure p95 cached read and fresh briefing, queue delay, retry/error rate, candidate count, and provider cost. Review logs for serious errors and source text leakage. If any gate fails, disable both flags and stop Secretary claims.

No staging deployment, live migration, pilot activation, or rollback exercise is recorded by this document until evidence is added here with commit, timestamp, and observed result.
