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

- Implementation commit: `5543c20cc7b8f07635fb77b05e9b3264c382ef77`; pushed to `fix/ocr-form-latency`. Draft PR: https://github.com/burapa009/plaiflowapp/pull/4. No merge.
- All five CI jobs passed: https://github.com/burapa009/plaiflowapp/actions/runs/38064828542. The Python CI job covers the export service; the 22 OCR tests were run separately on the installed Paddle environment.
- API and OCR were uploaded from the exact Git archive of this commit, without local environment files or unrelated changes. Both reached SUCCESS. API `/readyz` returned 200; OCR logs confirmed model initialization and private `/health` 200.
- API deployment: https://railway.com/project/3489cf3f-c0e6-4883-b5e1-6bc9a6b98fda/service/8a397a6a-760d-4ab1-8132-a122bcf869fc?id=55a33989-59a3-44aa-a0dd-ef52c8c77583
- OCR deployment: https://railway.com/project/3489cf3f-c0e6-4883-b5e1-6bc9a6b98fda/service/5cbff34a-155e-4f02-86a8-3717f8a113a1?id=7119770e-d69e-41a6-a46b-274fb632a704
- Vercel deployment `dpl_GfsHNW6Cai41zgfnncWnuqciHdhQ` reached READY and was promoted. Inspecting https://plaiflowapp.vercel.app resolves to https://plaiflowapp-fl42raoof-burapa009s-projects.vercel.app. The first upload attempt was rejected as unauthorized; account verification refreshed the existing session and the retry with the verified team succeeded.

## Authenticated live acceptance

- The user selected the known synthetic fixture after the extension's file chooser reported unavailable file access. No browser permission bypass was used. The agent sent it through the normal authenticated upload UI.
- Document: `345262c0-3394-406b-b3c5-8739da1e2723`, file `pilot-first-page.png`, 21,367 bytes. Stored SHA-256 matched the tested fixture: `2c62f01dc80fd790f0f139c7eccac7ab29e318871b51e6e2b3e9d06366c19be6`.
- First attempt completed. Server receipt to OCR completion: **5.338 s**; queue **0.909 s**; worker **2.696 s**; inference plus transfer **2,059 ms**. This is a clear, single-page synthetic receipt on the existing live CPU service, not an accuracy or latency guarantee for other inputs.
- The document list changed to ready-for-review without manual reload. Opening the new row displayed the receipt form with `TEST-0001`, `2026-09-28`, and total `1070.00`; absent parties and payment details remained blank. Browser automation's navigation/observation delays mean no exact upload-click-to-form-visible latency is claimed.
- Saved the unchanged draft through the normal UI. A fresh page reload retained revision 1, document number, date, and total. Both review acknowledgements remained unchecked; no confirmation or financial posting was performed.
- Screenshot saved locally as `ocr-live-saved-receipt.png` in this task's visualization directory. User-facing proof: https://plaiflowapp.vercel.app/o/bb87fea6-ff54-4f1e-bf58-dcf117381cce/documents/345262c0-3394-406b-b3c5-8739da1e2723

Live acceptance covers the document list's automatic update and persisted prefilled draft. Pending-detail polling has component-test coverage; the small live fixture completed before that view was opened. Existing completed OCR results were not mass-reprocessed.

Do not claim every document finishes in 20 seconds. Blurred images, orientation retries, many pages, and queued work can take longer. Original user template recognition is not fully corrected.

Rollback: redeploy the previous successful API and OCR releases from the service deployment history (API October 5, OCR preceding this release). Prior web is `dpl_55VVbh3X2T9R8WXxhAr7S4egZfTR`, https://plaiflowapp-q9is0c5zk-burapa009s-projects.vercel.app. No schema rollback is needed.
