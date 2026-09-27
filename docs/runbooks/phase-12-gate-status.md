# Phase 12 gate status — 2026-09-27

Release candidate: **none**. Baseline branch `feat/business-management` at `57442b2` plus an uncommitted worktree. This is a work log, not a signed release decision. Public billing must remain disabled.

| Gate | Status | Evidence available now | Required before pass |
| --- | --- | --- | --- |
| Security | Open | Local API role/session tests added; RLS policy correction prepared in migration 16 | Isolated PostgreSQL non-owner runtime-role proof, cross-tenant route/worker matrix, LINE/upload drills, deployed role and credential inventory, finding retests |
| Billing/payment security | Open | Existing Omise test-mode Starter monthly success/failure; local webhook rotation regression | Staging old/new-secret drill, forged/replay/duplicate and cross-tenant test-mode matrix, Owner-only receipt/history, seller/accountant confirmation |
| Subscription/reconciliation | Open | Local aged-charge scanner and lag log signal prepared; [failure/grace procedure](./billing-failure-and-grace.md) | Monthly/six-month/yearly boundaries and renewal, cancellation/upgrade policy, delivered 15-minute alert with named owner, provider outage and recovery drill |
| Entitlement/usage | Open | Existing Plan gate and per-Organization intake lock | Paid monthly allowance/carry ledger, transition and concurrency tests, direct API/worker bypass proof |
| Export security | Open | Local CSV/XLSX sanitization and late-error regression prepared | Current-rights artifact download tests, staging files opened and inspected, cross-tenant/revoked-link matrix, large-export and object-store checks |
| Performance | Open | Existing request duration logs | Agreed numeric thresholds, reproducible p95/load/Web Vitals/queue and cost baselines |
| Reliability | Open | Local Go/web/Python checks passed; Railway staging services displayed Online with a `/readyz` 200 log entry | CI on final commit, migration rehearsal, deployed critical user-flow smoke, delivered alerts, a production checkout pause that keeps reads/reconciliation, failed-deploy and rollback exercise |
| Backup/restore | Open | No observed restore drill | Isolated restore with objects, tenant/receipt/payment reconciliation and measured RPO/RTO |
| Privacy | Open | Policy work exists under `docs/legal/` | Approved data map, retention and exercised export/deletion request |

Railway staging read-only inspection on 2026-09-27 showed API, worker, jobworker, Postgres, and two staging buckets Online. A 15-minute log search for `error` found no matching log entries; this is not post-release monitoring proof. The Observability page showed no configured blocks. The Railway production environment showed only a failed `plaiflowapp` service and no separate working API/database. The public Pricing page still said payment was closed. The direct staging URL was blocked by the browser client and unavailable from the shell, so no new critical-flow smoke was recorded.

Local PostgreSQL integration tests that depend on `OCR_TEST_ADMIN_URL` and `OCR_TEST_RUNTIME_ROLE` have not run. No Phase 12 migration has been applied to staging; no current worktree build has been deployed. Do not count existing `/readyz`, prior test payments, or a green local build as a pass for another gate.

For each eventual gate pass, record the exact commit/build, environment, test data and timestamp, result, evidence link, owner, residual risk, and rollback action. Resolve the open decisions in [Phase 12 decisions](../phase-12-decisions.md) and all [tickets](../phase-12-ticket-breakdown.md) before the [public-sale gate](../phase-12-ticket-breakdown.md) is signed.
