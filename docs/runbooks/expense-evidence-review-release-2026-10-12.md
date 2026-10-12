# Expense evidence review — 2026-10-12

Implementation of the first review-workflow slice from the independent PlaiFlow system design. Based on `db65b7e` in the existing isolated document-workflow checkout; unrelated root changes are excluded. User authorized implementation and the existing push/live-deployment workflow.

## Scope

- The real canonical editor shows original OCR values, available raw text and page/line references for scalar fields. It distinguishes missing values and user corrections; low-quality or ambiguous OCR remains a review concern.
- Evidence positions are used only when their OCR run matches the saved source snapshot. Older source values are labelled explicitly and never receive newer coordinates. Missing locations are disclosed; no bounding boxes are invented.
- A review summary shows active scalar fields filled from OCR, differences from OCR, and review issues. The issue list focuses the corresponding field or table, including controls in collapsed sections. Counts are not accuracy scores and do not include table-cell provenance.
- Populated optional sections open initially so extracted values are visible. All seven expense types retain their canonical layouts and hidden values. Type/field edits still reset both acknowledgements, and the existing Server Action, revision checks, tenant authorization and save endpoints remain in use.
- Uses PlaiFlow tokens and original code/layout; no Paypers source, assets, copied copy, or private-service reverse engineering. REA is a development analysis tool, not a production dependency.
- No API/OCR service, provider, migration, financial posting, billing or feature-gate change. This improves review usability; it does not establish a 10–20 second latency guarantee.

## Verification

- Web suite: 37 tests passed. Added source-preservation, cross-run isolation, invalid-location, uncertainty and seven-type active-field regressions. Lint, typecheck and Webpack production build passed.
- Local browser: all seven types retained an edited document number; changing type disabled confirmation after both acknowledgements had been checked. Issue navigation focused the second item quantity. Original/form switching at 375 px retained the edit. No document-level horizontal overflow at 375, 768 or 1440 px.
- Local preview uses production components with synthetic data and cannot persist. Real live save/reload verification is separate; physical phones and a new OCR latency benchmark were not tested.

## Release

- Implementation commit: `0d2b06b230dec281b079c72a3f3e8978255422a4`, pushed to `feat/expense-evidence-review`. Draft PR: https://github.com/burapa009/plaiflowapp/pull/5, based on `fix/ocr-form-latency`; no merge.
- All five CI jobs passed (web, Go, Python, migrations and secrets): https://github.com/burapa009/plaiflowapp/actions/runs/38204856973. Migration tests ran against CI's disposable database, not production.
- Deployed the exact committed `web/` archive with the existing project link and ignore rules. No local environment files, dependencies or `.next` were uploaded.
- Deployment `dpl_3oJVEG745vk29s2rFaFGxipck9St` reached READY at https://plaiflowapp-cin5t6wld-burapa009s-projects.vercel.app. Promoted only after CI passed; Vercel inspection confirmed https://plaiflowapp.vercel.app resolves to this deployment.
- Public smoke: homepage 200; local workflow route 404; review preview contains no mock editor and renders not-found content (streamed HTTP 200). Opening the known synthetic test document in the browser returned to sign-in because no authenticated session was available. Live draft save/reload remains unverified for this release; requested sign-in without blocking deployment.

Rollback: the previously verified production deployment is `dpl_GfsHNW6Cai41zgfnncWnuqciHdhQ`, https://plaiflowapp-fl42raoof-burapa009s-projects.vercel.app. Web-only rollback needs no schema or API changes.
