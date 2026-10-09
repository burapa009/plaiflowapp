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
- Development-only fixtures use the production components at `/local-preview/workflow` and `/review-preview`. They use simulated data, cannot save real data, and are not deployed.
- Physical phones and authenticated live persistence are separate checks.

## Deployment

Pending push, CI, Vercel production deployment, and live verification. Previous web deployment: `dpl_Buq8jX1wvaQuGEoXAJWHsUWX6HWp` / `https://plaiflowapp-6gjvt8lcs-burapa009s-projects.vercel.app`.

Status reads are currently at most two existing API reads per document; a batch endpoint should precede increasing the 20-document page size.
