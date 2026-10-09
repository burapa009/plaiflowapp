# Document workflow release — 2026-10-09

User authorized push and live deployment, and selected applying prototype A to real document pages.

The isolated candidate starts from `c2fe2f3` / `origin/feat/accounting-document-forms`, the source of the currently deployed document forms. Main-worktree changes are excluded.

## Change

- The actual document list now shows intake, OCR processing, human review, and confirmation as separate stages. Status is loaded through the existing authenticated OCR/form APIs. A failed read shows an unavailable state rather than pretending OCR succeeded.
- Counts and stage filtering are explicitly limited to the current page of up to 20 documents. Server-side filename/source/date/file-status filtering and pagination remain available.
- The expense picker offers seven supported types; an older document's existing type remains available so opening it does not convert its data. The existing canonical field/rule configuration, source values, saved drafts, revisions, and backend authorizations remain in use.
- Confirmation in the web editor requires both the amount/uncertainty check and the full-document check. Any edit or type change resets both. The Server Action also checks both acknowledgements; the API still enforces its existing authorization and acknowledgement requirements.
- Mobile rows become cards; original/form switching keeps both panels mounted. Desktop retains the original beside the form. Upload, download, attachments, and existing feature-gated export actions remain connected to their existing endpoints.
- No API release, database migration, OCR feature flag change, payment execution, accounting posting, or tax assessment is part of this release.

## Local verification

- Web tests: 33 passed, including workflow classification, stale OCR confirmation, type choices, canonical form rules, CSRF upload checks, and cancelled-exit guards.
- Lint, typecheck, and Webpack production build passed.
- Browser checks covered 375 px and 1440 px layouts with no horizontal page overflow, stage filtering, upload expansion, changing type, retained values, both acknowledgement gates, and mounted original/form switching.
- Development-only fixtures use the production components at `/local-preview/workflow` and `/review-preview`. They use simulated data and cannot save real data. The local-preview route is excluded from the deployment; review-preview is guarded by `notFound()` outside development.
- Physical phones and authenticated live persistence are separate checks.

## Deployment

- Pushed `feat/document-workflow`; implementation commit `0a01eb1510afe29a9ad36e44756a06a90e156113`. Draft PR: https://github.com/burapa009/plaiflowapp/pull/3, based on `feat/accounting-document-forms`. No branch merge was performed.
- CI run https://github.com/burapa009/plaiflowapp/actions/runs/37955295917 passed all five jobs: Go, Python, migrations, web, and secrets. The first scan flagged a public Railway deployment UUID in the prior release runbook. Commit `8600a17` adds only that historical finding's exact fingerprint to `.gitleaksignore`; the full-history scan then passed locally and in CI.
- The deployed web source is the exact `web/` Git archive from `0a01eb1`, with the existing Vercel project link. No local environment file, dependencies, or unrelated main-worktree changes were uploaded. Later scan/runbook commits do not change web source.
- Vercel deployment `dpl_55VVbh3X2T9R8WXxhAr7S4egZfTR` reached READY at https://plaiflowapp-q9is0c5zk-burapa009s-projects.vercel.app. It was built with production settings and `--skip-domain`, then promoted after CI passed. Inspecting https://plaiflowapp.vercel.app resolves to this deployment.
- Public smoke checks: the landing page renders; unauthenticated `/organizations` returns to sign-in; `/local-preview/workflow` returns 404. `/review-preview` renders the not-found page with no mock editor (its streamed response uses HTTP 200).
- Authenticated live upload/OCR/draft save/reload was not verified in this release: no signed-in session was available. Google login returned `invalid_client` with the pre-existing staging placeholder configuration; the LINE QR login expired without a completed sign-in. Local fixture verification must not be described as persisted live API evidence.

Rollback target: previous web deployment `dpl_Buq8jX1wvaQuGEoXAJWHsUWX6HWp` / https://plaiflowapp-6gjvt8lcs-burapa009s-projects.vercel.app. No database or API rollback is required for this web-only release.

Status reads are currently at most two existing API reads per document; a batch endpoint should precede increasing the 20-document page size.
