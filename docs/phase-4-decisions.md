# Phase 4 decision record

This records decisions reached during the Phase 4 design interview. Provider selection and external proof remain implementation prerequisites, not unsettled product decisions.

## Settled in round 1

- One Document unit is charged only after the file passes intake validation and its original is stored successfully. OCR and later extraction outcomes do not determine this charge.
- Drive intake begins with files explicitly selected by a user; continuous folder sync is outside this first scope. The existing PlaiFlow-to-Drive export direction remains.
- A private PlaiFlow object-storage copy is the canonical original for accepted Web, LINE and Drive documents. External source movement or deletion does not remove that original.
- Phase 4 CSV/XLSX document export covers an authorized document list and metadata, not extracted invoice/receipt values.
- Keep the Phase 3 commercial limits: Free 30, Starter 300, Business 1,000 Documents per Organization calendar month; reset at 00:00 on the first day in the Organization Timezone. Warn at 80% and 90%; hard-deny new Document Intake from every channel at 100% without automatic overage billing. Existing authorized reads and exports remain available.

## Existing constraints to preserve

- The current Organization Drive Connection is for customer-owned export with `drive.file` and server-held credentials. Drive intake must not silently broaden that connection or treat Drive metadata as tenant authorization.
- Entitlement and quota enforcement happen server-side after Organization authorization. They cannot vary by intake channel.
- Existing Phase 3 CSV/XLSX export templates and Task export remain separate from the new Document list template.

## Settled in round 2

- Phase 4 delivers intake, private original storage, an inbox/list and metadata export. OCR, field extraction and human correction are outside this first scope. A later processing failure does not reverse a completed Document unit.
- Accept PDF, JPEG and PNG up to 20 MiB per file. ZIP, Office files and native Google Workspace documents are outside this first scope. Validate bytes and detected type consistently for Web, LINE and Drive; filename and provider MIME are hints only.
- Deduplicate matching file content within one Organization across channels. Keep one Document, add a Document Source for each accepted origin and do not consume another Document unit. Retries and duplicate webhooks return the same outcome. Never deduplicate across Organizations.
- An active Member may submit through Web. LINE intake accepts linked Users in direct messages and authorized Members in a connected group. If the target Organization cannot be determined unambiguously, stop before storing an original and ask the sender to choose one.
- Owner/Admin selects individual Drive files through Google Picker from the Organization's authorized connection. Do not scan folders or subfolders and do not schedule sync in this first scope. The selected file can be outside the existing PlaiFlow export folder, subject to the connection's granted per-file access.
- At quota exhaustion, do not create a Document or retain the original bytes. Record a content-free rejection event, tell the sender through a safe channel, and notify Owner/Admin with the reset date. Do not reveal plan or document details into a LINE group.

## Settled in round 3

- Duplicate identity is an exact byte-content hash scoped to one Organization. A repeated file in a later month or at quota exhaustion resolves to its existing Document, records a new accepted origin, and uses no new monthly unit. Visually similar files with different bytes are distinct.
- Type, size and malware checks must pass before an original is accepted into private storage and one usage unit is completed. A failed scan or detected malware stays outside Document Usage; temporary bytes are removed within the bound settled in round 5.
- Inbox distinguishes a temporary receiving/checking attempt from an accepted Document. Accepted Documents are Available or Archived; rejected attempts stay in a separate intake history and are not Documents.
- Owner/Admin may see and download all Organization Documents. A Member may see only Documents they submitted or were assigned. A LINE group submission is Owner/Admin-only until responsibility is assigned. Every retrieval rechecks tenant and document authorization and uses short-lived signed access.
- Phase 4 has no automatic retention deletion. Owner/Admin may move a Document to Trash, restore it within 30 days, and permanently delete the original after that period. Deletion never refunds consumed monthly Usage.
- Do not create a Task for every new Document automatically. A User may create a Task from a Document. Alert on failed intake, approaching/exhausted quota, and unresolved Organization or assignee mapping for LINE files; do not reveal sensitive document details in a LINE group.

## Settled in round 4

- Matching content submitted while its Document is Archived adds a Document Source without changing its Archived state. A matching item in Trash must be restored by Owner/Admin before reuse; never restore it silently. After permanent purge, the same bytes can become a new Document and consume a new monthly unit.
- Document list CSV/XLSX export has one row per Document and a fixed versioned template: Document ID, display filename, detected type, byte size, status, first source channel, source count, authorized submitter/assignee display, received time and updated time. Exclude provider file IDs, LINE identifiers, signed URLs and file contents.
- Document list and export filters are received-date range, status, source channel, submitter, assignee and filename search. Export uses the same filters and shows a row count before confirmation.
- Only Owner/Admin may request a bulk Document export, for one authorized Organization. A Member may view and retrieve only their authorized individual Documents.
- Reuse Phase 3 export budgets: 0–5,000 rows synchronous, above 5,000 background only for Business; CSV at most 100,000 rows/256 MiB and XLSX at most 50,000 rows/128 MiB, with a 64 MiB export memory budget. Exports above the ceiling require narrower filters. Temporary export files expire after 24 hours.
- A manual Drive import selects at most 20 individual files per batch. Process asynchronously with persisted status and checkpoint per file. No scheduled sync, folder scan or account-wide change cursor belongs to this first scope.
- Each original retrieval authorizes Organization, Membership and Document visibility before issuing a signed URL lasting at most 5 minutes. Audit intake/rejection, original retrieval, Archive/Trash/Restore, export request and export download without recording sensitive filename, file bytes, provider IDs or bearer URLs in logs.

