# PlaiFlow Phase 5: Durable Jobs, Worker Protocol and Async Export Seam

Status: **ready-for-agent; product scope confirmed by the user on 2026-09-21**. This specification authorizes no implementation, migration, deployment, credential creation, OCR provider activation or external infrastructure change. It follows the confirmed [Phase 5 decision record](./phase-5-decisions.md) and [ADR-0032](./adr/0032-durable-generic-jobs-and-api-worker-boundary.md).

## Problem Statement

PlaiFlow already records background export work and processes several queues, but the worker currently reaches PostgreSQL directly and each workload owns parts of its own claim/retry behavior. This makes it harder to add OCR safely, recover abandoned work consistently, prove exactly-once outcomes under retries, and generate large exports without putting database credentials or whole tenant datasets inside a worker process.

Users need large exports to finish outside the request timeout, remain visible after leaving the page, and produce a secure temporary download. Future OCR needs the same durable claim, lease, heartbeat, retry and completion contract without committing Phase 5 to an OCR provider or extraction model.

## Solution

Introduce one Organization-scoped Durable Job model and one authenticated Internal Job API used by workers. The API owns persistence, tenant checks, claim locking, leases, retry classification, stale recovery, artifact authorization, audit and metrics. Workers receive short-lived, narrowly scoped access to one claimed attempt and never receive PostgreSQL credentials.

Phase 5 enables this protocol for large CSV/XLSX exports. It also defines and proves the provider-independent OCR job seam: an OCR job references an authorized Document, can be claimed and heartbeated through the same protocol, obtains temporary access only to that Document, and can report a provider-independent completion or failure. OCR execution, OCR provider integration, extracted field definitions and correction UI are excluded.

Exports of at most 5,000 authorized rows remain synchronous. Larger entitled exports are Durable Jobs. A worker reads an immutable authorized export snapshot through opaque cursor pages of 500 rows, streams the artifact to private object storage through temporary scoped access, and completes the job by attaching verified artifact metadata. The requester and current Owner/Admin can later obtain a 15-minute signed download URL. The artifact expires after 24 hours.

## User Stories

