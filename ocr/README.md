# PlaiFlow OCR service

The existing durable OCR job worker can call this private FastAPI service with
`OCR_SERVICE_URL` and `OCR_SERVICE_TOKEN`. The service loads one pinned CPU
PaddleOCR model at startup, recognizes each image or PDF page, and returns
structured OCR. It does not extract tax IDs, totals, or other accounting fields.
The existing Go extraction layer reads the separately authorized OCR artifact.

## Local run

From the repository root, install `ocr/requirements.lock` into Python 3.12,
set `PADDLE_PDX_CACHE_HOME` to the pinned model cache, set a random
`OCR_SERVICE_TOKEN` of at least 32 characters, and run
`python -m plaiflow_ocr.api` with `PYTHONPATH=ocr`. `GET /health` reports
readiness after the model loads. For Docker, run
`docker compose -f ocr/compose.yml up --build` with `OCR_SERVICE_TOKEN` in the shell environment. The compose
health check calls `/health` and binds the service only to localhost port 8001.

`POST /v1/ocr` accepts one `file` part containing JPEG, PNG, WebP or PDF and a
`Bearer` service token. The response has page number, page text, confidence, pixel polygon,
pixel and normalized bounding boxes, center, size and reading order. Pixel
coordinates use the EXIF-corrected upright image or the rendered PDF page at
200 DPI. A large image may be downscaled for inference; coordinates are mapped
back before responding. `build_ai_payload(page)` and `build_ai_text(page)` in
`normalize.py` provide compact inputs for a separate extraction layer.

The HTTP service defaults to 20 MiB and 50 PDF pages. The existing PlaiFlow
document intake and durable job artifact remain capped at 20 MiB and 20 pages.
Set `OCR_MAX_PDF_PAGES=20` on the service when connecting the current PlaiFlow
worker; the 50-page default applies only to standalone HTTP use.
Responses over 8 MiB fail rather than returning incomplete OCR data.
Invalid files return 400, excessive size 413, and inference failure 500 with a
stable error code and request ID. The service does not store a database record.
`OCR_INCLUDE_RAW_RESULT` defaults to false and should remain false for normal
operation because debug output contains document text.

## Rollout

Deploy `ocr/railway.service.toml` as a separate **private** service using the
same Docker image as the worker. Set the token in both services and configure
`OCR_SERVICE_URL=http://<service>.railway.internal:<port>` only on the worker.
Railway's [private network](https://docs.railway.com/networking/private-networking/how-it-works)
encrypts service traffic with WireGuard; the client
also accepts HTTPS for other deployments. Do not expose the OCR service to browsers.
The worker calls `/v1/ocr` when this URL is present; otherwise it retains its
existing local subprocess path. It continues to claim and publish results
through the existing leased job protocol. Use one Uvicorn worker: every process
loads another model. The observed synthetic Windows smoke used roughly 680 MiB
peak RSS for one process tree; size CPU/RAM from representative documents and
container measurements before increasing replicas. Inference is serialized per
process, while the FastAPI event loop stays available for health checks.

Structured artifacts use schema version 2, model version
`paddleocr-3.7.0-ppocrv5-th-v2`, and preprocessing version `v2`. The artifact
retains its existing normalized `polygon` for the accounting parser and adds
`pixel_polygon`, boxes, dimensions, confidence flags and reading order. Go
continues to read version 1 artifacts. Before routing jobs to the new worker,
drain version 1 jobs, verify a restorable database backup, and set the
disabled `ocr_settings` row to the two v2 version strings. Only then enable
OCR under the existing release gate. No database update or deployment is
performed by this change.

```sql
UPDATE ocr_settings
SET model_version = 'paddleocr-3.7.0-ppocrv5-th-v2',
    preprocessing_version = 'v2'
WHERE singleton AND NOT enabled
  AND model_version = 'paddleocr-3.7.0-ppocrv5-th-v1'
  AND preprocessing_version = 'v1';
```

Require exactly one updated row before activating OCR; do not run this on an
unbacked database. Deploy the Go schema reader and new worker together after
the old queue drains, so an old worker cannot publish a v1 artifact labeled v2.

Keep ingress request-body limits aligned with `OCR_MAX_FILE_SIZE_MB` because
the multipart parser spools a request before the route checks file size. Use
TLS outside Railway's encrypted private network, restrict callers to the backend worker, rotate the shared token through
secret management, and leave debug output disabled. Uploaded files live in a
private temporary directory and are deleted after each request; logs contain
IDs, type, size, timing and aggregate confidence, never OCR text. The service
has no database or object-store credentials. A future queue can replace the
worker-to-service call without changing the OCR response.