## Settled in round 5

- Web accepts at most 20 files per batch. Each file independently validates, reserves/completes/releases Usage and reports its outcome; one bad file does not roll back accepted siblings.
- LINE intake considers only image or file messages whose downloaded bytes validate as PDF, JPEG or PNG. Text, URL, video and audio messages are outside this scope. After signature verification, durably enqueue the event and acknowledge the webhook quickly; download media promptly in a worker. Provider redelivery and worker retries must be idempotent.
- If malware scanning is unavailable, keep temporary bytes encrypted and isolated while the attempt remains Checking, for at most 24 hours. Retry with a bound; then reject, purge temporary bytes, release any Usage reservation and notify the sender. A failed or unavailable scan never becomes an accepted Document.
- For a selected Drive file, recheck the Organization's authorized connection and selected file access at execution. Require the file revision to match selection; a changed, moved, deleted or inaccessible file fails that attempt without Usage and asks for reselection. Accepted PlaiFlow originals do not change with later Drive edits.
- Authorize each original retrieval before issuing a signed URL valid for at most 5 minutes. Phase 4 accepts the residual access window of an already-issued URL after Membership revocation; never log or expose the URL elsewhere.
- Async intake reports per-file state so the user may leave the page. Bound per-Organization and global worker concurrency; retry only transient network, rate-limit and server failures with jittered exponential backoff and provider retry hints. Stop retrying within 24 hours and expose a terminal outcome.
- No Thailand-only storage requirement is currently specified. For staging, use a Railway private Storage Bucket in Singapore (`asia-southeast1-eqsg3a`) and deploy a ClamAV `clamd` service in the same region, reachable only through Railway private networking. Keep staging and production buckets separate. Encrypt originals in the application before upload because Railway Buckets do not currently offer server-side encryption. The bucket's API `REGION` value comes from its credentials and may be `auto`. Verify the API region during deployment. Do not treat this as a production storage approval until object deletion and backup purge within 30 days are verified.

## Settled in round 6

- When a LINE sender has no unambiguous Organization, do not download or retain the original. Send a private Organization-selection path and ask the sender to resend the file after selection; never reveal the filename or Plan details in a group.
- Owner/Admin Document Dashboard shows Usage, limit, reset time, counts by Available/Archived/Checking/rejected state, source-channel summary and recent attempts/Documents. Member summaries and lists count only Documents they are authorized to see.
- When another Member submits identical bytes, that Member may see the shared original and only their own Document Source; Owner/Admin may see all sources. Deduplication does not disclose other submitters or source-channel details to Members.
- Usage Reservation belongs to the Organization calendar month at intake start in the Organization Timezone. Completion/release and retries keep that period across midnight; the same intake cannot complete twice or move between months.
- A mid-month downgrade immediately applies the new limit to new Documents without deleting or hiding accepted originals. Existing authorized reads and exports continue; an identical already-accepted file remains accessible without a new unit. New unique intake resumes at period reset or when an adequate Plan becomes effective.

## Settled in round 7

- Retain content-free rejected-attempt history for 90 days and safe security/export audit for 365 days. Do not retain rejected file bytes beyond the temporary-file cleanup deadline.
- A Task linked to a Trashed or permanently purged Document remains, but cannot open the original and shows a removed-document state without preserving sensitive filename text. Permanently purge the original after 30 days in Trash; provider backup copies must be purged within 30 more days, as settled in round 8.
- Trash is excluded from document-list export by default. Owner/Admin may explicitly filter Trash metadata before permanent purge; purged Documents never appear in list export.
- `Document Quota Exhausted` is distinct from structural `Plan Over Limit`. It blocks only new unique Document intake; other authorized work continues. A no-charge duplicate may still resolve to its existing Document.

## Settled in round 8

- The one-time Business Trial permits 100 accepted unique Documents across its entire 14 days, even across a calendar-month boundary. Prior Free usage does not reduce this Trial allowance. Documents accepted during Trial still contribute to the current calendar month's count when Trial ends; Free 30 then applies without deleting prior Documents.
- An Organization Timezone change does not move existing monthly Usage or Reservations. The new timezone applies to the next monthly period only.
- A file with a valid Usage Reservation made before a Plan downgrade may complete within the 24-hour retry window. New attempts use the new Plan limit immediately.
- Permanently purged originals must disappear from provider backups within 30 days after permanent deletion. Verify this capability with the selected storage provider before production use.

## Mandatory implementation invariants

- All channels use the same server-side Organization authorization, Document entitlement, atomic Usage reservation and finalization, exact-byte deduplication, MIME sniffing, malware boundary and size checks. A concurrent batch or retry cannot exceed a hard limit or increment Usage twice.
- LINE signature verification and connected-group/linked-User mapping precede intake authorization. Google Drive metadata is never tenant authorization; the server-held connection, selected-file grant, current Membership and Organization binding are checked independently.
- Originals and temporary files live in private object storage, never PostgreSQL or a public web directory. File handling and CSV/XLSX generation must stream within bounded byte and memory budgets; list filters require indexed, paginated queries.
- Document CSV protects every untrusted cell from spreadsheet formulas. XLSX writes untrusted values as text. Both formats are tenant-scoped and audited.
