# PlaiFlow Phase 4: Multi-Channel Document Intake and Document List Export

Status: **product scope confirmed by the user on 2026-09-20**. This document records product decisions made during the Phase 4 design interview. It does not claim implementation, provider provisioning, migration, live OAuth, malware-scanner readiness or end-to-end proof. [Decision record](./phase-4-decisions.md) captures the individual choices.

## Problem Statement

Teams receive business files through Web, LINE and Google Drive but cannot put them in one authorized Organization inbox, track where each file came from, see intake outcomes, or export a filtered document list. Retries and cross-channel duplicates must not consume quota twice or reveal one tenant's files to another.

## Solution

An authorized Organization can receive business-document originals from Web, verified LINE messages and explicitly selected Google Drive files; keep a private PlaiFlow-owned canonical copy; find and retrieve its authorized Documents; and export an authorized metadata list as CSV or XLSX. Phase 4 adds no OCR, extracted fields, image transcription, accounting ledger, scheduled Drive sync or general Drive folder scan. Existing Task, Business Contact, LINE Login and Drive export behavior remain separate.

The first intake supports PDF, JPEG and PNG up to **20 MiB per file**. Native Google Workspace files, Office documents, ZIP, text, video and audio are outside this scope. A filename, provider MIME value, Drive parent folder or LINE message type never substitutes for byte-level validation.

## User Stories

1. As a Member, I want to upload one PDF, JPEG or PNG from Web, so that I can put a business file into my Organization inbox.
2. As a Member, I want to upload up to 20 files with a separate result for each, so that one invalid file does not hide successful uploads.
3. As a Member, I want to see Receiving, Checking and Rejected attempts, so that I know whether to wait or try again.
4. As a Member, I want clear type, size, scan and quota rejection messages, so that I can correct the problem without guessing.
5. As a linked LINE User, I want to send one supported file in a direct message, so that it enters my selected Organization.
6. As an authorized Member of a connected LINE group, I want to send one supported file in that group, so that Owner/Admin can process it.
7. As a LINE sender with more than one possible Organization, I want a private selection path and a resend instruction, so that my file is never stored under the wrong Organization.
8. As a LINE sender, I want expired or unavailable message content reported safely, so that I know to resend it.
9. As an Owner/Admin, I want to choose individual Google Drive files from my Organization's authorized connection, so that I can import the intended files without a folder scan.
10. As an Owner/Admin, I want each selected Drive file to show its own progress and result, so that I can retry only failures.
11. As an Owner/Admin, I want a changed, moved, deleted or inaccessible Drive file to fail safely, so that an unintended revision is never imported.
12. As a Document viewer, I want to see each accepted file's channel and permitted provenance, so that I know where it entered PlaiFlow.
13. As an Owner/Admin, I want repeated selection of the same Drive revision to resolve to the same outcome, so that a retry does not create another Document or Source.
14. As an Owner/Admin, I want a later Drive revision to be treated according to its validated bytes, so that changed content is not silently mistaken for the original.
15. As a Member, I want identical bytes from Web, LINE or Drive to resolve to one Document within my Organization, so that duplicates do not consume another unit.
16. As an Owner/Admin, I want monthly used, limit, remaining and reset time, so that I can plan intake before quota runs out.
17. As an Owner/Admin, I want warnings at 80% and 90% and a clear exhaustion state at 100%, so that I can act before new unique intake stops.
18. As a sender at the limit, I want a safe rejection of a new unique file but access to an existing duplicate, so that quota does not block already accepted content.
19. As a Member, I want an inbox and paginated list containing only Documents I submitted or was assigned, so that I can find my authorized work.
20. As an Owner/Admin, I want to see all Organization Documents, attempts, statuses and source summaries, so that I can manage the team inbox.
21. As an Owner/Admin, I want to assign a group-origin Document, so that the responsible Member can access it without exposing it to the whole group.
22. As an authorized viewer, I want to open an original through a short-lived link, so that I can inspect the file without making storage public.
23. As an Owner/Admin, I want to Archive, Trash and restore Documents within 30 days, so that I can organize or recover them.
24. As a task owner, I want a linked Task to remain understandable after its Document is removed, so that the work history is not lost.
25. As a User, I want to create a Task from a Document when needed, so that the team can act on a file without an automatic Task for every intake.
26. As an Owner/Admin, I want a filtered Document list with date, status, channel, submitter, assignee and filename search, so that I can find the right records.
27. As an Owner/Admin, I want to preview the export row count and download filtered metadata as CSV or XLSX when entitled, so that I can use the list outside PlaiFlow.
28. As an Owner/Admin, I want Trash excluded from export unless I explicitly include it, so that removed records do not appear unexpectedly.
29. As a Member, I want another Member's sources, filenames, counts and exports hidden, so that my access does not reveal their files.
30. As an operator, I want safe job status, retry, quota and audit signals, so that I can diagnose failures without reading file contents or credentials.

