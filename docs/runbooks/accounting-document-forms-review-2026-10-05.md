# Accounting document forms: continuation review 2026-10-05

Candidate: `.scratch/inbox-release-20261004`, branch `feat/accounting-document-forms`, base `4f506401948f1ceb9b5711253ba28935b14d4172`. All implementation remains uncommitted. Main-tree unrelated changes were preserved.

## Standards review

1. **P1, fixed; regression checks pass:** initial canonical forms ignored human edits in legacy review drafts. `httpapi/document_form_routes.go` now overlays the eligible current-OCR draft, including explicit empty values, even when member reviewing has subsequently been disabled. `expected_legacy_draft_revision` travels through the Server Action and is checked under the document lock on first canonical save/confirm. Old legacy draft writes are rejected once canonical data exists; the existing old-confirm guard remains. Source: handoff requirement to preserve human additions and AGENTS review-workflow constraints.
2. **P2, code fixed; browser verification pending:** print ignored `source_superseded` and labelled old OCR confirmations as current. `print/page.tsx` now identifies the earlier OCR confirmation and requests a new review. Source: AGENTS acknowledgement/uncertainty constraints and design-system sections31/46.

## Spec review

1. **P1, fixed; regression checks pass:** entered total108 could evade discrepancy detection when calculated subtotal/VAT were left blank although complete rows implied107. `extraction/document_form.go` compares entered values with `CalculateForm` without changing raw evidence. Five cases cover mismatched total/VAT/base, matching amounts and unknown values. Requirement: keep original and calculated amounts separate to show differences.
2. **P2, fixed; real DB regression checks pass:** reference snapshots/allocation limits ignored a known calculated total when the entered total was blank. `postgres/document_forms.go` shares `referenceFormTotal`: entered value first, including zero, then saved assessment total; recalculation only for old records without an assessment. Three cases cover calculated107, entered108 and entered0, upper limits and unchanged original evidence. Requirement: avoid requiring duplicate entry of fully calculated totals; billing references must include source and allocated amounts.

## Validation

- `go test ./internal/extraction ./internal/httpapi`: PASS on latest code.
- Real disposable PostgreSQL18.4: `go test ./internal/postgres -run 'TestDocumentForms(PersistenceAndIsolation|ReadinessRequiresMigration25)' -count=1`: PASS. Includes all22 types, new legacy-conversion/revision cases, new calculated reference cases, original permissions/RLS/concurrency cases, and readiness migration enforcement.
- The legacy regression exposed missing integer casts in existing SaveDraft audit JSON; `review_drafts.go` casts old/new revisions explicitly. This path passes the real DB test.
- Full test initially exceeded the default30-document plan because additional fixtures were added. Test setup now grants a temporary Business trial only to its disposable organizations.
- The exceeded-quota path exposed an existing SQLSTATE42P08 in `documents.go`. Added `document_quota_test.go`:30 distinct intakes succeed,31st must return ErrQuota and persist rejection reason with90-day expiry. It failed before the fix and passed after adding the missing `$6::timestamptz` cast. Production quotas remain unchanged.
- Web lint: PASS. Web tests:31/31 PASS. `npm run build -- --webpack`: PASS, including TypeScript and detail/print routes. `git diff --check`: PASS.
- Earlier browser evidence remains historical. During this round, a36-row local form save failed because Next hot-reloaded the new payload field while the old fixture API still rejected unknown fields. It is not persistence proof. A new fixture compiled from final code started successfully, then ended PASS on shutdown. Final browser save/reload and print review did not occur.
- Automatic approval review rejected the local Next production-start command with `blocked by policy`; no additional reason was supplied. No alternate launch was used to bypass the rejection.
- No new PDF was exported/visually accepted this round. The earlier3-page PDF generation did not include page-by-page visual inspection.

## Open work and next checks

1. Resume the exact candidate; review the final tracked and untracked files. Avoid line-ending-only changes and unrelated main-tree issued-document work.
2. Include the one-line quota rejection SQL fix and its real DB regression in final diff review; this additional defect is now fixed and verified.
3. Restart disposable DB/API plus Next when permitted; verify final-code authenticated save/reload, first canonical legacy migration, stale OCR print warning. Generate36-row PDF from a fresh saved fixture and inspect every rendered page.
4. Preserve release gate: migration25 before API traffic, then CI and authenticated staging/production verification. No commit, push, migration outside disposable databases, deploy, or feature-flag change occurred.

## Shutdown

The latest fixture Go process ended PASS and performed cleanup. PostgreSQL was shut down with `pg_ctl -m fast -w stop`; the task-owned Next dev process was stopped. No listening process remained on3110,55439,55440 at checkpoint. Older interrupted-run disposable DB remnants may still exist as already noted in the previous handoff; none were deleted opportunistically.
