# RunPod OCR migration — staging pilot configured

## Current and target flow

PlaiFlow currently accepts and encrypts documents in private object storage, creates a PostgreSQL `durable_jobs` OCR job, and lets a Railway worker claim a two-minute lease. The worker publishes an OCR v2 artifact through the Go API. Extraction and accounting suggestions remain in Go; human confirmation is separate. The public Vercel site currently uses the Railway **staging** API. See [OCR staging status](ocr-microservice-staging-status.md) for the last deployed evidence.

The RunPod candidate retains that contract. A lightweight Railway dispatcher claims only selected organizations, submits an async RunPod `/run` job, polls `/status`, and keeps the PlaiFlow lease alive. RunPod fetches the source through the Go API using two distinct, short-lived `ocr:input` tokens and the current lease. The API decrypts the stored object. RunPod returns raw page lines with text, polygon, confidence, dimensions, and duration; the dispatcher normalizes them into the existing v2 artifact. Go validates the hash, geometry, model version, active lease, and document state before publication. The RunPod worker never receives storage credentials, the OCR signing key, database access, or permission to publish results.

The new `GET /v1/o/{organization}/documents/{document}/matches` endpoint is default-off and pilot-scoped. It compares a source OCR draft or confirmed review against **confirmed** documents in the same organization. PostgreSQL first narrows by amount (±0.01) and date (±3 days), then computes `pg_trgm` vendor similarity for at most 50 candidates. A high score is still a candidate; it never confirms a review or writes an accounting suggestion. Existing vendor-to-category rules remain separate.

## Staging evidence — 2026-09-29

Commits `a71a88e` and `8157e34` are on `origin/feat/business-management`; CI runs `36459755708` and `36475386992` passed. The pre-v19 off-host backup `.tools/backups/plaiflow-runpod-pre-v19-20260929.dump` has SHA-256 `25daebdeee5c11d7860a3263d63d9aa55e97a1a5d5f88aa9a28ab01a9d7fd770`. A disposable restored database and staging both report migration `19`, `dirty=false`; the restored copy has `pg_trgm` and `document_match_index`. Local Go tests and 13 OCR Python tests passed.

Railway staging API deployment `f63f2e7d-d2c3-4043-b092-cfffcda12bde` (commit `8157e34`) and OCR service deployment `d24cc325-2738-438a-89f1-60a63f19fc2c` report `SUCCESS`. The API `/readyz` and private OCR `/health` returned 200. `OCR_PROVIDER=railway` remains the default. Only Organization `44f1ca09-d7d9-421c-80fd-4783098e62f2` (PlaiFlow RunPod OCR Synthetic Pilot) is routed to RunPod; matching remains disabled. The OCR service runs its CPU worker and a separate RunPod dispatcher. Its restricted endpoint key is in Railway staging variables, not source control.

RunPod endpoint `0oemotfk966zij` uses the immutable image tagged `98f8fd49c77f9d6617b07a91b1664c6d545f3fea`, Active Workers `0`, Max Workers `1`, a five-second idle timeout, and a 600-second execution timeout. After L4-only placement was throttled, RTX 3090 and RTX A5000 were also allowed. The first sample job was interrupted when routing was paused to confirm uploaded samples; its eventual completion has no provider timing metadata and is not GPU proof.

The isolated pilot's `pilot-first-page.png` completed on its first attempt through RunPod (provider job `b5b0a8b9-1cb3-409d-a7e2-8ce026d6c392-u2`, one page). Persisted processing time was 71.364 seconds, including 65.727 seconds of provider queue/startup and 2.820 seconds of GPU execution. The document page shows Thai OCR text, six positioned/confidence-bearing lines, and a reviewable accounting draft. Initially the draft kept the total amount empty with a warning because the sample says `รวม 1,070.00`; after staging deployment `f63f2e7d`, opening the same artifact derived a total of 1,070.00 with line-6 evidence. No OCR rerun was needed. RunPod reported zero running workers, zero queued jobs, and a $0/s spend rate after the request, but its Workers tab still showed one idle worker record; full worker-count scale-to-zero remains to be checked. The console briefly showed Max Workers `3`; it was reset and verified at `1` before further load. Balance was $14.99 (down from $15); billed usage chart still showed $0 because of reporting lag. The 500-page/concurrent benchmark, measured cost per page, and rollback rehearsal remain open. Production rollout and billing stay unchanged.