## Implementation Decisions

### Domain and lifecycle

- `Document` is one accepted original within one Organization. Acceptance requires allowed type, size, malware result and a durable private original copy. OCR or later extraction never determines acceptance.
- `Document Source` records one accepted origin from Web, LINE or Drive. One Document can have multiple Sources without consuming another quota unit. Retries of the same origin do not create another Source.
- `Document Intake Attempt` is a per-file request and its progress/rejection record. An attempt is not a Document until acceptance. Attempts can be Receiving, Checking, Accepted or Rejected; accepted Documents can be Available, Archived or in Trash.
- Exact byte content, identified by a server-computed digest and scoped to Organization, defines duplication. Visually similar files with different bytes are separate. Never match or reveal duplicates across Organizations.
- An Archived duplicate gains a new Source and stays Archived. A duplicate in Trash requires Owner/Admin restoration; it is never silently restored. Content received after permanent purge is a new Document and a new Usage unit.
- A Member who submits a duplicate sees the original and only their own Source. Owner/Admin sees all Sources. A LINE group submission is visible to Owner/Admin until a responsible Member is explicitly assigned.
- A Document can be linked to a Task by deliberate user action. Intake does not create a Task per file.

### Plan, Usage and quota

| Plan | Accepted Documents per Organization calendar month |
|---|---:|
| Free | 30 |
| Starter | 300 |
| Business | 1,000 |

The one-time Business Trial has a separate cap of **100 accepted unique Documents across its full 14 days**, even across a calendar-month boundary. While Trial is active, its 100-Document cap is the intake limit and the 80%/90% warnings use that cap. Documents accepted before Trial do not reduce it. Trial Documents still contribute to their Organization calendar month, so at Trial expiry the current month's total is compared with the effective Free limit of 30. No prior Document is removed.

The monthly period begins on the first day at 00:00 in the Organization Timezone. Bind each attempt's Usage Reservation to the period at intake start; retries, scanner delay and finalization keep that period. A timezone change affects only the next monthly period. A valid reservation made before a Plan downgrade may finish within the 24-hour retry window; new attempts use the new Plan immediately. Warn Owner/Admin at 80% and 90%, and deny a new unique Document at 100% without overage billing. A valid duplicate resolves to its existing Document, even at 100% or after a Plan downgrade, without a new unit. This may require a bounded temporary download and hash before the server can know whether the file is a duplicate.

Charge at most one unit for an accepted unique Document. Failed validation, malware detection, scanner outage past its deadline, missing provider access, quota rejection and retries consume no completed unit. A later processing failure and Archive/Trash/deletion do not refund an accepted unit. A downgrade does not delete existing Documents; it blocks new unique intake until reset or an adequate Plan takes effect. Existing authorized read and export paths remain available. `Document Quota Exhausted` blocks only new unique intake; it is distinct from structural `Plan Over Limit` and does not stop unrelated authorized work.

Every channel follows the same server-side order: authenticated/verified origin, Organization and Membership authorization, entitlement and period binding, bounded temporary acquisition, type/size/malware validation, exact-byte duplicate check, atomic Usage Reserve/Complete or Release, durable original promotion and Document/Source finalization. Concurrent workers and retries must be safe under a unique Organization/content constraint and stable channel idempotency key; object-store/DB partial failures need cleanup and reconciliation. A completed unit is never recorded without a retrievable original.

### Intake channels

#### Web

