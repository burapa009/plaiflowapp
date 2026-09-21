# Phase 5 decision record

Status: Confirmed by the product owner on 2026-09-21.

This records the shared decisions from the Phase 5 Durable Jobs, Worker Protocol and asynchronous export design interview. It is a design record, not implementation proof.

## Job model and scope

- Use one generic Durable Job model for background work, distinguished by `kind`.
- The protocol supports `export` and `ocr`; Phase 5 enables export first and leaves OCR provider-independent.
- Canonical states are `Queued`, `Running`, `Completed`, `Failed`, and `Cancelled`. Cancellation is allowed only while Queued.
- Large exports above 5,000 rows use the async path. Existing synchronous export remains available through 5,000 rows.

## Worker boundary and protocol

- Workers do not receive PostgreSQL credentials. They use authenticated internal API endpoints for claim, heartbeat, chunk reads, completion and failure.
- Worker authentication uses short-lived, environment-specific service tokens with worker identity and narrow scopes.
- Claim at most 10 jobs per batch. A lease lasts 2 minutes and is renewed every 30 seconds. Empty polling backs off from 1 to 15 seconds or uses a 15-second long poll.
- Every lifecycle request binds `job_id`, `attempt_id` and `lease_token`. A duplicate completion for the successful attempt is a no-op; a stale lease cannot complete or replace an artifact.
- The job service claims with `SKIP LOCKED` or its equivalent. Expired leases are reclaimed into Queued and the new attempt receives a new lease token.

## Retry and failure

- Retry only transient errors with jittered exponential backoff, provider retry hints and at most five attempts.
- The worker reports a safe error code; the API owns the retry allowlist and does not trust an arbitrary worker-provided retry decision.
- Timeouts, rate limits and temporary storage failures are retryable. Invalid requests, tenant authorization failures and unsupported formats are terminal.
- Security violations fail immediately and create an audit event. Exhausted retries end in Failed; Phase 5 does not add a separate dead-letter status.

## Export generation and access

- Read authorized export rows through cursor-based internal API chunks of 500 rows and write the output as a stream. Never load the whole dataset or workbook into memory.
- Bound global and per-Organization concurrency. Preserve the existing CSV/XLSX format ceilings and memory budgets from Phase 4.
- Store completed export artifacts in private object storage, never PostgreSQL. A job references the artifact only after upload completes.
- Only the requester and current Owner/Admin of the same Organization may obtain a download URL. Authorization is rechecked for every request.
- Signed URLs expire after 15 minutes. Artifacts expire after 24 hours. Job metadata remains for 90 days and export audit remains for one year.

## OCR seam

- OCR jobs reference an existing authorized Document. Results are structured metadata tied to that Document.
- Original files and any generated files remain in private object storage. PostgreSQL does not store file blobs.
- OCR execution is not enabled until a provider and product behavior are specified separately.

## Observability and audit

- Measure queue wait, runtime, attempts, retries, stale reclaims, failures by safe error code, export rows, export bytes, artifact downloads and worker heartbeat age.
- Alert when queue wait exceeds its operating target, stale reclaim rises, or no healthy worker heartbeat is observed for 2 minutes.
- Audit job creation, claim/reclaim, retry, completion/failure, artifact creation, signed URL creation, download, cleanup and authorization failure.
- Audit records carry Organization, actor or worker identity, job ID and request ID without tokens, signed URLs, file contents or sensitive row values.

## Migration and rollback

- Add the generic `jobs` schema before switching producers or workers.
- Migrate only active Queued/Running export jobs and preserve a compatibility read path for historical `export_jobs`.
- Do not remove the old table or compatibility path until the new worker passes smoke tests and the retention window closes.
- Roll back the worker before destructive schema cleanup. If necessary, stop new async job creation, allow valid leases to finish or expire, and let the previous path continue reading existing export history.
- Migrations require safe up/down behavior before staging deployment.

## Ready gate before implementation is considered complete

- Protocol contract, lifecycle transitions and ownership checks are specified and tested.
- Duplicate claim/completion, stale recovery, bounded retry and cancellation-before-running are tested.
- Tenant-scoped artifact authorization and signed retrieval are tested.
- Large export tests demonstrate bounded row chunks and memory use.
- Queue, worker and export metrics are visible; required alerts and audit events are present.
- Staging migration, smoke test and rollback runbook are proven separately from local tests.
