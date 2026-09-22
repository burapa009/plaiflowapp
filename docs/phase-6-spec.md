# Phase 6 — PaddleOCR Worker

Status: **ready-for-agent (local specification)**. Product decisions Q1–Q17 accepted on 2026-09-22; the user requested specification synthesis only. This document follows the [Phase 6 decision record](./phase-6-decisions.md), [Phase 5 specification](./phase-5-spec.md) and [ADR-0032](./adr/0032-durable-generic-jobs-and-api-worker-boundary.md). Implementation, tickets, migrations and deployments are not part of this delivery.

## Problem Statement

Users can bring business documents into PlaiFlow through Web, LINE and Google Drive, but still need to open each original and read its contents manually. They need printed Thai and English text available alongside the Document, with its page position and recognition confidence, without losing the original or exposing another Organization's data.

OCR is expensive and may fail on malformed input or interrupted execution. Users need truthful processing status, safe retries and one authoritative published result despite duplicate delivery, worker crashes or concurrent requests.

## Solution

Automatically process eligible accepted Documents with a dedicated CPU PaddleOCR worker through the existing Durable Job protocol. Retain the original unchanged and publish a private, versioned OCR Result Artifact only after all pages succeed. Authorized users can view processing status and retrieve the text and positions; current Owner/Admin members can request manual reprocessing.

Apply the same authorization, entitlement, validation and idempotency rules to all intake channels. Enforce resource bounds and prove accuracy and latency using representative Thai documents before production activation.

## User Stories

1. As a Document reader, I want printed Thai text recognized so that I can read it without transcribing the original.
2. As a Document reader, I want English and numbers recognized alongside Thai so that mixed business documents remain useful.
3. As a Web uploader, I want eligible accepted files processed automatically so that I do not start a separate workflow.
4. As a LINE direct sender, I want my accepted Document processed through the same pipeline so that channel choice does not affect correctness.
5. As a LINE group sender, I want OCR bound to the mapped Organization so that group documents remain isolated.
6. As a Drive importer, I want OCR to use the stored original so that recognition does not depend on later Drive availability.
7. As a Document reader, I want Document Source provenance preserved so that I can trace where the original came from.
8. As a PDF uploader, I want every supported page processed in order so that the result represents the whole document.
9. As a mobile sender, I want EXIF orientation and page rotation handled so that rotated photographs can be read.
10. As a Document owner, I want the original preserved so that preprocessing cannot damage my evidence.
11. As a user, I want clear queued, running, completed and failed states so that I understand progress.
12. As a user, I want rejected input explained safely so that I can replace an unsupported or corrupt file.
13. As a user, I want duplicate delivery to reuse existing work so that it does not create competing results.
14. As a user, I want OCR retries to leave Document Usage unchanged so that infrastructure failures do not consume intake quota again.
15. As an Owner/Admin, I want to retry a failed run so that a corrected transient condition can be recovered.
16. As an Owner/Admin, I want a changed model or preprocessing version to create a traceable result so that I can compare versions.
17. As a Document reader, I want text grouped by page and line so that I can relate it to the original.
18. As a Document reader, I want positions and raw confidence retained so that uncertain recognition can be inspected.
19. As a Document reader, I want low-confidence text retained so that potentially useful content is not silently discarded.
20. As a user, I want an incomplete PDF result kept unpublished so that partial processing cannot appear successful.
21. As an Organization member, I want results subject to the Document's current access rules so that OCR cannot bypass permissions.
22. As an Owner/Admin, I want trash and deletion to prevent late publication so that background work cannot resurrect removed content.
23. As an operator, I want retries and stale recovery to use the existing queue so that OCR does not introduce another job system.
24. As an operator, I want enforceable page, memory, disk and time limits so that hostile input cannot exhaust the service.
25. As an operator, I want temporary files cleaned after success, failure and crashes so that originals do not accumulate on workers.
26. As an operator, I want scoped worker credentials and temporary object access so that a worker has limited authority.
27. As an operator, I want latency, queue wait, retries and resource metrics so that processing problems are visible.
28. As an operator, I want content-free logs and audit records so that diagnostics do not leak business documents.
29. As a product owner, I want reproducible Thai accuracy and CPU benchmarks so that production readiness has evidence.
30. As an operator, I want a reversible rollout so that OCR can be disabled while document intake and export continue.

## Implementation Decisions

### Existing architecture and required additions

