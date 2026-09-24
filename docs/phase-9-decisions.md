# Phase 9 — Human Review and Approved CSV/XLSX decisions

Status: decisions synthesized into [Phase 9 specification](./phase-9-spec.md) and implemented behind a default-off gate on 2026-09-24. This document records product decisions; it does not itself authorize database migration or feature activation.

## Agreed in round 1

- Phase 9 presents the existing Phase 7 extraction confirmation and Phase 8 accounting-suggestion approval in one guided review flow. It does not introduce a third persisted approval. An Approved export row requires both a confirmed current extraction and an approved current suggestion.
- The actionable review queue has at most one current item per Document. Reprocessing or a new extraction revision supersedes the prior actionable revision; the old revision remains history and cannot be approved.
- A Member with Document access may propose corrections. Only an Owner or Admin may confirm extraction or approve a suggestion. Assignment never grants additional authority.
- Raw export means authorized Document metadata, not OCR text, unconfirmed extraction fields, or original-file bundles. Confirmed structured export and Approved suggestion export remain separate products.
- Assignment creates an existing Task only when explicitly requested. Closing the Task does not approve the Document or remove an unfinished review item from the queue.
- The first Approved CSV/XLSX template is generic, fixed, and versioned. A named destination template needs the destination's official current workbook and a verified trial import.

## Agreed in round 2

- The default review queue shows only actionable items, oldest first. It has All, Assigned to me, and Unassigned filters. Archived Documents require an explicit filter; Trash Documents are excluded.
- An Owner/Admin may explicitly assign one active review Task to an active Member who can access the Document. Other authorized users may read the review, but each save checks the current revision to prevent overwrites. If the assignee loses access, the unfinished item becomes unassigned. Closing the Task does not complete review.
- On a wide screen, show the original beside the review form; on a narrow screen, switch between them. Each proposed field links to its source page/line when evidence exists. Keep machine proposals visibly distinct from human-confirmed values.
- Saving a review draft and confirming extraction are separate actions. Each field is explicitly accepted, corrected, or marked unknown. Revision checks protect concurrent edits; missing required fields block confirmation.
- Return for correction and Reprocess are separate actions; neither deletes the original. Reprocessing creates a new revision and retains old human corrections in history without carrying them into a new approval automatically.
- After confirmation or approval, an explicit Save and next action opens the next item under the same filters, with metadata prefetched. If none remains, return to the queue with a completion message.

## Agreed in round 3

- Bulk actions may assign and organize review items but may not confirm extraction or approve suggestions. A reviewer must inspect each Document's evidence individually. The selection cap is still to be set.
- Keyboard navigation covers previous/next item, next field, and saving a draft. Confirmation and approval require an explicit labeled action with a fresh blocker check; single-letter shortcuts do not fire while editing a field.
- Raw Document Export uses accepted date and defaults to Available Documents. Authorized users may explicitly include Archived or Trash metadata; purged Documents are excluded. Approved Export uses approval date and defaults to Available; Archived requires an explicit filter, and Trash/purged are excluded.
- Freeze the exact `accounting_suggestions_v1` column names, order, and value semantics proposed in `docs/phase-8-spec.md` as the first Approved CSV/XLSX contract. Any incompatible column change requires a new template version.
- A Member may see sensitive fields only for a Document they are currently authorized to read, even if not assigned its review Task. Queue rows do not expose Tax IDs or amounts. Only Owner/Admin may request Approved export.
- Append-only audit records Organization, actor, action, safe reason, entity IDs, revisions, time, and request/job correlation. Sensitive old/new values remain in separately protected review history, not operational logs or audit metadata. Review-history retention requires a signed-off policy before production activation.

## Agreed in round 4

- Raw Document Export remains Owner/Admin only, including an explicit Trash metadata export.
- A bulk assignment/organization request selects at most 50 review items. The server validates the current Organization, authorization, and revision of every item; one invalid or stale item rejects the entire batch with refresh guidance. There is no partial success.
- An Approved export request freezes authorized row IDs, source/suggestion revisions, filters, template version, and requester. Recheck current role, Document visibility, and revisions before download. A changed or revoked snapshot cannot be downloaded and requires a new request. Export artifacts expire after 24 hours; authorized download links expire after 15 minutes.
- Approved exports of at most 100 rows use the bounded synchronous path. Larger requests use Durable Jobs with visible Queued, Running, Ready, Failed, and Expired states, subject to existing Plan entitlements and export row/byte ceilings. Never silently truncate. The current implementation has no Approved async export path yet.
- Protected correction values remain only while their Document exists; purge them with permanent Document deletion. Keep content-free audit for one year under the existing audit contract. Privacy/legal sign-off of correction-history retention is required before production activation.
- Initial performance test workload is five concurrent reviewers, Documents up to 20 PDF pages, and 5,000 Approved rows. Measure queue, next-item, and first-preview-page P50/P95 latency plus Approved export rows per second; set numeric acceptance targets from a representative baseline before activation.

## Agreed in round 5

- A post-approval correction creates a new review revision and makes the old approval ineligible for current export. The old export remains audit history but a stale snapshot is not offered for download. A new confirmation and approval are required; PlaiFlow does not recall files already imported into an external system.
- Return for correction requires a bounded server-owned reason code and may include a private note. Its Task remains open and the assignee is notified without document content, Tax IDs, or amounts in the notification.
- If the original cannot be opened, block confirmation and approval and offer retry. When the original is available but an OCR field lacks evidence, a reviewer may enter a manual value tied to the original and recorded as such in review history.
- The export workspace separates Raw Document metadata, Confirmed structured values, and Approved suggestions into three plainly named tabs. Show filters, row count before request, job status, and a safe explanation when download is unavailable. Interpret date filters in Organization timezone and write template timestamps in UTC.
- A Member sees current values and their own corrections only for Documents they can currently access. Owner/Admin may inspect Organization review history and content-free audit metadata.
- Phase 9 can be developed and tested behind a default-off feature flag. Production activation requires an accepted OCR/extraction benchmark, signed-off correction-history retention, cross-tenant/role tests, representative queue/preview/export performance, and a staging Thai-document flow through review and Approved export.

## Agreed in round 6

- Phase 9 adds Durable Job exports for all three products above their synchronous thresholds: Raw Document metadata above 5,000 rows, Confirmed structured values above 100 rows, and Approved suggestions above 100 rows. Keep existing Plan entitlements and hard row/byte ceilings.
- A Member may report an issue and propose a draft correction. Only an Owner/Admin may Return for correction or Reprocess; reprocessing makes the former actionable revision ineligible for approval.
- An assigned review Task stays open across reprocessing while its assignee retains Document access. Its deep link resolves the current actionable review revision after fresh authorization; history records the superseded revision. Do not create a duplicate Task or permit approval through a stale link.

## Security and performance requirements supplied for Phase 9

- Reauthorize deep links and every review or download request against current Organization Membership and Document access. Bulk actions must preserve tenant boundaries; approval must be audited. Approved export requires the appropriate role; sensitive fields are filtered by role.
- Design for a paginated queue, fast next-item flow, metadata prefetch, and an optimized original preview. Define the async bulk-export threshold and Approved export throughput from an explicit workload and measured baseline in later interview rounds.

## Release gates, not unresolved product choices

- Set numeric P50/P95 and Approved-export throughput pass criteria after a representative baseline. Do not claim they pass from a synthetic test alone.
- Obtain privacy/legal sign-off for correction-history retention and accepted OCR/extraction benchmark evidence before production activation.
- Verify current-membership authorization, tenant isolation, revision races, audit, spreadsheet safety, export expiry, and the end-to-end staging flow before opening the feature flag.
