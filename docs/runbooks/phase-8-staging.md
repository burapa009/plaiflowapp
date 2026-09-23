# Phase 8 isolated staging release checklist

Status: implementation in progress; this is a procedure, not deployment evidence.

The existing Railway `staging` environment is connected to the public PlaiFlow site. Never apply migration 10 or enable `ACCOUNTING_ENABLED` there. Use a new, isolated Railway project/database/API and a separate protected Vercel staging project. Seed only disposable, synthetic documents. The user approved this separation and approved adding the new callback to the existing LINE test channel without removing its old callback.

1. Record new staging project/service/deployment IDs, domain, database version/dirty state, and an independently recoverable database snapshot. Keep production/public connection strings and object-store keys out of staging. Verify the separate DB and private storage.
2. Apply migrations 1–10 only to the new staging DB. Require clean version 10. Migration 10 is additive and the down migration intentionally refuses data-destructive rollback. Keep `ACCOUNTING_ENABLED=false` during initial deployment; enable only after `/readyz` and authorization checks pass. Both API and web need `ACCOUNTING_ENABLED=true` for the user-facing flow.
3. Use a staging-only LINE callback URL and test credentials. Add the callback alongside the existing test-channel URL; never replace the public callback. Protect the staging Vercel deployment.
4. Smoke a disposable Thai tax invoice through OCR, confirmed extraction, exact Vendor/branch or explicit unmatched reason, human category approval, deterministic Vendor-specific mapping, review queue, and exactly one generic `accounting_suggestions_v1` CSV/XLSX row. Verify warning/conflict, source revision change, formula-looking seller name, blank versus `0.00`, Member approval denial, and another Organization's read/write/export denial.
5. Check test/lint/typecheck/build, API `/readyz`, migration version, logs, request duration/P50/P95, export size, errors, slow queries, CPU/memory, and provider cost fixed at zero. The Phase 6 OCR benchmark has no human references and `acceptance_passed=false`; staging smoke does not authorize production activation.

Rollback: set `ACCOUNTING_ENABLED=false` on API and web, restore the previous isolated-staging deployments, and verify `/readyz`, Phase 7 extraction, and existing exports. Preserve additive version-10 tables and audit history. Do not run the Phase 8 down migration or delete suggestions/rules merely to revert application code. Investigate a dirty migration before any redeploy. Keep the public site untouched throughout.