- Reuse Document, Document Source, Organization Context, Entitlement, Durable Job, Job Lease and Worker Protocol. OCR does not create a second Document or modify source provenance.
- The existing job kind includes `ocr`; claim, heartbeat and failure routes already exist. Result upload/completion currently implement export semantics. Extend the protocol with OCR-specific input access and result publication rather than treating JSON as an export artifact.
- Extend the existing job store, HTTP boundary and Document presentation with only the OCR operations required here. Keep the CPU inference process separate from the Go API, intake worker and export worker.
- Current worker authorization uses general operation scopes. Enforce worker identity, allowed job kind and ownership of the active attempt server-side; a client-supplied `kinds` filter is not authorization.
- Use Linux Python 3.12 for the separate OCR runtime. The existing Python export package uses Python 3.14 and retains its runtime.

### Trigger, quota and user behavior

- Enqueue only after the Document passes type/content checks and malware scanning and the accepted original is stored durably. A missing scanner verdict is not acceptance.
- Use the same server-side entitlement and tenant checks for Web, LINE direct/group and Drive. Workers read the immutable PlaiFlow original, not a user-provided URL or live provider download.
- Persist the accepted-Document-to-job handoff durably using the existing transactional queue boundary; a crash after intake must not silently lose OCR scheduling.
- No new OCR commercial quota or billing in this phase. Initial input limits and bounded execution control resources. OCR attempts, retries and reused results never increment Document Usage.
- Only current Owner/Admin members may request manual reprocessing. Server-side checks apply even when the UI hides the action.
- Expose OCR status, safe failure code and authorized result access through the existing Document experience. The Document remains available when OCR fails. Reading a result follows Document permissions, not export requester-only rules.

### Runtime and input limits

- Planned package pin: `paddleocr==3.7.0`; detection model `PP-OCRv5_mobile_det`, recognition model `th_PP-OCRv5_mobile_rec`. Pin a demonstrably compatible CPU PaddlePaddle build and model bundle checksums before implementation is considered ready. Compatibility is not established by this specification.
- Load models once per inference process. Record exact model bundle and preprocessing versions. Provision verified model assets at build/deployment time rather than accepting arbitrary model locations from jobs.
- Printed Thai, English and numbers are supported. Accept JPEG, PNG and PDF, up to 20 MB and 20 PDF pages. Implementation must use a single documented byte interpretation at API and worker boundaries, consistent with the intake limit.
- Reject corrupt, unsupported, encrypted/password-protected or over-limit files with terminal `invalid_input`. Count PDF pages before rasterization; enforce image dimensions before allocating full decoded images where the decoder permits it.
- Rasterize PDFs at 200 DPI, one page at a time. Bound decoded pages to 25 megapixels and resize inference copies whose longest side exceeds 3,500 pixels. Do not decode an entire PDF into memory.
- Normalize EXIF orientation, handle page rotation at 0/90/180/270 degrees, enable line orientation detection and apply mild deskew. Keep aggressive denoising, binarization and document unwarping disabled by default.
- Keep coordinate transforms sufficient to map recognition coordinates back to the corrected page; resizing and deskew must not produce inaccurate overlays.

### Job lifecycle, retries and resource bounds

- Reuse `Queued`, `Running`, `Completed`, `Failed`, `Cancelled`. Retry returns work to Queued; exhausted attempts end in Failed. Do not introduce independent retry/stale/dead status values. Cancellation remains limited to Queued under the Phase 5 contract.
- Claim one job per request; execute one job per worker process, with no more than one prefetched job. Zero prefetch is the preferred initial implementation so a second lease cannot wait behind a long OCR run.
- Retain two-minute leases and heartbeat every 30 seconds. Heartbeat must continue during inference, not merely between pages. Empty polling backs off using the existing protocol rather than polling continuously.
- Limit each attempt to 45 seconds per page and 15 minutes for the entire Document, including its processing lifecycle. Permit at most three total attempts. API-controlled error allowlists govern retry with jittered backoff.
- Retry transient crashes, timeouts, storage and network failures; reject invalid input, authorization failures and resource-limit violations terminally. Repeated deterministic timeouts cannot loop beyond the attempt cap.
- A timeout must terminate the underlying render/inference operation; returning a timeout while CPU work continues is insufficient. An isolated process may be replaced after termination and reload its models.
- Start at 2 vCPU and 4 GB RAM per worker, at most two replicas per environment, with 512 MB temporary storage per job and per worker. Enforce bounds rather than relying on monitoring alone.
- Stop claiming on shutdown, maintain the active heartbeat while draining within a bounded grace period, and let unfinished work recover through lease expiration if forced to exit. A lost lease prevents publication.

### Idempotency and persistence

