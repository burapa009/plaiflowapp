# PlaiFlow Phase 8 — Accounting Suggestion and Export Mapping

Status: **implemented behind default-off feature flags (2026-09-23); not activated**. The generic export column contract and Phase 8 ticket breakdown were approved by the user. Commits `5acf16b` and `0b1fc8f` implement the gated flow; production activation still depends on the listed Phase 7, retention, security, and performance gates.

## Problem Statement

An Organization can review a Document's extracted fields, but turning a confirmed expense document into a usable category and Vendor reference still requires repetitive manual work. A name match can confuse different branches, a single Vendor can sell different types of goods, and an OCR result is not accounting authority. Users also need a structured file that preserves the reviewed values and explains how each suggestion was made, without PlaiFlow silently posting to a ledger or claiming compatibility with an unverified accounting-system import template.

## Solution

For each eligible, confirmed expense-side Document revision, PlaiFlow proposes at most one Organization-scoped Accounting Suggestion. It first uses approved deterministic mapping rules, explains the rule or ambiguity, and routes unresolved cases to a review queue. An Owner/Admin explicitly chooses an Expense Category and approves the suggestion. Approved current revisions can be exported as one safe, structured CSV or XLSX row per Document. The first release has a generic, versioned export only. Destination Mapping Templates are a versioned contract for a later verified destination, not a claim that the generic file imports into any named product.

## User Stories

1. As an Owner, I want Phase 8 to use only confirmed Document values, so that an OCR guess cannot become an accounting suggestion.
2. As an Admin, I want to see which Document and confirmed revision a suggestion came from, so that I can compare it with the original.
3. As a Member with Document access, I want to see the current suggestion and its basis, so that I can flag a mistake without approving it.
4. As an Owner, I want an editable, flat list of my Organization's Expense Categories, so that the terms match my business.
5. As an Admin, I want to Archive and Restore categories, so that old history remains understandable without offering obsolete choices for new work.
6. As a Member, I want active categories to be selectable in a proposed correction, so that I do not have to type an untracked label.
7. As an Owner, I want Vendor candidates to match a Business Contact by strong identity, including branch where relevant, so that similar names do not link the wrong party.
8. As an Admin, I want an unmatched or conflicting Vendor shown explicitly, so that the system does not invent a party.
9. As an Owner, I want to approve a reusable Vendor/category rule separately from correcting one Document, so that one unusual purchase does not train future suggestions.
10. As an Admin, I want a Vendor-and-document-type rule to take precedence over a Vendor default, so that a more specific approved rule is applied predictably.
11. As an Owner, I want equal-priority conflicting rules to stop and request review, so that a recent edit does not silently win.
12. As an Admin, I want an approved per-Document choice to take precedence over rules, so that my explicit decision is preserved.
13. As an Owner, I want a Vendor default to be a candidate rather than an authoritative classification, so that a Vendor selling different goods cannot auto-post a wrong category.
14. As a reviewer, I want the candidate Vendor, category, rule basis, and warnings displayed separately, so that I know what I am approving.
15. As a reviewer, I want missing, ambiguous, and failed states distinguished, so that technical failure is not mistaken for a business exception.
16. As a reviewer, I want no uncalibrated accuracy percentage, so that a deterministic match is not misrepresented as statistical confidence.
17. As an Owner, I want a category required before approval, so that an approved Accounting Suggestion has a useful classification.
18. As an Admin, I want to approve a category with an unmatched Vendor only by recording a reason, so that a missing identity remains visible.
19. As a Member, I want to propose a correction without confirming it, so that I can help review while preserving role boundaries.
20. As an Owner, I want pending suggestions based on an older rule version to be refreshed before approval, so that I cannot accidentally approve stale logic.
21. As an Admin, I want a source-review change to supersede its suggestion, so that the next export uses the correct Document revision.
22. As an Owner, I want old approved decisions and exports to retain their original rule and template versions, so that historical files remain explainable.
23. As a reviewer, I want unmatched and conflicting suggestions in a review queue, so that I can work through exceptions without an automatic Task for every file.
24. As an Admin, I want to create a Task only when I choose to assign an exception, so that the Task list is not flooded.
25. As an Owner, I want only Approved current suggestions in the Phase 8 export, so that unreviewed candidates do not leave the Organization as accounting-ready rows.
26. As an Admin, I want one row per Document with reviewed values and an explicit category, so that I can reconcile the file against its source.
27. As an export requester, I want an empty cell to mean unknown and `0.00` to mean reviewed zero, so that absence is not fabricated as an amount.
28. As an export requester, I want a stable row identifier across re-exports of the same approved revision, so that I can recognize duplicates before importing elsewhere.
29. As an export requester, I want CSV and XLSX cells safe to open in spreadsheet software, so that untrusted names cannot become formulas or links.
30. As an Owner, I want the generic file clearly labeled as generic, so that I do not mistake it for a tested PEAK, FlowAccount, or Express import.
31. As an Owner, I want future destination formats versioned by PlaiFlow while my Organization owns its mapping values, so that a destination change does not leak or alter another Organization's rules.
32. As an auditor, I want an attributable history of suggestion approval, correction, mapping-rule changes, and exports, so that I can reconstruct what was known at the time.
33. As a user in another Organization, I want no access to these rules, suggestions, caches, or exports, so that tenant data stays isolated.
34. As an operator, I want unmatched/conflict rates, cache hit rate, P50/P95 latency, and fallback cost measured without document content, so that I can assess quality and cost safely.
35. As an Owner, I want no direct authoritative posting or automatic external AI transmission, so that PlaiFlow remains a review-and-export tool under my control.

