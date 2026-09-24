# PlaiFlow Phase 9 — Human Review and Approved CSV/XLSX

Status: **implemented behind default-off `REVIEW_ENABLED` (2026-09-24); not activated**. Migration 12 and the guided review/export flow are prepared in code. The schema migration, staging workflow proof, accepted OCR benchmark, privacy sign-off, and representative performance targets remain release gates. This specification synthesizes the accepted [Phase 9 decisions](./phase-9-decisions.md), preserves the versioned mapping rule in [ADR 0033](./adr/0033-keep-accounting-mapping-decisions-versioned.md), and uses the terms in [CONTEXT.md](../CONTEXT.md).

## Problem Statement

A Document can have an OCR Result Artifact, proposed extraction fields, and an Accounting Suggestion, but the human steps are split across pages. A reviewer cannot reliably work an assigned queue, compare each proposed value with the original, save an unfinished Review Draft, correct a mistake after approval, or move quickly to the next Document. Existing exports have separate meanings and bounded synchronous paths; users need to know whether a file contains Document metadata, confirmed values, or approved accounting suggestions. Larger exports need a visible background lifecycle. Incorrectly treating a candidate, stale approval, or revoked access as authoritative would expose sensitive business data or produce a misleading file.

## Solution

Give each Organization one guided review flow for the **current actionable revision** of a Document. Show the original beside extracted values and the Accounting Suggestion, support explicit field decisions and saved Review Drafts, then require Owner/Admin confirmation and approval through the existing two domain decisions. Assignment uses an explicit Task; closing it never approves a Document. A version change supersedes old actionable work while retaining authorized history.

Provide an export workspace with clearly separated **รายการเอกสาร** (Raw Document metadata), **ข้อมูลยืนยัน** (Confirmed structured values), and **ข้อมูลอนุมัติ** (Approved suggestions) tabs. The Approved file is a fixed, generic, versioned CSV/XLSX contract containing one current Approved Export Row per Document. Show filters, estimated row count, job status, and download eligibility. Route exports over their synchronous thresholds through Durable Jobs and private, expiring Export Artifacts. Every review, bulk action, deep link, export request, and download rechecks current Organization Membership, Document access, revision, role, and Plan entitlement.

## User Stories