- Fingerprint Organization ID, Document ID, immutable original checksum, model bundle version and preprocessing version.
- Concurrent duplicate requests return the active Queued/Running job. A completed matching fingerprint reuses its result. Model or preprocessing changes create a distinct result; an original cannot be changed silently under an existing job.
- Enforce active-work and successful-publication uniqueness atomically in PostgreSQL. A failed historical job must not prevent an explicit authorized retry; a new job records `retry_of` while retaining the same computation fingerprint.
- Persist metadata for Document/Organization linkage, input checksum/version, fingerprint, model/preprocessing/schema version, job/attempt linkage, page count, status, result checksum/byte count, private artifact reference and timestamps. Keep result text, images and JSON blobs out of PostgreSQL.
- Completion verifies the current worker identity, Organization, Document version, attempt and lease. Duplicate completion of the same successful attempt is a no-op; a stale attempt cannot replace or delete the winning artifact.
- Processing is at least once under recovery. The guarantee is one authoritative publication per fingerprint, not exactly-once physical inference.

### OCR Result Artifact and atomic publication

- Store encrypted JSON in private object storage, with a versioned schema containing input/model/preprocessing identity, total duration and ordered pages.
- Each page carries its one-based number, corrected dimensions, applied rotation, duration, joined text and ordered lines. Each line carries text, raw model confidence in 0..1 and four clockwise polygon points normalized to 0..1 in the corrected page coordinate space.
- Preserve low-confidence lines. Join lines in reading order using newlines. Reading order is not a guarantee of table reconstruction or business-field extraction.
- Bound artifact bytes, page/line counts and string sizes at the upload boundary; establish concrete schema limits before implementation and cover them with rejection tests. Never accept unbounded worker JSON.
- Publish only after all pages succeed and upload, schema validation and checksum verification complete. A failed page retries the whole Document; no page checkpoint or user-visible partial success.
- Uploads use unique attempt-scoped locations chosen by the server. Validate the lease before accepting input/upload requests and revalidate at publication. Clean up rejected and abandoned uploads without deleting a published result.
- Plain blank pages may legitimately have empty text. Distinguish them from failed recognition of readable content through benchmark labels or human review.

### Security, temporary access and deletion

- Use a dedicated environment-specific OCR worker identity and short-lived scoped service credentials. It may claim OCR only and act only on its currently leased jobs. It receives neither database credentials nor permanent bucket credentials.
- Reuse the existing credential model only with server-enforced worker restrictions; possession of a shared signing secret must not allow minting a more privileged export or general-document identity.
- Issue short-lived signed input access and attempt-scoped output access through the trusted API. Validate downloaded size, type and checksum against the accepted original. Do not follow arbitrary URL redirects or trust worker-supplied Organization IDs, object keys or Document metadata.
- Use randomized per-attempt directories with process-only permissions, generated filenames and no symlink traversal. Run document parsing with bounded resources; prevent active PDF content or filenames from becoming executable commands.
- Stream downloads, page processing and result writes. Clean up on success, exception and timeout. On startup, scavenge abandoned directories older than one hour only inside the worker-owned temporary root, never another active worker's workspace.
- Authorize result retrieval with current Document access before issuing a five-minute signed URL. Use private/no-store responses and safely render OCR text as untrusted text. A directly issued storage URL remains a bearer capability until expiry; do not claim instantaneous revocation of already issued URLs.
- Trash prevents new publication and retries. Queued work may be cancelled; Running work must be denied publication and stopped or failed terminally without adding a Running-to-Cancelled transition.
- Permanent deletion removes all results and artifacts and rejects late completion. Coordinate deletion with final publication and orphan cleanup so a late upload cannot restore deleted data.
- Current results live with the Document; superseded results remain for 30 days, job metadata for 90 days and content-free audit for one year. OCR retention is distinct from the shorter export-artifact retention.

### Observability and safe rollout

- Measure queue wait, seconds/page, pages/job, peak memory/job, attempts, retries, timeouts, invalid inputs, model version and worker heartbeat age. False-empty rate requires reviewed labels; do not equate all empty pages with failures.
- Alert when failure rate exceeds 5% for 15 minutes, processing P95 exceeds 20 seconds/page, or worker heartbeat is absent for two minutes. Declare denominators and separate invalid-input failures from runtime failures in dashboards.
- Audit scheduling, manual retries, input access, publication/rejection, signed retrieval and cleanup using Organization/job/attempt IDs and safe codes. Exclude OCR text, original content, tokens, signed URLs and credentials from logs and metrics. Keep high-cardinality IDs out of metric labels.
- Add schema and compatible API operations before enabling producers or workers. Keep existing export operations functional. Require safe migration and rollback proof in staging when implementation is requested.
- Rollback disables new OCR scheduling and drains or fences workers before reverting incompatible API behavior. Preserve Documents and completed results; do not depend on destructive schema rollback to stop OCR.