The five synthetic PDFs completed 100/100 pages through RunPod, each on attempt one. The app displayed pages 1–20 of the Thai receipt and low-contrast receipt PDFs, including OCR text and positional/confidence metadata. The figures below are **per document**, with 20 pages in each document; they are not per-page latency percentiles.

| Synthetic PDF | Processing s | GPU execution s | Provider queue/startup s |
| --- | ---: | ---: | ---: |
| Thai receipt | 28.348 | 13.347 | 13.976 |
| High-resolution Thai tax invoice | 21.969 | 9.226 | 9.918 |
| English invoice | 21.939 | 5.569 | 14.066 |
| Rotated receipt | 34.524 | 15.301 | 17.799 |
| Low-contrast receipt | 28.247 | 10.586 | 16.265 |
| **Total / 100 pages** | **135.027** | **54.029** | **72.024** |

Aggregate GPU execution was 0.540 seconds per page; processing was 1.350 seconds per page when each document's queue time is included. Across only five document jobs, processing p50 was 28.247 seconds and p95/p99 were both 34.524 seconds (nearest-rank), so these percentiles are not a concurrency or per-page SLA. RunPod balance moved from $14.99 before the five PDFs to $14.96 afterward; this rounded, delayed balance change is an indicative approximately $0.03, not a reconciled cost/page. The billing explorer still reported $0.00. No retry or failed RunPod job occurred in these five documents. OCR text is visibly readable on the synthetic pages, but repeated values do not establish field accuracy on customer documents. The bare-`รวม` rule fix is covered by Go tests and verified in the staging UI against the stored OCR artifact.

The user stopped further testing after this 100-page pass. The generated `.scratch/runpod-corpus-500` files were not uploaded; no 500-page, concurrent, or Railway control comparison was run. The pilot remains scoped to its sample Organization, with RunPod Active Workers `0` and Max Workers `1`. Keep staging default on Railway and matching disabled until the remaining gates are deliberately resumed and verified.
## Required order