1. As an Owner, I want one queue of Documents needing human action, so that I can see unfinished reviews without switching between extraction and accounting pages.
2. As an Admin, I want the queue to show the oldest actionable item first, so that work does not remain unseen.
3. As a reviewer, I want All, Assigned to me, and Unassigned filters, so that I can focus on the right work.
4. As an Admin, I want Archived Documents included only when I select them, so that the ordinary queue reflects active work.
5. As a reviewer, I want Trash Documents omitted from the actionable queue, so that I do not work on an unavailable original.
6. As a reviewer, I want the queue paginated with stable navigation, so that a large Organization remains usable while new items arrive.
7. As an Owner, I want to explicitly assign a review Task to an active Member with Document access, so that responsibility is clear without granting a new role.
8. As an assignee, I want a Task deep link to open the current authorized Document Review Item, so that reprocessing does not leave me on a stale revision.
9. As an Admin, I want a review item to become unassigned when its assignee loses access, so that the queue does not hide stranded work.
10. As a reviewer, I want the original and proposed fields visible together on a wide screen, so that I can compare them directly.
11. As a reviewer on a narrow screen, I want to switch between original and review form, so that both remain legible.
12. As a reviewer, I want a proposed field to point to its source page and line when available, so that I can check its evidence quickly.
13. As a reviewer, I want machine proposals visually distinct from human-confirmed values, so that I do not mistake a candidate for a decision.
14. As a Member with Document access, I want to accept, correct, or mark a field unknown in a Review Draft, so that an Owner/Admin can finish the review.
15. As a reviewer, I want to save a Review Draft separately from confirmation, so that interrupted work does not imply approval.
16. As a reviewer, I want a stale edit rejected with refresh guidance, so that another person's correction is not overwritten.
17. As an Owner or Admin, I want required fields and blockers checked before confirmation, so that incomplete extraction cannot be treated as confirmed.
18. As an Owner or Admin, I want to approve a current Accounting Suggestion only after its source extraction is confirmed, so that an Approved Export Row is based on reviewed values.
19. As a reviewer, I want a clear warning when the original cannot be opened, so that I do not confirm values without the source.
20. As a reviewer, I want to enter a manually read value when OCR has no evidence but the original is visible, so that a missing OCR location does not force an invented machine value.
21. As an Owner or Admin, I want to return an item for correction with a bounded reason and optional private note, so that the assignee knows what to resolve.
22. As an assignee, I want a content-free notification when work is returned, so that I can act without sensitive values appearing in notification channels.
23. As an Owner or Admin, I want to reprocess a Document without deleting its original or correction history, so that I can recover from a poor OCR result.
24. As a reviewer, I want a new OCR or extraction revision to supersede the prior actionable revision, so that I cannot approve stale evidence.
25. As an Owner or Admin, I want a correction after approval to require new confirmation and approval, so that the current export reflects the current revision.
26. As a reviewer, I want Save and next to preserve my queue filters and open the next item, so that routine review is fast.
27. As a keyboard user, I want safe shortcuts for previous/next item, next field, and saving a draft, so that I can work quickly without accidentally approving.
28. As an Admin, I want to assign or organize up to 50 selected review items in one request, so that triage is efficient.
29. As an Admin, I want the entire bulk request rejected if one item is unauthorized or stale, so that I never receive an ambiguous partial result.
30. As an Owner, I want Raw Document metadata export with explicit Available, Archived, and Trash status filters, so that I can obtain the authorized inventory I need.
31. As an Owner, I want Confirmed structured export distinct from Approved export, so that a Document without an approved category is not misrepresented.
32. As an Admin, I want Approved export to include only current confirmed extraction and current approved suggestion revisions, so that every row has a human decision behind it.
33. As an export requester, I want to see the selected date basis, filters, row count, format, and template version before requesting a file, so that I know what it will contain.
34. As an export requester, I want large files to show Queued, Running, Ready, Failed, or Expired states, so that I know whether to wait, narrow filters, or request a new file.
35. As an Owner, I want a stable row ID and versioned column contract, so that I can reconcile repeated generic exports without assuming the destination deduplicates them.
36. As an export requester, I want a stale or revoked Export Artifact blocked at download with a clear explanation, so that outdated or unauthorized data is not released.
37. As an Owner, I want approval, correction, return, reprocess, export, and download actions auditable, so that a dispute can be investigated without exposing field values in operational logs.
38. As a Member, I want queue summaries to omit Tax IDs and amounts while allowing field review only on Documents I can access, so that list views do not disclose unrelated sensitive details.
39. As an Organization administrator, I want a 20-page preview and the next queue item to remain responsive with several reviewers active, so that document review is practical during a workday.
40. As a product owner, I want Phase 9 kept behind a default-off feature flag until benchmark, privacy, security, and performance gates pass, so that a synthetic smoke test is not mistaken for release acceptance.

## Implementation Decisions

### Review queue and assignment

- A Document Review Item is the one actionable current review for a Document in its Organization. A newer OCR/extraction or review revision supersedes prior actionable work; historical revisions remain available only through authorized history and cannot be approved.
- The default queue includes actionable Available Documents, oldest first. Provide All, Assigned to me, and Unassigned views, plus an explicit Archived filter. Exclude Trash and purged Documents. Page with a stable cursor and bounded size; do not scan or return the whole Organization to render one page. Queue rows omit Tax IDs, amounts, raw OCR, and private correction text.
- Assignment is an explicit reuse of the existing Task domain, with at most one open review Task for the current Document work. Owner/Admin may assign an active Member with current Document access. Assignment never changes review or approval authority. Task completion alone does not remove an unfinished Document Review Item.
- When a Membership or Document access ends, remove the effective assignee from actionable work. A Task link resolves the current revision by Document after current-membership authorization; the old revision is visible only as permitted history. Reprocessing retains an open Task if its assignee still has access and does not create a duplicate.
- Save and next uses the current server-authorized queue filter and next-item position. Prefetch only minimal next-item metadata, scoped by Organization and current role; do not prefetch original bytes or sensitive fields.

### Original, extraction, suggestion, and correction

