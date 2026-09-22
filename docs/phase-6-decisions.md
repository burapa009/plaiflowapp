# Phase 6 decision record

Status: All interview decisions (Q1–Q17) accepted by the product owner on 2026-09-22. Final shared-understanding confirmation is pending.

This records settled decisions from the Phase 6 PaddleOCR worker design interview. It is a design record, not implementation proof.

## Product scope

- Phase 6 recognizes printed Thai and English business documents, including numbers.
- Handwriting, table reconstruction, seals, formulas and general multilingual recognition have no accuracy guarantee in this phase.
- Use the Thai-specific PP-OCRv5 recognition model because the current PP-OCRv6 language list does not include Thai.

## Accepted input

- Accept JPEG, PNG and PDF.
- Limit each file to 20 MB and each PDF to 20 pages.
- Reject encrypted or password-protected PDFs.
- Determine the PDF page count before rasterizing it.
- Reject unsupported or over-limit input as `invalid_input`; it does not count as a successful OCR run.

## Result boundary

- Produce page and line results with page number, text, confidence, normalized bounding polygon, detected rotation, model/version and processing duration.
- Store the encrypted structured result as a private object-storage artifact.
- PostgreSQL stores only status, input checksum/version, model/version, page count and the artifact reference; it does not store the result blob.
- Business-field extraction is outside Phase 6.

## Initial runtime envelope

- Run OCR in a dedicated CPU worker, separate from the Go API and existing workers.
- Start with 2 vCPU, 4 GB memory and one OCR job at a time, then change capacity only from benchmark evidence.
- Use a separate Linux Python 3.12 worker image rather than changing the existing Python 3.14 export package runtime.

## OCR runtime and models

- Pin `paddleocr==3.7.0` and a verified compatible CPU build of PaddlePaddle in the dependency lock.
- Use `PP-OCRv5_mobile_det` with `th_PP-OCRv5_mobile_rec` for printed Thai, English and numbers.
- Enable text-line orientation detection. Keep document unwarping and other expensive optional modules disabled by default.
- Record model names, model versions and preprocessing version with every result.

## Preprocessing and decoded-input bounds

- Rasterize PDF pages at 200 DPI.
- Normalize EXIF orientation and automatically rotate pages to 0, 90, 180 or 270 degrees.
- Apply only mild deskewing. Do not apply aggressive binarization or denoising by default.
- Reject a decoded page above 25 megapixels and resize the inference copy when its longest edge exceeds 3,500 pixels.
- Preserve the accepted original without modification.

## Timeout and retry

- Limit processing to 45 seconds per page and 15 minutes per Document.
- Permit at most three OCR attempts.
- Retry only transient process, timeout, storage or network failures.
- Treat corrupt files, encrypted PDFs, unsupported types and exceeded input bounds as terminal `invalid_input` failures.
- Continue the Durable Job heartbeat while pages are processed.

## Temporary files

- Use an attempt-specific randomized temporary directory accessible only to the worker process; never construct paths from user filenames or follow symlinks.
- Enforce size and checksum while downloading and limit temporary storage to 512 MB per job.
- Delete temporary files on every exit path and scavenge attempt directories older than one hour at worker startup.
- Do not retain the original on local disk after the attempt ends.

## Worker authority

- Extend the Phase 5 Worker Protocol with a dedicated `ocr-worker` identity that can claim only `ocr` jobs.
- Bind heartbeat, completion, failure, input retrieval and result upload to the current `job_id`, `attempt_id` and `lease_token`.
- Retrieve originals with short-lived signed read access and write results through an attempt-scoped API or signed upload.
- The worker has no PostgreSQL credential, permanent object-storage credential, cross-Organization access or general document-read permission.

## Idempotency and reprocessing

- Fingerprint work by Organization, Document, original checksum, model bundle version and preprocessing version.
- Repeated requests return the existing Queued/Running job or reuse a Completed result for the same fingerprint.
- Changed input or model/preprocessing versions produce a new result.
- After terminal failure, an explicit retry creates a new job linked by `retry_of`; it cannot overwrite a result published by another attempt.
- Enforce deduplication with atomic insertion and database uniqueness. Historical failed jobs must not prevent an authorized retry.

## Atomic publication and result schema

- Publish a result only after every page succeeds. A failed page retries the whole Document within the attempt limit; no per-page checkpoints in Phase 6.
- Exhausted attempts produce Failed without presenting partial text as a successful result. Clean up incomplete or abandoned artifacts.
- Use a versioned schema. Coordinates are normalized to 0..1 against the orientation-corrected page; four polygon points are clockwise.
- Preserve raw model line confidence in 0..1 and do not automatically hide low-confidence text.
- Order lines in reading order and join page text with newlines. Record the corrected page dimensions for overlays.

## Benchmark acceptance gate

- Use at least 60 authorized reference pages: 20 Web uploads, 20 LINE images and 20 scanned PDF pages, covering receipts, tax invoices, office documents and mobile photos.
- Human reviewers establish reference text.
- On 2 vCPU / 4 GB, require median character error rate <=10%, P90 character error rate for clear scans <=15%, and false-empty results on readable pages <=2%.
- Require latency P50 <=8 seconds/page and P95 <=20 seconds/page, peak RSS <=3 GB, and a typical 20-page PDF completed within 5 minutes.
- These are acceptance targets, not measured performance claims. Select the fastest configuration meeting every gate; do not lower gates retrospectively to pass a failed candidate.

## Resource controls and observability

- Claim one job at a time with at most one prefetched job and one executing job per process. Load models once per process.
- Limit each worker to 2 vCPU, 4 GB RAM and 512 MB temporary storage; allow at most two replicas per environment initially.
- During graceful shutdown, stop new claims and continue heartbeats for current work.
- Measure queue wait, seconds/page, pages/job, peak memory/job, retries, timeouts, invalid inputs, false-empty rate and model version.
- False-empty rate requires reference labels or reviewed samples; raw empty-page count alone does not establish an OCR error.
- Log IDs, counts, duration, safe error codes and resource usage; never log OCR text, document content, credentials or signed URLs.
- Alert on failure rate >5% sustained for 15 minutes, processing P95 >20 seconds/page or worker heartbeat absence >2 minutes.

## Result retention

- Keep the current result for the lifetime of its Document. Keep superseded results for 30 days for rollback and review.
- Permanent Document deletion removes all associated OCR results and artifacts.
- Keep job metadata for 90 days and audit records without OCR content for one year, following Phase 5.

## Trigger, entitlement and usage

- Automatically enqueue OCR after a Document passes file validation and malware scanning and its original is stored successfully.
- Apply the same server-side entitlement checks and processing rules to Web, LINE and Drive intake.
- Retries and reuse of existing OCR results do not increment Document Usage.
- Defer separate OCR billing and commercial OCR quotas; enforce the accepted file, page and concurrency limits in Phase 6.
- Only current Owner/Admin members may request manual reprocessing.

## Result authorization and Document lifecycle

- Reading OCR results requires the same current authorization as reading the associated Document.
- Recheck authorization before issuing each signed result URL; URLs expire after five minutes.
- When a Document enters trash, stop publishing new OCR results and stop retries.
- When a Document is permanently deleted, reject subsequent completion and clean up artifacts uploaded by in-flight attempts so that late work cannot restore deleted data.

## Interview completion

- No remaining product questions from this interview. Await the product owner's confirmation of the consolidated scope before ending the grilling session.
- Runtime compatibility, benchmark results, implementation, tests and deployment remain unverified. This decision record does not authorize or establish their completion.