1. As an Owner/Admin, I want a large export request accepted quickly, so that I do not need to keep an HTTP request open while the file is generated.
2. As an Owner/Admin, I want exports of 5,000 rows or fewer to retain the current synchronous behavior, so that small exports remain immediate.
3. As an Owner/Admin, I want the same filters and format preserved when an export becomes asynchronous, so that the downloaded result matches the request I approved.
4. As an Owner/Admin, I want to see whether my export is Queued, Running, Completed, Failed or Cancelled, so that I know what action is possible.
5. As an export requester, I want to leave the page and return later, so that background processing does not depend on my browser session.
6. As an export requester, I want a completed artifact to show its row count, byte count, format and expiry, so that I can judge whether to download or rerun it.
7. As an export requester, I want a safe error code and retry guidance when a job fails, so that I can narrow filters or try again without seeing internal details.
8. As an export requester, I want to cancel my export while it is still Queued, so that unused work does not start unnecessarily.
9. As an export requester, I want cancellation rejected after processing begins, so that I never receive an ambiguous partial artifact.
10. As an export requester, I want a temporary download link, so that the export artifact is not publicly addressable.
11. As an export requester, I want the system to recheck my current Organization access before issuing a download, so that a stale job does not preserve revoked access.
12. As an Owner/Admin, I want to download another current Organization member's completed export when operationally necessary, so that team administration is not blocked by requester absence.
13. As a Member without export authority, I want other users' jobs and artifacts hidden, so that bulk data cannot leak through job history.
14. As a user whose Membership was removed, I want job status and artifact access denied immediately on subsequent requests, so that historical job ownership does not override current authorization.
15. As an Organization, I want every job isolated to my tenant, so that a worker request, cursor or artifact identifier cannot cross into another Organization.
16. As a worker, I want to claim a bounded batch of eligible jobs, so that I can process work without racing another worker.
17. As a worker, I want each claim to return a unique attempt and lease token, so that only my current attempt can change the job.
18. As a worker, I want to heartbeat long-running work, so that healthy processing is not reclaimed as stale.
19. As a worker, I want an expired lease to be reclaimable, so that a crashed process does not leave a job permanently Running.
20. As a worker, I want duplicate completion of my successful attempt to be a no-op, so that a network retry does not create another artifact or result.
21. As a worker, I want completion from an expired or replaced lease rejected, so that stale work cannot overwrite a newer attempt.
22. As a worker, I want transient failures rescheduled with bounded backoff, so that temporary provider or storage outages can recover.
23. As an operator, I want permanent and security failures to stop immediately, so that invalid or unauthorized work is not repeatedly executed.
24. As an operator, I want no job to retry more than five attempts, so that poison work cannot consume the queue indefinitely.
25. As an operator, I want idle workers to long poll or back off, so that an empty queue does not create hot database or API polling.
26. As an operator, I want queue wait, runtime, retries, stale recovery and failures measured by job kind, so that I can distinguish capacity problems from bad inputs.
27. As an operator, I want an alert when no worker heartbeat is healthy for two minutes, so that background processing failures are visible quickly.
28. As an operator, I want lifecycle audit records correlated by job, request and worker identity, so that incidents can be investigated without logging tokens or exported values.
29. As an operator, I want expired artifacts cleaned automatically, so that temporary exports do not accumulate indefinitely.
30. As an operator, I want completed and failed job metadata retained for 90 days, so that support can inspect recent outcomes after artifacts are gone.
31. As an operator, I want export audit retained for one year, so that artifact creation, authorization and download remain accountable.
32. As the platform owner, I want workers to operate without PostgreSQL credentials, so that compromise of a worker has a smaller blast radius.
33. As the platform owner, I want worker credentials scoped by environment and action, so that a staging token cannot operate production or perform unrelated API actions.
34. As the platform owner, I want active legacy export jobs migrated without losing status, so that deployment does not strand work already accepted.
35. As the platform owner, I want the previous export history readable during the migration window, so that rollback and support remain possible.
36. As the platform owner, I want schema cleanup deferred until the retention window and staging proof pass, so that rollback remains available.
37. As a future OCR worker, I want to claim an OCR job through the same Durable Job protocol, so that OCR does not invent a second queue lifecycle.
38. As a future OCR worker, I want temporary read access only to the referenced Document, so that the worker cannot browse Organization storage.
39. As a Document owner, I want an OCR retry to remain tied to the same Document and job, so that retries do not create another Document or consume Document Intake Usage.
40. As an operator, I want OCR failures represented by safe provider-independent codes, so that switching OCR providers does not change the public job lifecycle.
41. As a product owner, I want Phase 5 to stop at the OCR protocol seam, so that provider selection, extraction rules and review UI can be designed separately.

## Implementation Decisions

### Durable Job domain

- A `Durable Job` is an Organization-scoped request for background processing. One generic job model supports the allowlisted kinds `export` and `ocr`; kind-specific payload and result records remain behind the job service boundary.
- Canonical status transitions are `Queued → Running → Completed | Failed`. `Queued → Cancelled` is allowed. No other cancellation transition exists in Phase 5.
- A job carries an immutable Organization, kind, requester or trusted server actor, safe payload reference, creation time, availability time and attempt limit. Mutable lifecycle data includes current status, attempt count, lease expiry, safe failure code and terminal timestamps.
- A claim creates a unique `attempt_id` and random `lease_token`. Only a hash of the lease token is stored. Lifecycle calls require job ID, attempt ID and the current token.
- A two-minute Job Lease begins at claim and is renewed by heartbeat every 30 seconds. Renewal never changes attempt identity. An expired Running job can be reclaimed with a new attempt and token.
- Reclaiming an expired lease consumes an attempt. No job may exceed five claimed attempts. Exhaustion moves the job to Failed with a safe terminal code.
- Completion is conditional on the current Running status, attempt and unexpired lease. Repeating the successful completion for that same attempt returns the existing terminal result without a second write. A different or stale attempt receives a conflict response and cannot attach results.
- Terminal jobs are immutable except for retention cleanup metadata. Phase 5 does not add a separate dead-letter status.

### Internal Worker Protocol