## Implementation Decisions

### 1. Eligibility and source identity

- Phase 8 starts with expense-side `receipt`, `invoice`, or `tax_invoice` Documents whose current Phase 7 extraction revision is Confirmed and whose original is Available, or Archived when explicitly included by authorized filter. Unknown/other types, Trash, missing OCR, unconfirmed extraction, and superseded source reviews are ineligible. A file suspected of multiple business documents cannot be silently split; one Document yields at most one suggestion in v1.
- Bind every suggestion to server-owned Organization ID, Document ID, immutable confirmed extraction review ID/revision, source checksum/OCR identity where available, suggestion schema/rule-set version, and current Document visibility. Never trust browser-supplied Organization, Vendor, object key, or source revision as authority.
- Extend the reviewed extraction contract with optional seller branch identity before exact branch-based Vendor matching. Absence or ambiguity of branch is not silently interpreted as head office. Reuse the existing Business Contact concept: Vendor is a role, not a separate master entity. Contact Code may be retained for export after a human or strong match, but an invoice does not supply PlaiFlow's Contact Code as an automatic match key.

### 2. Organization reference data and approved rules

- Complete the Phase 3 intent for flat Organization-owned Expense Categories if it has not shipped. Seed a small editable Thai list; names are business groupings, not chart-of-accounts codes, tax determinations, or ledger entries. Archive excludes a category from new suggestions while preserving historical references. Owner/Admin mutates; Member reads/selects active entries.
- An Approved Mapping Rule belongs to one Organization and references an active Vendor Business Contact and active Expense Category. It may additionally target an eligible document type; a Vendor-only rule is a less-specific default. Store stable ID, scope/key, category, status, monotonic version, approver/time, and audit reason. Owner/Admin explicitly creates, approves, retires, or changes a rule; a one-Document correction never creates a rule by itself.
- Evaluate in this order: an approved per-Document decision; a current approved exact Vendor + document-type rule; a current approved Vendor default; then no candidate. Do not fuzzy-match names, search other Organizations, use creation time to break ties, or infer a category from OCR text. Equal-priority conflicting rules produce `rule_conflict` and no category. A Vendor rule suggests only; it never approves a Document.
- An archived Vendor/category or changed rule-set version invalidates new matches and pending approvals. Previously Approved suggestions and export snapshots retain their original IDs/versions; they are not retroactively rewritten. A pending suggestion under an old rule version must be regenerated before approval.

### 3. Suggestion schema, review, and exceptions

- Version the Accounting Suggestion schema. Its bounded metadata includes server-owned IDs, source review identity/revision, rule-set version, candidate Vendor/contact and category IDs (nullable), `suggestion_basis`, matching rule ID/version (nullable), warning codes, status, suggestion revision, reviewer/approval actor and time, and creation/update times. Monetary/source fields are read from the confirmed extraction; do not create a second unreviewed financial copy merely for a list view. Sensitive correction details require protected storage and access control, not content-bearing logs.
- Statuses are `Draft`, `NeedsReview`, `Approved`, `Superseded`, and `Failed`. Stable warning codes initially include `vendor_unmatched`, `branch_unknown`, `category_unmatched`, `rule_conflict`, and `source_changed`. Unknown and ambiguous fields remain empty, never invented. `Failed` means a technical failure, not a low-confidence business value. Keep warning lists and user-visible text bounded and server-owned.
- Do not present a numeric suggestion-confidence score. Until human-reference calibration exists, any machine confidence is `unrated`; show `suggestion_basis` such as human-approved choice, approved exact rule, approved default, no rule, or conflict. OCR line confidence remains OCR evidence only. An Owner/Admin's approved value is reviewed, not “model high-confidence.”
- Owner/Admin approval requires an active Expense Category, a current confirmed source revision, current rule-set version, current Organization role and Document access, and an expected suggestion revision in one concurrency-safe decision. Vendor may be null only with an explicit reason. Members may propose a correction within their Document access but cannot approve, manage rules, or bulk export. A stale revision returns conflict; no last-write-wins overwrite. Source-review changes supersede the old suggestion; the old approved snapshot remains in history.
- Present unmatched/conflicting items in an Organization-scoped review queue. Do not create a Task automatically for every warning. Explicit assignment may create a generic Task linked to the authorized review page; no business values in Task title or notifications. Closing a Task never approves a suggestion.
- Keep append-only correction/rule/template/export audit with Organization, actor, action, safe reason code, entity IDs, old/new versions, timestamp, and request/job correlation. Never put full names, tax IDs, amounts, free-text document content, tokens, or file URLs in operational logs or metrics. Correction-history retention needs legal/privacy approval before production.

