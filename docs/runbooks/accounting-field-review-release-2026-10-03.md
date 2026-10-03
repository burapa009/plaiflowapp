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

These local mock checks are separate from the live persistence evidence recorded below. Genuine type correction and a persisted corrected-value case were not exercised on unrelated live documents. No accounting posting is authorized by this test.

Push/deploy authorization from the handoff remains valid. Preserve classifier scope and the disabled dispatcher. Broad AI/accounting features, editable line items, WHT/VAT policies and multi-upload remain future work requiring their own contracts and representative real documents.

## Next steps

1. Local review interaction/visual checks above passed through supported browser APIs; viewport overrides were reset. No raw-CDP bypass was used.
2. Test a real authorized synthetic pilot document (or local authenticated fixture) against its original; do not overwrite the confirmed pre-receipt with invented values. Test draft gating when enabled, failed save, unknown values, stale conflicts, genuine type change and preserved edits/dirty warning.
3. Verify GitHub CI for this exact commit, make exact tracked API/web snapshots, then deploy API before web to staging Railway/API and production Vercel. Preserve scoped flags; no schema migration is needed.
4. Verify API SUCCESS/readiness, Vercel READY/alias and authenticated confirmation/reload independently; record exact IDs and results here. Roll back to the previous API/web images without deleting reviews or changing flags if needed.

## Release result (2026-10-03 UTC)

- Owner explicitly approved push to `burapa009/plaiflowapp` / `feat/business-management` and Railway staging API / Vercel deployment after the initial automatic review rejected push for insufficient direct destination authorization. Push succeeded without force after that approval.
- Implementation commit `d73cfeaad6b7a8d00f4cde790e296f9739be5ca7`; CI `37134219448` passed web, Go, Python, migrations and secret checks.
- Railway staging API deployment `92575f10-0ca8-4755-9406-222f78dc0078` SUCCESS; image digest `sha256:7a806578d518330798d60220c933f7d415b579f3b9cb2d8c2a6a2e1f6074cdff`; `/readyz` returned 200 and ready. API snapshot SHA256 `FA51672C3A9A6948F3616E658CBCFB50FD67342B4FA8A0973C5882D91FCCF065`.
- First Vercel deployment `dpl_ACvmYpfzyT7jbfFZogS3tB3RDYSb` built with Turbopack and reached READY, but authenticated confirm failed twice (including a fresh reload) with `UnrecognizedActionError` for action `70bcb431126b9be0e2742650c609f81816f661d9fa`. Minimal unauthenticated POST with this Next-Action ID returned 404 / `x-nextjs-action-not-found: 1` / `Server action not found.`; no review was persisted and revision stayed 1. Do not roll back to this build.
- Build-only fix `2433622630917747b005c2f6ecc545b4a7064093` sets Vercel `buildCommand` to `npm run build -- --webpack`, matching validated local production builds. CI `37134829782` passed. Vercel `dpl_HBwNryVBMJC9xj5jcLL7fACVeDYC` READY, aliased to `https://plaiflowapp.vercel.app`. Same review scenario succeeded after this change. This is evidence for a bundler-specific build problem; the exact Turbopack manifest defect was not independently established. No artificial unit regression test was added for a deployment-only failure; live action + persistence checks exercise the actual seam.
- Owner-authenticated confirmation of authorized pilot document `1f23a9fc-1b44-48f6-8a70-4c7358b57db0` succeeded at `2026-10-03T15:54:33.848897Z`, redirecting to `?extraction=confirmed`. After independent reload, all 11 decisions remained (5 accepted, 6 unknown), values remained as checked against the original, and database review revision was 2 bound to OCR job `ca18ce81-4b0d-4428-83ba-516c06ea528e`. Empty/uncertain tax, branch, currency and subtotal/VAT values were not invented. The document remained pre-receipt. Local screenshot: `.scratch/field-review-confirmed-2433622.jpg`.
- Read-only post-check: schema version 24, dirty false; accounting suggestions for the pilot document remained zero; OCR job Completed / attempt 1; queue counts and classifier org scope / disabled dispatcher were preserved. No migrations or OCR retries were performed.
- Remaining coverage: representative documents across all 19 types, genuine type correction, live saved-draft gate (when enabled), and persisted corrected-value scenarios. Existing route tests cover conflict/auth/draft denial; mock browser covers edits and failed-save retention. These remaining scenarios are not claimed as live end-to-end proof or grounds to expand posting/AI automation.