- The Internal Job API is the single high-level seam for worker interaction. Workers do not connect to PostgreSQL or call tenant-facing session endpoints.
- Worker authentication uses short-lived bearer credentials issued for one environment, worker identity and allowlisted scopes. Minimum scopes are claim, heartbeat, read-input, upload-artifact, complete and fail; a worker receives only the scopes required by its supported kinds.
- Authentication and scope checks occur before job lookup to avoid leaking job existence. Tokens, lease tokens, signed URLs and raw payload data never appear in logs, metrics or audit metadata.
- Claim accepts an allowlisted set of job kinds, a maximum batch size of 10 and an optional wait of at most 15 seconds. The service uses `SKIP LOCKED` or equivalent transactional locking, orders eligible work by availability and age, and returns no more than the requested bound.
- Empty claim calls use long polling up to 15 seconds or client backoff from 1 to 15 seconds with jitter. The protocol must not require hot polling.
- A claim result contains job ID, kind, Organization ID, attempt ID, lease expiry and a minimal kind-specific reference. It does not contain database credentials, object-store master credentials, another tenant's metadata or unrestricted queries.
- Heartbeat renews only the caller's current unexpired lease. It returns the new lease expiry and is idempotent within an attempt. A heartbeat cannot revive a terminal, cancelled, stale or replaced attempt.
- Fail accepts a safe error code and diagnostic correlation ID. The API, not the worker, owns the allowlist that classifies an error as transient, permanent or security-sensitive.
- Transient failures are rescheduled with jittered exponential backoff and provider retry hints within the five-attempt ceiling. Permanent failures become Failed immediately. Security failures become Failed immediately and emit a security audit event.
- Complete accepts only kind-appropriate result metadata. The API verifies the current attempt, tenant binding, object/result reference and declared bounds before making the terminal transition.

### Large export protocol

- The tenant-facing export route performs current session, Organization, role, entitlement, filter and row-ceiling checks before creating a job. Client-supplied Organization IDs, row counts and entitlement claims are never authoritative.
- Exports of 0–5,000 rows remain synchronous. Exports above 5,000 rows require the existing background-export entitlement and become `export` Durable Jobs.
- Preserve Phase 4 ceilings: CSV at most 100,000 rows or 256 MiB; XLSX at most 50,000 rows or 128 MiB; generation has a 64 MiB memory budget. Requests above a ceiling fail before enqueue and ask for narrower filters.
- The export job freezes its authorized Organization, requester, versioned template, format, normalized filters, authorized row count and `data_as_of` snapshot boundary. Later client input cannot mutate the job.
- A claimed worker reads rows only through the Internal Job API. The row endpoint requires the current lease and returns an opaque cursor, no more than 500 rows, the stable snapshot boundary and an end marker.
- An export cursor is opaque, job-bound, Organization-bound and filter-bound. It cannot be replayed for another job or used after the attempt loses its lease.
- Row authorization and tenant filtering occur in the API/database transaction. The worker cannot choose a table, SQL expression, Organization or arbitrary field list.
- CSV generation streams rows, protects every untrusted cell from spreadsheet-formula execution and uses the versioned export header. XLSX generation writes untrusted values as text through a bounded-memory streaming writer.
- The API issues a short-lived signed upload grant for one server-chosen object key under the current job. The worker never chooses an arbitrary bucket/key and never receives bucket master credentials.
- Completion verifies that the uploaded object belongs to the job and records verified format, row count, byte count, checksum and expiry. A partial or unverified upload is not an Export Artifact and is cleaned asynchronously.
- A completed Export Artifact is private, scoped to one Organization and expires after 24 hours. File blobs never enter PostgreSQL.
- The tenant-facing job status endpoint returns safe lifecycle and artifact metadata only. It never returns object keys, lease information, worker identity, signed upload URLs or row data.
- A download request rechecks current Membership and Organization. The original requester or a current Owner/Admin may receive a signed download URL valid for 15 minutes. Other users receive a non-enumerating denial.
- A queued requester or current Owner/Admin may cancel the job. Running and terminal jobs cannot be cancelled. Cancellation is atomic with claim so exactly one transition wins.

### OCR protocol seam

- An OCR job references one existing Document ID, Organization ID and immutable input object version/checksum selected by the trusted server. It does not copy a client-provided object key into the job.
- OCR creation requires current Document authorization and the future OCR entitlement/usage decision at the server boundary. Phase 5 does not define OCR pricing, quota amount or user-facing trigger.
- A worker with OCR scope may claim and heartbeat OCR jobs through the same protocol. The input endpoint verifies the current attempt, Organization and Document binding before returning a short-lived signed read grant for that original only.
- OCR completion is provider-independent. The generic protocol can accept a versioned result reference, safe outcome metadata and checksum, but Phase 5 does not define extracted fields, confidence scores, page structures, correction state or accounting behavior.
- OCR retry is idempotent for the same job and does not create a Document, Document Source or Document Usage unit. Document Intake remains governed by Phase 4.
- No OCR provider SDK, model, prompt, webhook, callback, parser, user interface or production worker handler is implemented under this specification.

