# Accounting field review release — 2026-10-03

Continuation of `accounting-review-ui-handoff-2026-10-03.md`. Source is the isolated `.scratch/classifier-release` checkout, branched from `cd22ec043818a4f4bf2793400a3a95c595fc95ab` onto `feat/accounting-field-review`. The dirty main checkout is preserved.

## Implemented

- Review uses shared labels/categories for all 19 recognized document types. It groups the 11 persisted extraction fields into document information, seller, buyer, amounts and tax; OCR line items and extra tax data retain their read-only contracts.
- Every editable field requires a human decision: accepted, corrected or unknown. Changing a value clears its decision. Empty optional fields require unknown, without inserting guessed values. New confirmations opt in with `field_review_version=1` and `expected_document_type`.
- API validates decisions against the exact OCR proposal/submitted value, preserves legacy clients, and retains Owner/Admin, tenant, CSRF, review acknowledgement, OCR/review revision and saved-draft gates. Confirmed JSON artifacts now keep decisions and the original normalized values alongside submitted values, OCR job, actor and time; database metadata supplies the committed review revision on reload. This requires no database migration and does not approve accounting.
- Linked error summary reveals the relevant form tab/mobile form view, focuses the field and shows inline errors. Values are controlled so a failed save keeps edits. Existing navigation/type-correction dirty guards remain.
- Desktop document/form proportions are 45/55; mobile retains the existing document/form switch and original/attachments viewer.
- Inbox upload is prominent and full width, with genuine one-file PDF/JPEG/PNG and 20 MiB limits, existing filters/pagination and gated integrations. The before-reading detail reuses the viewer and shows real backend status plus a manual refresh link. No artificial progress, bulk upload or lifecycle state was added.
- `/review-preview` is development-only mock data; preview actions do not persist and production calls `notFound()`.

## Verification

- Go 1.27.1: `go test ./...` and `go vet ./...` passed. Database integration cases requiring `OCR_TEST_ADMIN_URL` were not run locally.
- HTTP route tests: 17 opt-in/legacy cases and 9 saved-draft cases, including artifact persistence/reload, unknown/missing/invalid decisions, accepted-but-edited, stale OCR/type/review/draft revisions, unsaved fields/decisions, tenant/Member/CSRF/auth/acknowledgement denial, unavailable original and legacy compatibility.
- Web: 12 tests, lint, typecheck and production Webpack build passed with nonsecret build placeholders. `git diff --check` passed.
- Local development HTTP GET `/review-preview`: 200, all 11 decision controls and buyer/amount fields in HTML, with the mock-data notice. This proves rendering, not browser interaction or persistence.
- Read-only Railway flag refresh: classifier true only for `44f1ca09-d7d9-421c-80fd-4783098e62f2`, no LLM classifier URL, dispatcher false. No flags, OCR jobs, migrations or accounting approvals were changed.

## Browser verification

The initial missing-browser condition was resolved when the owner connected Chrome. Local desktop screenshot at width 1745 and narrow viewport (requested 375x812, observed innerWidth 341) showed no horizontal overflow. The review retained its document/form switch. Editing `TEST-0001` to `TEST-0002` cleared the accepted decision; an incomplete submit focused the error summary; clicking its total-amount link opened the amount tab and focused `extraction-total_amount`. After checking all 11 fields, submitting the mock returned its deliberate no-save error, focused the alert and retained `TEST-0002` and its corrected decision. The primary CTA was changed to accessible Primary 600 and the acknowledgement touch target to 44px after visual review.

These are local mock interaction checks, not database persistence evidence. The owner-authenticated live browser reached the three-organization selection screen. Real save/confirm/reload and genuine type correction still need the deployed code and an authorized pilot document checked against its original. No accounting posting is authorized by this test.

Push/deploy authorization from the handoff remains valid. Preserve classifier scope and the disabled dispatcher. Broad AI/accounting features, editable line items, WHT/VAT policies and multi-upload remain future work requiring their own contracts and representative real documents.

## Next steps

1. Local review interaction/visual checks above passed through supported browser APIs; viewport overrides were reset. No raw-CDP bypass was used.
2. Test a real authorized synthetic pilot document (or local authenticated fixture) against its original; do not overwrite the confirmed pre-receipt with invented values. Test draft gating when enabled, failed save, unknown values, stale conflicts, genuine type change and preserved edits/dirty warning.
3. Verify GitHub CI for this exact commit, make exact tracked API/web snapshots, then deploy API before web to staging Railway/API and production Vercel. Preserve scoped flags; no schema migration is needed.
4. Verify API SUCCESS/readiness, Vercel READY/alias and authenticated confirmation/reload independently; record exact IDs and results here. Roll back to the previous API/web images without deleting reviews or changing flags if needed.
