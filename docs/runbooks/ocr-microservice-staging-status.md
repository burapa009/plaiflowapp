# OCR microservice staging status — 2026-09-28

The OCR pilot is active through `https://plaiflowapp.vercel.app`, whose current
web deployment uses the Railway **staging** API. Railway's existing
`plaiflowapp` service slot runs the private FastAPI service and leased OCR worker
in one Trial container. The service has no public domain. No Railway plan
upgrade was made.
The database OCR scheduler applies to all accepted documents in this staging
environment; the organization allowlist limits the extraction/accounting UI,
not OCR queueing.

| Check | Evidence |
| --- | --- |
| OCR image | Railway staging deployment `3391fdcb-c92d-45e0-95b4-6b96a3a3575f` SUCCESS; `/health` 200 inside the container |
| API | Railway staging deployment `7bc0ce44-a8df-4af3-acb2-b8b4cc959b56` SUCCESS; public `/readyz` 200 |
| Web | Existing Vercel production deployment `dpl_9LdSK483TtUxENpBVVuePrtj71tY` READY; document detail loaded after API activation |
| Configuration | `ocr_settings`: enabled, `paddleocr-3.7.0-ppocrv5-th-v2`, preprocessing `v2`; API `EXTRACTION_ENABLED=true`, `ACCOUNTING_ENABLED=true`, `REVIEW_ENABLED=false`; extraction/accounting routes restricted by `OCR_PILOT_ORGANIZATION_IDS` |
| Real document | Pilot document `ce0ad965-3df1-4677-bf62-98c8d8502d7c`, job `72903d1e-3f33-423f-996f-bdde4985d964`, Completed on first attempt; 1 page in 4.8 s; web displays 44 lines with confidence and coordinates plus an unconfirmed accounting draft |
| Memory | The first image attempt hit the 1 GB Trial cgroup limit. After bounding the model input to 1600 pixels, the new deployment's observed cgroup peak was 863 MiB and `oom_kill=0`. This is one document, not a capacity benchmark. |
| Data | Database backup `.scratch/phase-13/backups/staging-ocr-pre-v2-20260928.dump` SHA-256 `E3A8A9780BC8CF284988FB62688DA91B9C852E539284854BC333F6C3A3B8DDA7` was restored in a disposable database before cutover. No schema migration was needed. |
| Checks | Python OCR unittest suite: 9 passed. Go API `go test ./...`: passed. Anonymous pilot extraction returned 401; nonpilot route returned 404. |

The OCR service records file type, size, page count, duration, and confidence
without OCR text. Extraction logs duration and zero external provider cost;
compute cost is not metered per document. The Trial credit and 1 GB memory cap
limit sustained use. Automatic Railway database backups/PITR remain unavailable
on this plan. Do not call this a full production reliability gate pass.

If the new worker becomes unhealthy, first set `ocr_settings.enabled=false` to
stop scheduling new documents, then stop the OCR service in Railway after
in-flight leases settle. Disable `EXTRACTION_ENABLED` and `ACCOUNTING_ENABLED`
on the API to hide drafts. Keep the v2 database versions and published artifacts;
the API reads both v1 and v2 artifacts. Re-enable only after a fresh smoke test.