### Persistence, migration and compatibility

- Add the generic jobs schema before switching any producer or worker. Required query paths support eligible claim ordering, Organization history, requester history, terminal retention and stale-lease recovery.
- Use database constraints for allowed kind/status values, nonnegative attempt counts, terminal timestamp consistency and Organization-bound result references. Tenant authorization remains enforced in application and PostgreSQL boundaries where applicable.
- Existing active background export rows in Queued or Running state migrate to generic export jobs with stable identifiers and safe normalized payloads. A legacy Running row without a valid new lease becomes reclaimable rather than trusted as actively owned.
- Preserve a compatibility read path for historical export status while the old table remains. Do not dual-execute the same export from both queue models.
- Producers switch only after the new schema and API can read/write jobs. Workers switch after protocol contract tests pass. Destructive removal of the old export table/read path is a later migration after the 90-day metadata window and staging proof.
- Up/down migration behavior must preserve readable export history and must not delete completed artifact references. Rollback stops new generic job creation, rolls back workers before schema cleanup, and lets valid leases finish or expire.

### Storage, cleanup and retention

- Private object storage holds Export Artifacts and any future OCR-generated file. PostgreSQL stores metadata and references only.
- Cleanup is idempotent background work. It removes expired export objects after 24 hours, abandoned partial uploads, and expired temporary grants without changing completed audit history.
- Job metadata is retained for 90 days. Audit records for creation, claim/reclaim, retry, completion/failure, artifact creation, signed URL issuance, download, cleanup and authorization failure are retained for 365 days.
- Cleanup failure is retryable and observable. A missing already-expired object is treated as successful cleanup.

### Security

- Worker credentials follow least privilege, are environment-specific, short-lived and server-validated. Production and staging credentials are never interchangeable.
- Workers have no database credentials and no unrestricted object-store credentials. Input and upload access use job-bound signed temporary grants.
- Every job, cursor, input grant, upload grant, artifact and audit record is bound to one Organization. A worker-provided identifier is never sufficient tenant authorization.
- User-facing job ownership never overrides current Membership. The requester and current Owner/Admin download policy is re-evaluated for every signed retrieval request.
- Failure responses and logs use safe codes. Do not log exported cell values, OCR content, sensitive filenames, object keys, signed URLs, bearer tokens or lease tokens.
- CSV formula protection and XLSX text handling remain mandatory for asynchronous output.
- Internal endpoints require transport security, request-size bounds, rate limits and replay-resistant token validation.

### Performance and fairness

- Claim at most 10 jobs per request and use transactional skip-locked claiming. No worker scans the full queue.
- Process work with bounded global concurrency and a lower per-Organization concurrency so one tenant cannot monopolize workers.
- Export row pages contain at most 500 rows. Cursor pagination and streaming keep the worker within the 64 MiB memory budget.
- Database indexes support available/status/kind claim ordering and Organization/status/time history without full-table scans.
- Queue polling uses long poll or bounded jittered backoff. Heartbeat cadence is 30 seconds and must not become a per-row operation.
- Artifact upload and original download stream bytes. No file blob or whole export dataset is buffered in PostgreSQL or worker memory.

### Observability

- Record queue wait, runtime, attempts, retries, stale reclaims, terminal failures by safe kind/error code, export rows, export bytes, artifact downloads and worker heartbeat age.
- Metrics must avoid unbounded labels: do not label by Organization, User, Job, Document, filename, token or cursor.
- Alert when no healthy worker heartbeat has been observed for two minutes, queue wait exceeds the operating target, stale reclaim rises materially, cleanup falls behind or terminal failures spike by kind.
- Correlate safe structured logs and audit with request ID, job ID, attempt ID and worker identity. Never include secrets or exported/OCR content.

## Testing Decisions