### 4. Generic CSV/XLSX export and destination mapping

- The first active template is fixed `accounting_suggestions_v1`, generic CSV/XLSX, one row per Approved current Document/suggestion revision. Owner/Admin only, within the existing Plan capability gates. Available Documents are the default; Archived requires an explicit filter; Trash/purged are excluded. The Phase 7 structured Document export remains a separate route for records without an approved Phase 8 category.
- **Approved v1 columns:** `row_id`, `document_id`, `source_review_revision`, `document_type`, `reviewed_issue_date`, `reviewed_document_number`, `reviewed_seller_name`, `reviewed_seller_tax_id`, `reviewed_seller_branch`, `vendor_contact_code`, `expense_category_id`, `expense_category_name`, `reviewed_currency`, `reviewed_subtotal`, `reviewed_vat_amount`, `reviewed_total_amount`, `suggestion_basis`, `rule_version`, `warning_codes`, `approved_at`, `template_version`. An optional unknown is an empty cell; reviewed decimal zero is `0.00`. If a per-Document human choice had no reusable rule, `rule_version` is empty and the basis says human-reviewed. `row_id` is a stable opaque identifier for the approved suggestion revision; re-exporting it does not create a new row identity.
- Export only reviewed/approved values. No OCR raw text, inferred ledger account, tax treatment, payment channel, arbitrary user formula, storage key, signed URL, or private audit payload. CSV uses the existing UTF-8/BOM, quoting, and formula-neutralization contract; XLSX writes untrusted text as text cells with no formulas, macros, or external links. Template/version and generic status must be visible in filename and metadata; document the safe CSV encoding so a machine importer does not mistake a protective prefix for source data.
- Freeze the authorized row identities, source/suggestion revisions, filters, template version, requester, and row count at export request. Recheck current role and Document visibility at download. A changed, removed, or inaccessible revision fails safely and asks for a new export rather than silently substituting current data. A generic file is not guaranteed to deduplicate in any external product; the stable row ID helps human reconciliation only.
- Respect the currently implemented bounded synchronous path (100 reviewed rows). Above that, use the existing Durable Job design with hard row/byte/time limits and private artifacts after it is integrated and verified for this export; until then reject oversized requests explicitly, never truncate or buffer an unbounded Organization into memory. Reuse the existing short-lived, authorized download and artifact-expiry pattern where applicable.
- A Destination Mapping Template is a platform-owned, fixed, versioned column/validation contract for one named destination. Organization-specific mapping values stay isolated and versioned; users cannot add arbitrary columns, formulas, or code. No named adapter is active in Phase 8 v1. A later adapter requires the destination's current official workbook, required-field semantics, an explicit policy for any account/tax/payment fields, a human-controlled import workflow, and a successful trial import. Do not call a file PEAK-compatible merely because PEAK documents XLSX import.

### 5. Security, performance, and release boundary

- Enforce Organization and role checks at every rule/suggestion read and write, review-queue link, export snapshot, and download, backed by tenant-scoped database policies. Cache keys must include Organization and approved rule-set version; invalidate on rule/category/Vendor changes. No cross-Organization learning, shared Vendor-name cache, or reuse of another customer's corrections.
- Treat OCR text, Document metadata, category labels, Vendor names, and all spreadsheet fields as untrusted input. Bound request/body sizes, rule counts, candidate counts, warning counts, export rows/bytes, work duration, and concurrent jobs. Preserve least-privilege worker access and private storage. The primary risks are cross-tenant rule/cache leaks, stale-source exports, formula injection, unaudited edits, and a generic file being mistaken for a verified destination file.
- Deterministic mapping is the only v1 engine. LLM fallback is disabled, sends no Document data externally, and incurs zero provider cost. Measure eligible evaluations, no-rule/conflict counts and rate, cache hit/miss, rule-evaluation and end-to-end P50/P95 latency, export duration/size, stale-version conflicts, and provider cost/status without content-bearing metric labels. Do not fabricate an accuracy rate or numeric SLO before a representative baseline; measured targets are a release gate.
- Use additive, default-off migrations/feature flags and preserve Phase 7/other exports if Phase 8 is disabled. No destructive rollback of historical suggestions/rules. Production activation requires Phase 7 OCR/extraction acceptance evidence, a signed-off correction-history retention/deletion policy, security/performance tests, staging smoke of a reviewed Thai tax invoice through a single generic XLSX row, healthy logs/metrics, and a verified flag-off rollback path. Current Phase 6 benchmark does not provide a passing human-reference accuracy baseline.