1. Record current deployment IDs, OCR queue counts, database version and dirty state. Take an off-host `pg_dump -Fc`, restore it into a disposable PostgreSQL database, and apply migration 19 there. Run the tenant and rollback checks below. Do not alter the current staging worker yet.
2. Build from the repository root: `docker build --platform linux/amd64 -f ocr/Dockerfile.runpod -t <private-image>:<immutable-tag> .`. Build pre-downloads and SHA-256 verifies the **existing** `PP-OCRv5_mobile_det`, `th_PP-OCRv5_mobile_rec`, and text-line orientation models. GitHub Actions [built and pushed the immutable GPU image](https://github.com/burapa009/plaiflowapp/actions/runs/36458674778) tagged `ghcr.io/burapa009/plaiflowapp/plaiflow-ocr-runpod:98f8fd49c77f9d6617b07a91b1664c6d545f3fea` and verified wheel metadata and bundled models on a CPU runner. GPU inference remains unverified until a RunPod worker executes a job.
3. Push the immutable image to a private registry. Create a **queue-based RunPod Serverless Flex** endpoint with Active Workers `0`, Max Workers `1` initially, one suitable GPU, idle timeout about 5 seconds, execution timeout 600 seconds, and job TTL at least 840 seconds. Do not create an always-on Pod. Set `OCR_INPUT_API_URL` on the RunPod worker to the exact public HTTPS Go API origin. Restrict endpoint key access to the dispatcher.
4. Apply migration 19 to staging using `scripts/migrate-railway.ps1 -Action up-one` only after the restored backup and migration test pass. Confirm version `19`, `dirty=false`, `pg_trgm` available, and the indexed candidate query with `EXPLAIN (ANALYZE, BUFFERS)` on a disposable copy. Deploy the Go API with `OCR_PROVIDER=railway`, `MATCHING_ENABLED=false`, and `OCR_RUNPOD_ENABLED_ORGS=<pilot UUID>`. This preserves the old OCR path for other organizations.
5. On the current OCR Railway service, set `OCR_PROVIDER=railway`, `OCR_RUNPOD_DISPATCHER=true`, `RUNPOD_API_KEY`, and `RUNPOD_ENDPOINT_ID`. Its supervisor keeps the CPU worker for nonpilot organizations and starts a lightweight RunPod dispatcher for the pilot. Do not put the API key in Vercel, `NEXT_PUBLIC_*`, logs, or the RunPod worker.
6. Upload a consented pilot image/PDF and verify queue → RunPod job → v2 artifact → Thai extraction → human review. Check duplicate result, timeout/retry, trash-before-result, 20-page boundary, other-organization denial, and job visibility. Enable `MATCHING_ENABLED=true` only after migration/index checks. Existing confirmed reviews are **not backfilled** into the new match index; re-confirm or perform a validated backfill before interpreting missing matches.
7. After representative 100-page and 500-page tests, measured cost, and rollback rehearsal, set API `OCR_PROVIDER=runpod` and switch the OCR Railway service to `OCR_PROVIDER=runpod`. Its supervisor then starts only the dispatcher, so Railway no longer loads PaddleOCR. Verify container RSS and that an idle RunPod endpoint scales to zero. This is a **staging** switch only. Production billing and production rollout remain off.

## Configuration

| Place | Variable | Meaning |
| --- | --- | --- |
| Go API | `OCR_PROVIDER=railway|runpod` | Default provider for new claims; unset means Railway |
| Go API | `OCR_RUNPOD_ENABLED_ORGS` | Comma-separated pilot organization UUIDs routed to RunPod before the default switch |
| Go API | `MATCHING_ENABLED` | Registers matching suggestions route; default false |
| Go API | `OCR_AUTO_MATCH_THRESHOLD`, `OCR_REVIEW_THRESHOLD` | Candidate labels only; defaults 0.90 and 0.70; must satisfy `0 < review < auto <= 1` |
| Railway OCR service | `OCR_PROVIDER=railway|runpod` | Local CPU worker or lightweight RunPod dispatcher |
| Railway OCR service | `OCR_RUNPOD_DISPATCHER=true` | While local CPU mode is active, start a second dispatcher for pilot jobs |
| Railway OCR service | `RUNPOD_ENDPOINT_ID`, `RUNPOD_API_KEY` | Server-side RunPod endpoint and secret |
| Railway OCR service | `RUNPOD_GPU_CLASS` | Optional label for cost analysis, not verified by the provider response |
| RunPod worker | `OCR_INPUT_API_URL` | Exact public HTTPS Go API origin allowed for source reads |

Existing `JOB_API_URL`, `OCR_WORKER_AUTH_KEY`, `OCR_WORKER_ID`, `APP_ENV`, OCR artifact storage, extraction and accounting flags stay in use. Never point a staging worker at a production API. The RunPod input contains scoped bearer tokens and a lease token: protect provider job payloads and access logs as confidential document access material. The tokens expire in 15 minutes, are single-use per source request, and remain bound to an active PlaiFlow lease.

## Failure, retry, and rollback

The source file stays in PlaiFlow storage if RunPod fails. `durable_jobs` already caps attempts at three and applies exponential backoff to `temporary_upstream`. Only `invalid_input` is terminal. A provider job can finish after its PlaiFlow lease expires; that stale result cannot publish. A second provider invocation may still incur cost. The existing Owner/Admin OCR retry action can create a new attempt after failure; do not repeatedly retry while Go API lease endpoints return 503.

For pilot rollback, remove the pilot UUID from `OCR_RUNPOD_ENABLED_ORGS`, turn off `OCR_RUNPOD_DISPATCHER`, and let current leases settle. For a staging-default rollback, set the API and OCR service `OCR_PROVIDER=railway` and redeploy the previous CPU-capable image. Keep migration 19 in place after data exists; it is additive and does not alter old OCR artifacts. If matching behaves incorrectly, set `MATCHING_ENABLED=false`. Do not run the down migration against live pilot data.

The `document_ocr_runs.provider`, `provider_job_id`, `provider_submitted_at`, `processing_ms`, `provider_execution_ms`, `provider_queue_ms`, `gpu_class`, and nullable `provider_cost_microusd` columns support inspection. **Null cost is not zero cost.** RunPod bills worker startup, execution, and idle time; use its billing export with completed page counts to calculate actual cost per page. Do not infer billable cost solely from `executionTime`.

## Verification and monitoring

Run `go test ./...`, `go vet ./...`, `go build ./cmd/server` from `api/`; run `python -m unittest discover -s ocr/tests -v` with `PYTHONPATH=ocr`. Run the real PostgreSQL suite with a disposable `OCR_TEST_ADMIN_URL`; a skipped test is no RLS/migration proof. Check migration 19 up/down on a **disposable** database, organization A/B read denial, duplicate submit, provider timeout, malformed result, cold start, no OCR text or tokens in logs, and a restored backup. Monitor queued/running/failed OCR jobs, `provider_job_id`, provider queue/execution times, API 5xx, OCR service RSS, RunPod worker count, and billed spend. Use correlation/job/document/organization IDs without raw document text.

For capacity tests, measure p50/p95/p99 queue wait, GPU processing, end-to-end latency, errors, retries, pages, and billed cost for 100 then 500 representative pages and concurrent uploads. Compare with the existing Railway CPU path on the **same** annotated corpus. A synthetic 100-page sequential RunPod pass is proved; the 500-page/concurrent benchmark, Railway control comparison, reconciled cost/page, managed database backup/PITR, and full production gates are not. It must not be labelled production-ready yet.

The staging pilot is authorized for **synthetic/sample documents only**. Do not send customer or production documents to RunPod. Generate the initial 100 pages with `& .\.tools\ocr-venv\Scripts\python.exe scripts\generate-runpod-corpus.py --pages 100 --output .scratch\runpod-corpus` from the repository root. This creates five 20-page PDFs: Thai receipt, Thai tax invoice at high resolution, English invoice, rotated receipt, and blurred low-confidence receipt. The generated reference text is synthetic; OCR accuracy on these pages does not establish accuracy on real invoices. Upload the same PDFs to a RunPod pilot organization and a Railway control organization through the normal staging document flow; confirm each job and page count. Do not run the 500-page variant (`--pages 500`) until the 100-page run is healthy and its spend is reviewed.

The RunPod staging pilot has a **$10 total cap**. Check the RunPod balance and billed usage after each 20-page PDF and stop dispatching at **$8**, leaving $2 for delayed charges. Billing explorer may lag by one hour; also estimate spend from elapsed billable worker time at the selected GPU rate and stop at the earlier $8 signal. Never use the explorer lag as permission to exceed the cap. Keep Active Workers at 0 and Max Workers at 1; verify scale-to-zero after the final request. Capture billed USD, cold-start delay, p50/p95 OCR and end-to-end seconds/page, error rate, GPU seconds/page, RSS/VRAM peak, and cost per 1,000 pages. `provider_queue_ms` is queue plus startup delay, not a standalone model-load metric. Query completed `document_ocr_runs` with `page_count`, `processing_ms`, `provider_execution_ms`, `provider_queue_ms`, and `gpu_class`; divide billed dollars by completed pages and multiply by 1,000. Record retries and failed billable jobs separately. The current response does not persist VRAM; collect it from RunPod worker metrics or a GPU runtime probe before reporting that field as measured. Compare normalized OCR text, bounding boxes, confidence, and field warnings against Railway on the same synthetic documents.

## Security review (STRIDE)

Trust boundaries: browser→Go API (session/role), Go API→encrypted object store (server credentials), Railway dispatcher→RunPod API (RunPod key and scoped source capability), RunPod→Go API (lease-bound source fetch), Go API→PostgreSQL (tenant-scoped transactions/RLS). Assets include source financial documents, OCR text, source tokens, OCR keys, reviews, and accounting suggestions. Likely actors are a compromised tenant account, leaked worker/provider token, insider with provider job access, or malicious uploaded file.

| Threat | STRIDE / ATT&CK | Risk | Mitigation / owner |
| --- | --- | --- | --- |
| Forged provider output or replay overwrites another organization's result | S/T/E, T1565 | High | API validates lease, worker, SHA-256, model, document state and artifact schema; backend owner verifies cross-org test before rollout |
| RunPod input URL redirects to an attacker or internal address | I/E, T1530 | High | Exact `OCR_INPUT_API_URL`, fixed job path, disabled redirects, API-generated source URL; backend owner tests SSRF cases |
| Scoped source tokens leak through job payload/logs | I, T1552 | High | Single-use 15-minute tokens, no plaintext logs, least-privilege provider key, provider retention review; platform owner verifies logs/config |
| Cold start, 20-page PDF, or repeated provider retries exhaust quota | D, T1499.003 | Medium | Max workers 1, 3 PlaiFlow attempts, 14-minute job TTL, pilot scope, spend alert; platform owner measures load/cost |
| Similar OCR figures get treated as confirmed accounting data | T/R, T1565 | High | Matching only returns suggestions, conflicts and uncertain fields require review, audit retains confirmed values; product/backend owner checks review flow |

Residual gaps: no selected/approved external LLM provider, no LLM document transmission, no statement/payment-reference extraction, and no automatic booking. These are separate changes; the current matching endpoint covers confirmed invoice/receipt/duplicate candidates only.