An active Member, Admin or Owner may select up to 20 files per batch. Each file has independent progress and outcome; one rejection does not roll back accepted siblings. Stream into isolated private temporary storage with byte limits. Final status, quota and duplicate outcomes come from the server rather than browser-provided Plan or file metadata.

#### LINE

Verify the webhook signature before parsing, deduplicate webhook event and message IDs, persist an intake job, and acknowledge quickly. A worker promptly retrieves image/file message content with the server-held LINE channel token, then applies the common validation pipeline. Text, URL, audio and video messages do not create attempts. A direct-message sender must be a linked User; a group sender must be an authorized Member in a connected group. If Organization mapping is ambiguous, do not download or retain the original: privately request Organization selection and ask the sender to resend the file afterward. Do not disclose filenames, Plan details or document status in a group. If LINE content has expired, end the attempt and request a resend rather than retry indefinitely. [LINE content and webhook behavior](https://developers.line.biz/en/docs/messaging-api/receiving-messages).

#### Google Drive

An Owner/Admin selects up to 20 individual files through Google Picker using the Organization's authorized Drive connection. The selected file can be outside PlaiFlow's existing export folder; this is not whole-Drive listing, folder import, subfolder traversal, change-feed sync or scheduled sync. Keep a per-file job and checkpoint, then asynchronously check current connection, tenant binding, grant and selected revision before download. A changed, moved, deleted or inaccessible file ends that file's attempt without Usage and asks for reselection. After acceptance, later Drive changes do not alter the PlaiFlow copy. Drive tokens stay encrypted and server-side; browser-selected metadata is never tenant authorization. The existing PlaiFlow-to-Drive export direction remains. [Google `drive.file` guidance](https://developers.google.com/workspace/drive/api/guides/api-specific-auth).

Drive Source provenance is a server-verified snapshot: Organization and Connection identity, provider file ID and selected revision, selecting Membership, selection and acceptance times, and safe display metadata such as filename, provider-declared type and size. The detected type, measured size and content digest come only from downloaded bytes. Provider IDs and revision identifiers remain private to Owner/Admin and operators with a support need; they do not appear in Member views or list exports. A later provider rename, move or deletion never rewrites accepted provenance or the canonical original.

“Sync” in this phase means processing or explicitly repeating a selected-file import. Each file has a durable attempt/job with selection, authorization, download, validation and finalization checkpoints and a terminal accepted/rejected result. Resume from a persisted checkpoint only when the current Connection, Membership, file grant and revision still pass authorization; a checkpoint is never proof of current access. The stable origin identity is Organization + Connection + provider file ID + selected revision. Repeating that identity returns its prior result without another Source or Usage unit. A newly selected revision is a new origin: identical bytes add a Source to the existing Document, while different validated bytes create a new Document subject to current quota. If the file moved, was deleted, changed revision or lost its grant before acceptance, fail without Usage and require a new selection. There is no folder cursor, Drive change-feed cursor, polling job or scheduled reconciliation in Phase 4.

### Malware, storage, retention and retrieval

All three channels use the same content detection, 20 MiB limit and malware boundary. A scanner outage leaves the attempt Checking while encrypted, isolated temporary bytes are retried for at most 24 hours; then reject, purge temporary bytes and release Usage. An unsafe file never becomes an accepted Document. Stream files and downloads; never store blobs in PostgreSQL or a public directory.

Accepted originals live in private object storage as the canonical copy. Choose and record a production storage region close to the API before deployment; no Thailand-only requirement or provider has been chosen. There is no automatic deletion of Available/Archived Documents in Phase 4. Owner/Admin may Trash and restore for 30 days; permanent purge follows, without Usage refund. Provider backup copies must disappear within 30 days after permanent purge, and this capability must be verified before production use. A linked Task remains after Trash/purge but shows a removed-document state without retaining a sensitive filename or opening the original. A retrieval first rechecks Organization, Membership and Document visibility, then issues a signed URL valid at most 5 minutes. Phase 4 accepts that an already-issued URL may remain usable for up to five minutes after access revocation; bearer URLs and object keys must not appear in logs or analytics.

### Document Dashboard and Inbox

Owner/Admin sees monthly used/limit/remaining/reset time, counts for Available, Archived, Checking and rejected attempts, source-channel summary and recent activity. Member sees only Documents and counts within their own visibility. The list uses stable cursor pagination and indexed received-date/status/source/submitter/assignee filters, plus bounded filename search. Inbox exposes per-file status, safe error/retry guidance and an explicit action to create a linked Task. Notifications cover failed intake, 80%/90%/100% quota milestones and unresolved LINE mapping; they do not broadcast sensitive details to a LINE group.

### Document list CSV/XLSX export

Only Owner/Admin may bulk export an authorized Organization's Document metadata. CSV is available on all Plans; XLSX follows the existing Starter/Business entitlement, including Business Trial. One row represents one Document, regardless of Source count. A fixed, versioned template contains Document ID, display filename, detected type, byte size, status, first source channel, Source count, submitter/assignee display names, received time and updated time. Exclude raw contents, provider file IDs, LINE identifiers, signed URLs, object keys and raw audit events. Export uses the visible list's received-date, status, source, submitter, assignee and filename filters, and presents a row count before submission. Trash is excluded by default but may be explicitly filtered by Owner/Admin before purge; purged Documents never appear.

Reuse the Phase 3 budgets: up to 5,000 rows synchronous; above that only Business can request a background job. Hard ceilings are 100,000 rows/256 MiB for CSV and 50,000 rows/128 MiB for XLSX, with a 64 MiB export memory budget. A larger request must narrow filters. Stream CSV, write XLSX untrusted values as text, and neutralize spreadsheet formulas in every untrusted CSV cell. Temporary export files expire after 24 hours. Authorize each download and audit request, rejection, completion and download without sensitive data or bearer links.

### Reliability and performance

- Drive import and expensive file checks run outside the request path. Per-file checkpoints and idempotency keys allow restart without repeated Usage or Sources. No Drive account cursor exists while folder sync is out of scope.
- Bound worker concurrency per Organization and globally. Honor provider retry hints and use jittered exponential backoff for transient network/429/5xx failures; do not retry permission loss, unsupported type, malware or changed revision. End attempts within 24 hours. [Google Drive rate-limit guidance](https://developers.google.com/workspace/drive/api/guides/limits).
- Fast LINE webhook acknowledgment follows verified durable enqueue, not media download or scanning. Persist enough safe identity to recognize provider redelivery without storing raw webhook bodies in audit.
- Keep list queries paginated and status/date filters indexed. Stream originals and exports with byte and memory ceilings; apply request/batch/rate limits so a full-quota tenant cannot force unbounded duplicate hashing.
- Retain content-free rejected-attempt history for 90 days and safe security/export audit for 365 days. Audit safe transitions for intake, rejection, retrieval, Archive/Trash/Restore and export. Never log file bytes, sensitive filenames, provider tokens/IDs, signed URLs or other tenants' data.

### Observability

- Correlate request, attempt, job, Document, Source and Usage reservation through internal opaque identifiers without logging file content, sensitive filenames, LINE IDs, Drive IDs, tokens or signed links. Expose a safe user-facing attempt/job ID for support.
- Count attempts, accepted unique Documents, no-charge duplicates, rejected files by safe reason category, quota denials, webhook redeliveries, Drive retries/rate limits, scanner outages, stalled jobs, storage promotion/reconciliation failures, export requests/failures and cleanup backlog. Partition operator views by environment and Organization without cross-tenant data leakage.
- Record duration and queue age for webhook acknowledgment, provider download, validation, object promotion, inbox queries and export jobs. Alert on sustained queue age, repeated terminal failures, incomplete Usage/object reconciliation, missed temporary-file cleanup and missed purge/backup deadlines.
- Audit who selected a Drive file, who requested or downloaded an original or export, and state-changing actions with Organization and safe result. Retain the safe security/export audit for 365 days; do not treat operational metrics as an alternate content store.

## Testing Decisions

The primary test seam is the authenticated, Organization-scoped HTTP boundary with real PostgreSQL transactions, followed by one bounded worker cycle and an authorized read or export. This reuses the current webhook-to-PostgreSQL-to-worker, Organization authorization, Drive connection, Plan gate and CSV/XLSX export tests. Narrow doubles belong only at external LINE/Google, malware-scanner and private object-storage boundaries. Tests assert user-visible outcomes, persisted tenant isolation and Usage, not private helper calls or checkpoint implementation details. Actual provider, credential, storage and backup behavior needs separate staging proof.

The first tracer cases for later `/to-tickets` are one Web upload, one LINE direct file, one connected-group LINE file, one authorized selected Drive file through the shared Document pipeline, repeated selection of one Drive revision, duplicate webhook delivery, invalid file rejection, authorized inbox retrieval and filtered CSV/XLSX export. These are validation slices, not tickets created by this spec.

Acceptance examples:

1. A Web file, LINE redelivery and Drive selection carrying identical bytes for one Organization result in one Document, distinct authorized Sources and one completed Usage unit; another Organization gets its own Document.
2. Twenty simultaneous unique 20 MiB files cannot cross a hard quota; accepted files have originals and exactly one unit each, while rejected files have no completed Usage.
3. A duplicate at 100% or in a later month returns the existing Document without a new unit; a Trash match cannot restore itself.
4. A failed malware scan, revoked Drive grant, changed Drive revision or expired LINE content leaves no accepted Document or completed Usage.
5. A Member cannot list, retrieve or infer another Member's Document or Source through filters, counts, signed links, exports or guessed IDs; Owner/Admin remains tenant-scoped.
6. A Document list CSV/XLSX export contains only authorized metadata, resists formula injection, stays within row/byte/memory limits and has an audit trail without content or bearer links.
7. Browser and worker tests cover 375/430/768/1024/1440px Dashboard layouts, pagination, upload outcomes, quota warnings and recovery states; real provider and storage flows require separate staging proof.

8. A repeated Drive selection for the same Organization, Connection, file ID and revision returns its prior result without another Source or unit; a changed revision is separately validated and its bytes determine whether it is a new Document.
9. A worker restart at each Drive checkpoint resumes safely, rechecks current access and cannot complete a stale or revoked selection; there is no account-wide Drive scan or cursor.
10. Monitoring exposes stalled attempts, scanner outage, quota denials, provider throttling, failed promotion/reconciliation and missed purge deadlines without credentials, filenames, bytes or bearer links.

## Out of Scope

- OCR, extracted financial fields, AI transcription, human correction and accounting ledger work.
- Automatic Drive folder/subfolder discovery, scheduled sync, change-feed cursor, full rescans, Shared Drive and native Google Workspace file conversion.
- ZIP, Office, text, video and audio intake; public object access; per-file automatic Task creation; export of raw file contents or provider identifiers.
- Implementation, migration, provider provisioning, deployment, issue creation and ticket splitting in this spec-only step.

## Further Notes

The product scope was confirmed on 2026-09-20. This spec refines the selected-file Drive checkpoint and provenance contract and records the preferred later ticket slices without changing the agreed no-scheduled-sync boundary. Existing Drive export uses a customer-owned connection; its commercial entitlement and consent rules remain in force. Google documents describe `drive.file` with Picker as per-file access, and LINE documents advise signature verification, asynchronous webhook work and handling redelivery/content expiry. [Google authorization](https://developers.google.com/workspace/drive/api/guides/api-specific-auth), [LINE webhooks](https://developers.line.biz/en/docs/messaging-api/receiving-messages).

Implementation prerequisites and proof:

- Verify the selected provider can remove backup copies within 30 days after permanent Document purge, and that the Trial/monthly Usage transitions are testable across timezones and Plan changes.
- Select a malware scanner and private object-storage provider/region; document cost, encryption, temporary and backup purge controls. Confirm Google Picker access with the actual Drive OAuth client and re-consent flow. These are not yet provisioned or verified.
- Prove the changed Plan/Usage rule against the existing Phase 3 structural `Plan Over Limit` implementation; do not let document-quota exhaustion accidentally impose the structural mutation lock.
- Prove tenant isolation, concurrent hard-quota reservations, duplicate webhooks, cross-month jobs, Drive grant/revision changes, scanner outage, signed retrieval, retention cleanup, CSV injection and bounded export memory with focused tests. Run separate staging provider and storage checks before claiming the external workflow works.