## Testing Decisions

- The primary test seam is the authorized Organization-scoped API workflow: confirmed Document → candidate with rule provenance → correction/approval or conflict → one generic CSV/XLSX row. Assert user-visible responses, authorization, stable row contents, and audit effects rather than private function calls. Reuse the existing extraction-route HTTP fixture pattern and document export tests; do not add a test-only service boundary if the current API seam suffices.
- Table-test the deterministic evaluator only for combinations awkward to express repeatedly through HTTP: exact Vendor+type versus Vendor default, explicit per-Document choice, two equal-priority conflicts, missing branch, archived references, rule-version changes, and no-match. Use one shared rule-evaluation contract; do not test cache implementation details except observable invalidation/isolation.
- Contract-test the versioned suggestion and generic export schemas: required category, nullable Vendor with reason, null versus zero, bounded warning codes, stable row ID, column order/version, exact decimals/dates, one Document per row, and no raw OCR or invented account/tax/payment value.
- Test Owner/Admin/Member boundaries, another Organization's IDs and warmed cache, source revision races, rule updates during approval, archived Documents/reference data, and download after membership removal. Include PostgreSQL policy integration tests, not only handler mocks.
- Parse generated CSV and XLSX as a consumer would. Assert escaping for names beginning with whitespace/control and formula characters, XLSX text cells, no active formulas/links, correct Thai UTF-8, large-row rejection or durable-job completion, and no silent truncation. Reuse the existing spreadsheet safety tests.
- Staging smoke with a disposable Thai tax invoice: confirm source values, propose and approve one category, see an unmatched Vendor reason if applicable, export exactly one row, test a rule conflict and source change, and verify another Organization cannot see the rule/suggestion/file. Check structured audit events and content-free logs.
- Performance tests should cover cold/warm rule sets, cache invalidation, bounded matching and exports, queue wait, P50/P95, unresolved/fallback-eligible rate, and provider cost fixed at zero. Set numeric acceptance targets only after a representative baseline; a synthetic document alone does not prove classification accuracy.
- Verify additive migration/backfill behavior against existing Phase 7 data and flag-off rollback. The absence of a destination workbook means no named-destination import-success test or compatibility claim in v1.

## Out of Scope

- Full accounting ledger, chart-of-accounts ownership, journal entries, debit/credit balancing, tax determination, VAT/WHT advice, payment execution, and direct authoritative posting to any accounting system.
- Automatic submission or sync to PEAK, FlowAccount, Express, Google Drive, or another destination; a user-controlled generic file is the v1 output.
- Import-ready destination adapters, user-authored column layouts/formulas, and cross-Organization mapping templates or learned rules.
- LLM-based classification, external AI transmission, fuzzy Vendor-name matching, uncalibrated numeric confidence, and auto-approval.
- Automatic line-item extraction/splitting of a mixed-purpose Document, retroactive rewriting of approved history, and automatic Task creation for every ambiguity.
- Treating a generic CSV/XLSX file as a legal accounting or tax record without independent professional review.

## Further Notes

- The settled interview choices and glossary are recorded in the Phase 8 decisions and PlaiFlow context documents. The versioning rationale is recorded in the accounting-mapping ADR; the draft STRIDE analysis and first-party destination-format research are companion notes. They are design inputs, not evidence that controls have shipped.
- The main unresolved product sign-off is the proposed exact `accounting_suggestions_v1` column list/order. A destination-specific template remains deferred until a real customer destination and official current workbook are supplied. Correction-history retention and measured latency/cost targets are release gates, not values to invent in this spec.
- The current Phase 7 implementation is narrower than its draft spec, including a 100-row synchronous export cap. Phase 8 must not assume that a promised durable export or a calibrated extraction-confidence model already exists. The current Phase 6 benchmark records `acceptance_passed=false` and zero human references; no OCR/classification accuracy claim follows from a synthetic smoke test.