- Test at the highest stable seam: the authenticated Internal Job API plus the tenant-facing export status/download behavior. Persistence and storage are verified through externally visible state transitions rather than tests coupled to individual SQL statements.
- Reuse the existing HTTP route-test pattern for authorization, response contracts and user-visible outcomes; reuse the existing worker fake-store pattern only where a narrow client retry loop needs deterministic time/error control.
- Add migration integration coverage against PostgreSQL for constraints, `SKIP LOCKED` concurrency, stale lease reclaim and compatibility reads. These are database guarantees and cannot be proven by mocks.
- Prove that two concurrent claimers receive disjoint jobs and never exceed the batch bound.
- Prove heartbeat renewal for the current attempt and rejection for expired, replaced, cancelled and terminal attempts.
- Prove stale recovery creates a new attempt/token, consumes an attempt and rejects completion from the old lease.
- Prove duplicate completion of the successful attempt returns the original result without a second artifact, audit completion or terminal transition.
- Prove transient failure schedules bounded backoff, permanent failure terminates immediately, security failure audits immediately, and a sixth claim is impossible.
- Prove cancellation wins only while Queued and races safely with claim.
- Prove a worker token cannot use an ungranted scope, another environment, another job kind or an expired credential.
- Prove a cursor, lease token, signed input grant and upload grant cannot cross jobs or Organizations.
- Prove export filters and `data_as_of` remain immutable across every chunk and that the concatenated output contains each authorized row exactly once.
- Prove CSV formula cells are neutralized and XLSX values are text in asynchronous output, matching synchronous export safety.
- Prove CSV and XLSX ceilings, 500-row chunking and 64 MiB memory budget through a representative large-export test without asserting internal buffer implementation.
- Prove completion rejects missing, partial, oversized, wrong-format, wrong-key and checksum-mismatched uploads.
- Prove only the requester and current Owner/Admin can obtain a download, and revocation after completion blocks the next retrieval request.
- Prove signed URLs expire after 15 minutes, artifacts are deleted after 24 hours, missing expired objects clean idempotently, metadata remains for 90 days and audit remains for 365 days.
- Prove an OCR claim can receive access only to its referenced authorized Document, retry without creating Document Usage, and report provider-independent completion/failure. Do not test OCR extraction quality or provider behavior.
- Prove metrics and audit occur once for meaningful lifecycle transitions and contain no token, URL, row value, OCR content or sensitive filename.
- Run existing Go tests, vet and builds; run web lint, typecheck, tests and build. Migration, object storage, signed access and concurrent claim behavior require integration/smoke evidence in addition to local unit checks.

## Out of Scope

- OCR provider selection, credentials, SDK integration, model choice or production OCR execution
- OCR field schema, invoice/receipt extraction, confidence thresholds, validation rules, human correction and accounting workflows
- OCR pricing, quota amount, billing or product UI beyond the generic status seam
- A general workflow engine, DAG dependencies, priorities, cron scheduler, arbitrary user-defined jobs or cross-job orchestration
- Replacing existing inbound webhook, LINE delivery, Drive intake or notification queues unless separately migrated later
- Cancelling Running jobs, partial artifact delivery, pause/resume, manual attempt editing or a dead-letter queue UI
- Public object storage, permanent download URLs, storing blobs in PostgreSQL or giving workers master storage credentials
- Custom export columns/templates, exports beyond existing CSV/XLSX ceilings or whole-dataset in-memory generation
- Destructive removal of the legacy export table during the first rollout
- Production deployment, secret provisioning, external service creation and live migration execution under this spec-only step

## Further Notes

- The primary test seam is the Internal Job API. It keeps claim/lease/idempotency/security behavior behind one boundary and lets OCR and export share protocol behavior without sharing business payload internals.
- Current code already has `export_jobs`, worker heartbeats, `SKIP LOCKED` claim patterns, bounded worker loops, export audit and synchronous/background threshold logic. Phase 5 extends these behaviors while moving worker persistence access behind the API.
- The existing worker currently opens PostgreSQL directly. Treat removal of its database credentials as a migration and deployment change, not a refactor that can be inferred complete from unit tests.
- `ready-for-agent` here means the local specification is detailed enough for ticket decomposition. It does not authorize implementation, OCR activation, migration execution or deployment.
- No active issue-tracker configuration exists in this workspace, so this specification is published locally only. Run `/setup-matt-pocock-skills` before publishing it as an external tracker issue.