## Testing Decisions

- Primary seam: the existing authenticated HTTP Worker Protocol plus Document authorization boundary. Drive a Document through accepted intake, claim, input access, result upload/publication and authorized retrieval; assert user-observable state, output and access denial. Prefer this over tests of helper names or internal call order.
- Prior art: current worker claim/authentication HTTP tests, worker token and artifact-token tests, document service/intake/encrypted-storage/scanner tests and LINE Document worker tests. Extend these patterns rather than inventing a second queue test harness.
- Add real PostgreSQL integration coverage where a fake cannot prove atomicity: concurrent fingerprint insertion, competing claims, same-attempt duplicate completion, stale completion, explicit retry after failure and deletion racing with publication.
- Cover each intake channel reaching the same OCR pipeline, preserved Drive provenance, absent entitlement, scan failure, wrong tenant, wrong worker kind, expired credentials, forged object references and unauthorized manual retry/retrieval.
- Exercise timeout during inference while heartbeats continue, killed worker recovery, three-attempt exhaustion, whole-document retry after one failed page, shutdown and terminal invalid input.
- Use tiny checked-in or generated authorized image/PDF fixtures for deterministic protocol tests; stub only inference at its external execution boundary. Include bounded-input, decompression, oversized output, unsafe path and cleanup cases.
- Add a real pinned-model CPU smoke run proving Thai/English text, PDF page order, rotated pages, normalized coordinates and result schema. Protocol tests with fake OCR do not prove model quality or runtime compatibility.
- Test trash/deletion before claim, during upload and before completion; verify no late artifact is published and stale cleanup cannot remove a successful artifact.
- Confirm tests leave export behavior, Document Usage and original bytes unchanged. Separate status/authorization tests from actual object-storage and deployment evidence.

### Benchmark and release gate

- Use at least 60 authorized pages: 20 Web, 20 LINE and 20 scanned PDF pages, with receipts, tax invoices, office documents and mobile photos. Include Thai/English mixed text and representative rotations. Store human-checked reference text securely.
- Freeze the corpus and normalization rules before comparing candidates. Report per-page character error rate (CER), corpus aggregate and source/document-type breakdowns. Keep a clear-scan subset defined before scoring. Count Thai combining marks consistently; do not strip characters to improve scores.
- On 2 vCPU / 4 GB require median page CER <=10%, clear-scan P90 CER <=15%, and readable pages incorrectly returned empty <=2%.
- Require warm processing latency P50 <=8 seconds/page, P95 <=20 seconds/page, peak total worker RSS <=3 GB including inference subprocesses, and a representative 20-page PDF completed within five minutes. Report cold model startup and end-to-end queue/download/upload time separately.
- Record model/package versions, hardware, thread/concurrency settings, corpus revision, sample counts and failures. Never omit failed pages from the report; readable pages returning no text count as recognition failures.
- Choose the fastest configuration passing every gate. These are targets, not claims of achieved PaddleOCR performance. Failed gates block production activation and require an explicit product decision before thresholds change.
- Implementation completion additionally requires relevant tests/lint/typecheck/build, safe staging migration, real CPU worker smoke test, visible metrics and alerts, secure retrieval and a rehearsed rollback path. Local checks alone do not establish staging readiness.

## Out of Scope

- OCR implementation during this specification task, ticket decomposition, deployment or external resource provisioning.
- Business-field extraction, accounting classification, table reconstruction, layout understanding, translation, summarization or LLM processing.
- Guaranteed handwriting, seal or formula recognition; DOCX, XLSX, TIFF and general multilingual OCR.
- GPU runtimes, managed OCR fallback, aggressive image enhancement or model training.
- Search indexing, bulk historical backfill, bulk reprocessing, manual text correction and new OCR-specific notifications/tasks.
- Separate commercial OCR billing/quotas, page checkpoints and partial-result publication.
- New job queues, worker database access, or changes to LINE/Google authentication flows.

## Further Notes

- The user requested synthesis after accepting all interview recommendations; existing protocol seams are reused as specified rather than reopening the interview.
- The package/model names are planned pins from the accepted decision record. Exact PaddlePaddle compatibility, orientation/deskew implementation and bounded result-schema sizes are implementation validation items; none are established as tested here.
- Benchmark documents and ground truth are required inputs before the quality gate can run. Do not fabricate an accuracy result or use unauthorized customer documents.
- `ready-for-agent` means this local specification is available for later ticket decomposition. The project has no configured issue-tracker/triage documents, so external publication is pending `/setup-matt-pocock-skills`; this file is not proof of a published tracker issue.