- The review screen presents original preview, source page/line evidence, extraction proposal, Review Draft, and Accounting Suggestion in a single guided flow. Use adjacent panes on wide screens and an accessible switch on narrow screens. Candidate values, manually entered values, and confirmed values have distinct labels and states.
- Fetch original pages on demand with bounded memory and byte ranges or equivalent provider capability; prioritize the first visible page. Keep originals and private evidence behind current Document authorization. A preview failure blocks confirmation and approval until the original can be opened. A reviewer may record a value read directly from an available original when OCR evidence is absent, marked as manual provenance.
- Each reviewed field is explicitly accepted, corrected, or marked unknown. Required fields and warning/blocker rules remain those of the applicable Phase 7 extraction schema; unknown is not a substitute for a required value. Save Draft and Confirm are separate actions. A Member with Document access may save/propose corrections but cannot confirm extraction or approve a suggestion.
- A Review Draft has a monotonic revision. All save, confirm, and approve requests carry the expected current revision and are rejected on conflict. Confirmation rechecks the authoritative OCR/source identity, Document status, current role, required fields, and original availability in one authorization/concurrency-safe decision.
- The guided flow uses Phase 7 confirmation and Phase 8 suggestion approval. It adds no third approval state. Phase 8 approval requires Owner/Admin, an active Expense Category, a current confirmed source, current relevant rule-set version, and expected suggestion revision. A one-Document correction does not create a reusable mapping rule. Rule changes affect later suggestions as defined by ADR 0033.
- Return for correction and Reprocess are Owner/Admin actions. Return requires a bounded server-owned reason code, allows a private note, keeps the Task open, and sends a content-free notification. Reprocess preserves the original and prior protected correction history, creates a new actionable revision, and never copies old human values into a new approval. Technical failures use safe failure codes rather than being mislabeled as business rejection.
- A post-approval correction supersedes the current eligibility of the old approval. Reconfirmation and reapproval create a new Approved Export Row identity. Already imported external files cannot be recalled or updated by PlaiFlow; explain this when showing stale exports.
- Bulk operations cover assignment and queue organization only. They cannot confirm extraction or approve suggestions. Limit each request to 50 items, verify Organization, current access, role, Document state, and revision for every ID server-side, and commit all or none. A stale or cross-tenant item returns a safe conflict/authorization result without leaking the foreign item.
- Keyboard controls cover previous/next item, next field, and Save Draft; do not trigger while typing into an editable field. Confirmation and approval require explicit labeled controls and a fresh server blocker check. Provide visible focus, non-keyboard parity, and screen-reader descriptions for evidence links and status changes.

### Export products and lifecycle

| Product | Row eligibility and date basis | Default and explicit status filters | Synchronous threshold |
|---|---|---|---:|
| Raw Document metadata | Authorized Document inventory; accepted date | Available by default; Archived or Trash explicitly; never Purged | 5,000 rows |
| Confirmed structured values | Current confirmed extraction revision; confirmation date | Available by default; Archived explicitly; never Trash/Purged | 100 rows |
| Approved suggestions | Current confirmed extraction plus current approved suggestion; approval date | Available by default; Archived explicitly; never Trash/Purged | 100 rows |

