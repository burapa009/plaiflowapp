# Accounting document forms live release — 2026-10-05

Released to https://plaiflowapp.vercel.app. User authorized push and real deployment. Implementation source: `d054226638bfec80fcb44fbe91f58c68264ca7c3`, branch `feat/accounting-document-forms`; draft PR https://github.com/burapa009/plaiflowapp/pull/2. Unrelated main-worktree changes were excluded. No branch merge was performed.

## Verification

- GitHub push CI https://github.com/burapa009/plaiflowapp/actions/runs/37328922209 and PR CI https://github.com/burapa009/plaiflowapp/actions/runs/37328930741 passed all jobs, including real PostgreSQL forms/readiness/quota checks.
- The live website uses Railway environment `staging` in project `3489cf3f-c0e6-4883-b5e1-6bc9a6b98fda`; no alternative production environment was introduced.
- Pre-migration custom pg_dump was encrypted with DPAPI CurrentUser, round-trip verified and retained at `.scratch/cutover-backups/plaiflow-preforms-20261005.dump.dpapi`. Plain archive SHA256: `764592b792ddc7eb95d1fb52b16a68be8f4bd93734c1bea9a3e8e87a4e43953f` (493184 bytes). Do not publish this backup.
- Live migration25 committed transactionally; schema_migrations reports version25, dirtyfalse. Both forms/history tables have RLS and FORCE RLS enabled.
- Railway API deployment `d9a5c017-d077-4d6a-a8cb-a84ba71e2ced` reached SUCCESS. Image digest: `sha256:4a936711e10e0b9633fc0fbe92825339538406c85819b1ece08f39fe706f7baf`. Container started at2026-10-05T15:18:34UTC. /readyz returned HTTP200 with statusready.
- Vercel production deployment `dpl_Buq8jX1wvaQuGEoXAJWHsUWX6HWp` is READY at https://plaiflowapp-6gjvt8lcs-burapa009s-projects.vercel.app. Built with Webpack from the exact Git archive; initial --skip-domain build was promoted only after API SUCCESS. Inspecting https://plaiflowapp.vercel.app resolves to this deployment.
- Authenticated live Server Action: existing public example `ใบกำกับภาษี_iTAX.jpg`, document `0321e06b-afbe-4b58-b684-82d6049ae1c3`, organization `bb87fea6-ff54-4f1e-bf58-dcf117381cce`: saved an unchanged draft, then reloaded. UI retained `บันทึกร่างแล้ว · ฉบับ1`, receipt type, original date2020-11-16, unknown values blank, and review acknowledgement unchecked. No confirmation or financial posting was performed. Browser error log was empty.
- Live saved-data print summary loads draft revision1, rule version2026-10-05.1, Thai date16พ.ย.2563, and explicitly says completeness/tax eligibility are not assessed. Browser layout inspected. Screenshots: `.scratch/release-evidence/document-forms-live-save-reload-20261005.jpg` and `document-forms-live-print-20261005.jpg` (local only).

## Build delay and retry

First API deployment `3bfe6b3b-f667-4aee-92a1-7ae37b5fdb18` ended FAILED during build after approximately10minutes initialization and1minute scheduling. Its complete available build log contained only scheduling on Metal builder `builder-jeiblb`; no Docker/Go failure was logged. Railway showed a contemporaneous incident about GitHub-triggered build delays (https://status.railway.com/incident/8RELVRFI), which is contextual evidence, not proof of the cause of the CLI-upload failure. One retry of the identical snapshot succeeded; code and environment configuration were unchanged.

## Rollback and remaining scope

If an application rollback is needed, retain additive migration25. The guarded down migration refuses to erase forms/history data. Previous API deployment: `92575f10-0ca8-4755-9406-222f78dc0078`; previous web: `dpl_6RnPpVuCiz9nL5zvjnPwCkGuk9es`.

Worker/jobworker and classification/OCR flags were not changed. Authorization and unsupported official issuance/posting/tax gates remain. This release proves authenticated live draft save/reload and saved-summary display; it does not prove live human confirmation of uncertain OCR fields or final per-page PDF visual acceptance. Those are distinct checks, and no accounting acknowledgement was invented for deployment testing.
