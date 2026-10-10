# OCR to form latency fix — 2026-10-10

Isolated branch `fix/ocr-form-latency`, based on `bcce3cf` from the deployed document workflow. User authorized continuing the real OCR/form fix and the existing live release workflow. No unrelated main-worktree changes are included.

## Change and scope

- CPU OCR workers check an idle queue every 1–2 seconds instead of backing off to 15–16 seconds. Remote-provider and error backoffs remain unchanged.
- Paddle reads upright lines first; only low-confidence output gets one orientation retry, retaining whichever result has higher mean recognition confidence. Confidence is not a measured accuracy percentage. Existing input limits, model weights, coordinates, timeouts, and process isolation remain.
- The shared extraction path fills missing review fields from the stricter accounting parser, recognizes the combined receipt/tax-invoice form and a missing tone mark in the document-number label, and converts explicit head-office labels to branch code 00000. Conflicting or uncertain values stay unfilled; raw OCR remains available.
- Processing pages refresh every two seconds while visible. Completed editable forms do not poll. The manual status button now refreshes server data. An unknown document type asks the user to choose a type and use a clearer source image.
- No database migration, provider switch, hardware change, RunPod dispatcher activation, historical job replay, accounting confirmation, or posting.

## Evidence before release

- Live baseline for the user's 425×470 public blank template: receipt to OCR result 23.168 s, queue 6.575 s, worker 14.966 s. The page stayed on its old processing state until a real reload.
- Warm local OCR of the same file after the change: 8.219 s. Its title is still imperfect; missing template values must remain blank. This is inference timing on local hardware, not live end-to-end timing.
- Warm local OCR of a clear synthetic Thai receipt: 1.328 s. Its real OCR output passed extraction-to-form assertions for receipt, TEST-0001, 2026-09-28, subtotal 1000.00, VAT 70.00, and total 1070.00.
- Python OCR suite: 22 passed. Web suite: 34 passed; lint, typecheck, Webpack production build passed. Full Go tests and vet passed; targeted extraction suite passed again after adding head-office normalization.
- New regressions cover idle latency, orientation fallback, missing-to-form mapping, combined type, conflict/blank preservation, label tone mark, branch-code validity, refresh timing, visibility, cleanup, and manual refresh.

## Release and acceptance

Pending: exact committed archives for API, OCR, and web; healthy deployments; authenticated synthetic upload, automatic form appearance, measured live timing, and unchanged draft save/reload.

Do not claim every document finishes in 20 seconds. Blurred images, orientation retries, many pages, and queued work can take longer. Original user template recognition is not fully corrected.

Rollback: redeploy the prior API and OCR service releases and promote the prior Vercel deployment recorded in `document-workflow-release-2026-10-09.md`. No schema rollback is needed.