- Owner/Admin request and download exports under existing CSV/XLSX Plan capabilities. A Member may review accessible Document fields but cannot request these bulk files. The three products have separate, plainly labeled tabs, filters, row-count preview, format and template labels, job status, and safe error/expiry messages. Interpret date filters in Organization timezone; template timestamps are UTC.
- Raw export is metadata only: no original bundle, OCR text, unconfirmed extraction values, signed URLs, provider credentials, or audit payload. Confirmed structured export remains the separate Phase 7 product for Documents without an approved category. Approved export uses only reviewed source values and approved Vendor/category decisions.
- The Approved template is fixed generic `accounting_suggestions_v1`, one row per current Approved Document/suggestion revision, with columns in this exact order: `row_id`, `document_id`, `source_review_revision`, `document_type`, `reviewed_issue_date`, `reviewed_document_number`, `reviewed_seller_name`, `reviewed_seller_tax_id`, `reviewed_seller_branch`, `vendor_contact_code`, `expense_category_id`, `expense_category_name`, `reviewed_currency`, `reviewed_subtotal`, `reviewed_vat_amount`, `reviewed_total_amount`, `suggestion_basis`, `rule_version`, `warning_codes`, `approved_at`, `template_version`. A missing optional value is an empty cell; an actual reviewed zero is `0.00`. A human-only choice has no `rule_version`. `row_id` is stable for re-export of the same approval revision. An incompatible contract change creates a new template version.
- Do not present the generic file as import-ready for a named accounting product. No inferred ledger account, tax treatment, payment channel, arbitrary column/formula, or raw OCR enters Approved export. CSV follows UTF-8/BOM, correct quoting, and formula neutralization; XLSX writes untrusted values as text without formulas, macros, active links, or external references. Document how a machine importer should interpret a protective CSV prefix.
- At request time, freeze authorized row IDs, source/suggestion revisions, filters, format, template version, requester, and row count. Recheck requester/current Owner/Admin role, Organization, Document visibility/status, source/suggestion revisions, and Plan entitlement before download. If any included row changed or became inaccessible, fail the whole snapshot and require a fresh request; never silently replace or omit a row.
- Keep bounded synchronous behavior at the thresholds above. Larger entitled requests use the existing Durable Job claim/lease/retry protocol and private Export Artifact flow, subject to existing row, byte, time, and memory ceilings. Do not truncate or buffer an unbounded Organization in memory; stream/paginate rows and private artifact writes. Map the durable lifecycle to user-facing Queued, Running, Ready, Failed, and Expired. Failed/expired jobs have safe explanations and a new-request path.
- Export Artifacts expire after 24 hours; an authorized signed download URL expires after 15 minutes. Only the requester and current Owner/Admin with the required export authority may view job status or obtain a download. Authorization must be repeated at download even if the job completed earlier. Audit request, rejection, completion, download, and expiry without bearer URLs or file contents.

### Audit and security

- Every deep link, queue page, Document review read, field-draft write, assignment, return, reprocess, confirmation, approval, export snapshot, job status, and download checks current session, Organization Membership, Document visibility, role, and applicable Plan capability. Browser-supplied Organization IDs, Document IDs, revisions, filters, and bulk ID lists are selectors, not authority. Preserve CSRF protection on mutations and tenant-scoped database policies.
- Restrict sensitive field reads to a Member's currently accessible Document. Queue rows and notifications omit Tax IDs, amounts, correction notes, and OCR text. A Member sees current review values and their own correction history for accessible Documents; Owner/Admin can inspect Organization review history and content-free audit metadata. No cross-Organization cache, rule learning, or artifact access.
- Write append-only audit facts for draft changes, return/reprocess, confirmation, approval, bulk actions, template/export transitions, and downloads: Organization, actor, safe action/reason, entity IDs, old/new revisions, timestamp, and request/job correlation. Keep old/new field values in separately protected review history, never in audit metadata, operational logs, metrics, Task titles, or notifications.
- Purge protected correction values with permanent Document deletion. Retain content-free export/review audit for one year under the existing contract. Privacy/legal sign-off of this retention and purge behavior remains a release gate.
- Treat original content, OCR output, filenames, field values, notes, and spreadsheet cells as untrusted. Bound input sizes, evidence references, note lengths, warning codes, queue pages, bulk IDs, export rows/bytes, and job concurrency. Keep workers least-privileged, artifacts private, and download URLs short-lived. Test cross-tenant IDs, stale revisions, role removal, spreadsheet formulas, and export races.

### Performance and activation

- Use indexed, Organization-scoped keyset pagination for the queue and filters; avoid evaluating every Document or reading encrypted review blobs merely to list a page. Fetch detail only when opened. Prefetch minimal metadata for the likely next item without widening access. Optimize original preview for the first visible page and load other pages on demand.
- Measure queue page, next-item navigation, first preview page, review save/approval, job wait, export duration and throughput, and memory/bytes with five concurrent reviewers, up to 20-page PDFs, and a 5,000-row Approved export. Record P50/P95 and rows per second without content-bearing labels. Set numeric pass criteria from a representative baseline before activation; a synthetic test is not performance or OCR-accuracy acceptance.
- Implement additively behind a default-off Phase 9 gate. Keep existing Document, Phase 7, Phase 8, and export behavior intact when the gate is off. Activation requires accepted OCR/extraction evidence, signed-off correction-history retention, cross-tenant/role and spreadsheet safety proof, representative performance, and a staging Thai-document flow through confirmation, approval, and one correct Approved file. Verify flag-off rollback and do not use a deployment-ready indicator as evidence of end-to-end correctness.

## Testing Decisions

- The primary seam is the authenticated, Organization-scoped API workflow: current OCR-backed Document → Review Draft → confirmation → suggestion approval → export request → authorized download. Tests assert visible responses, persisted state, row content, and audit effects. Reuse the existing extraction, accounting, Document-export, and Durable Job HTTP test patterns instead of creating a parallel test-only service.
- Use database-backed integration tests where atomicity and tenant policies matter: one current item per Document, concurrent draft saves, reprocess during approval, rule-version change, post-approval correction, one active review Task, Membership loss, deep-link access, and all-or-nothing 50-item bulk actions. Assert foreign IDs do not disclose existence or partially mutate a batch.
- Test Owner/Admin/Member behavior from the public API, including Member proposals without confirmation, accessible versus inaccessible sensitive fields, content-free queue/notifications, revoked status/download rights, and Trash/Archived filters. Verify Task completion has no approval side effect.
- Exercise the review UI at desktop and narrow widths: original beside form or switchable, evidence navigation, unavailable original blocking, manual provenance, revision-conflict recovery, Save and next, keyboard focus/shortcuts, and equivalent pointer controls. Use one end-to-end browser path for the critical Thai-document review rather than tests that mirror component internals.
- Parse exported CSV and XLSX as a consumer would. Assert exact `accounting_suggestions_v1` column order and row identity, Thai text, dates/decimals, null versus `0.00`, category/Vendor basis, formula neutralization, XLSX text-only cells, current-revision filtering, and no OCR/raw/private fields. Test 0, threshold, threshold+1, and hard-cap row counts; a job must never report success with truncated output.
- Test synchronous and Durable Job exports through the same user-visible contract: frozen filters/revisions, transient worker retry, terminal failure, 24-hour artifact expiry, 15-minute URL expiry, requester/Owner/Admin visibility, role revocation, Document change during generation, and download after change. Include a 5,000-row Approved run that measures rows per second and bounded worker memory.
- Verify append-only, content-free audit for correction, return, reprocess, confirmation, approval, bulk rejection, export request/completion/download, and purge. Inspect logs/metrics for names, Tax IDs, amounts, original text, notes, tokens, and signed URLs.
- Before activation, record a representative benchmark with five concurrent reviewers and 20-page PDFs, derive numeric P50/P95 and throughput pass criteria, and rerun it in staging. Keep the OCR benchmark and human-reference accuracy gate separate from UI/export throughput evidence.

## Out of Scope

- A third approval state, auto-approval, bulk confirmation/approval, or Task completion that implicitly approves a Document.
- Accounting ledger entries, chart-of-accounts ownership, tax advice, posting, payment, or filing.
- Raw OCR text, unconfirmed extraction values, original-file bundles, or private audit payloads in Raw Document Export.
- A named destination adapter, user-authored columns/formulas, guaranteed import compatibility, or automatic recall of files imported elsewhere.
- Cross-Organization corrections, shared Vendor learning, unrestricted worker credentials, public artifacts, and background processing outside the existing Durable Job protocol.
- Claiming OCR accuracy, numeric latency compliance, or production readiness from synthetic smoke tests or a successful deployment alone.

## Further Notes

- The current implementation has separate extraction confirmation and accounting approval UI. Its accounting queue scans recent confirmed reviews by offset; Phase 9 needs one Organization-scoped actionable queue with stable pagination and explicit assignment semantics.
- Current Raw Document export rejects requests above 5,000 rows; current Confirmed and Approved exports cap synchronous output at 100 rows. Phase 9 must add the promised Durable Job path for all three rather than treating a larger rejected request as an asynchronous success.
- The Phase 6 benchmark currently reports `acceptance_passed=false` and zero human reference files. Local OCR execution or one successful file does not satisfy the activation gate.
- The agreed decisions are in [Phase 9 decisions](./phase-9-decisions.md). Relevant prior contracts are [Phase 5 Durable Jobs](./phase-5-spec.md), [Phase 7 extraction](./phase-7-spec.md), [Phase 8 accounting](./phase-8-spec.md), and [ADR 0033](./adr/0033-keep-accounting-mapping-decisions-versioned.md).
