"""RunPod Serverless OCR handler using PlaiFlow's pinned Thai PP-OCRv5 model."""

import hashlib
import json
import os
import re
import tempfile
import time
import urllib.parse
import urllib.request
from pathlib import Path

from plaiflow_ocr.engine import Engine
from plaiflow_ocr.protocol import NoRedirect
from plaiflow_ocr.worker import MAX_FILE, MODEL_VERSION, PREPROCESSING_VERSION


def build_handler(engine):
    trusted_api = os.environ["OCR_INPUT_API_URL"].rstrip("/")
    parsed_api = urllib.parse.urlsplit(trusted_api)
    if parsed_api.scheme != "https" or not parsed_api.hostname or parsed_api.path or parsed_api.query:
        raise ValueError("invalid_input_api_configuration")
    opener = urllib.request.build_opener(NoRedirect())

    def handler(job):
        data = job.get("input")
        if not isinstance(data, dict) or data.get("api_url") != trusted_api:
            raise ValueError("invalid_input")
        job_id = data.get("job_id", "")
        if not isinstance(job_id, str) or not re.fullmatch(r"[0-9a-f-]{36}", job_id):
            raise ValueError("invalid_input")
        if data.get("model_version") != MODEL_VERSION or data.get("preprocessing_version") != PREPROCESSING_VERSION:
            raise ValueError("incompatible_ocr_version")
        attempt = data.get("attempt_id", "")
        lease = data.get("lease_token", "")
        if not isinstance(attempt, str) or not re.fullmatch(r"[0-9a-f-]{36}", attempt) or not isinstance(lease, str) or len(lease) > 256:
            raise ValueError("invalid_input")
        prefix = "/internal/v1/ocr/jobs/" + job_id
        headers = {"X-Job-Attempt": attempt, "X-Job-Lease": lease}

        def fetch(path, token, limit):
            if not isinstance(token, str) or len(token) > 2048:
                raise ValueError("invalid_input")
            request = urllib.request.Request(
                trusted_api + path,
                headers={**headers, "Authorization": "Bearer " + token},
            )
            try:
                with opener.open(request, timeout=30) as response:
                    body = response.read(limit + 1)
            except Exception:
                # No URL, credential, OCR text, or provider exception enters logs.
                raise RuntimeError("input_unavailable") from None
            if len(body) > limit:
                raise ValueError("invalid_input")
            return body

        started = time.monotonic()
        metadata = json.loads(fetch(prefix + "/input", data.get("input_auth"), 16384))
        path = metadata.get("download_url", "")
        url = urllib.parse.urlsplit(path)
        if url.scheme or url.netloc or url.path != prefix + "/original" or not url.query or metadata.get("mime") not in {"image/png", "image/jpeg", "image/webp", "application/pdf"} or not 0 < metadata.get("size", 0) <= MAX_FILE:
            raise ValueError("invalid_input")
        content = fetch(path, data.get("original_auth"), MAX_FILE)
        digest = hashlib.sha256(content).hexdigest()
        if len(content) != metadata["size"] or digest != metadata["sha256"] or digest != data.get("sha256"):
            raise ValueError("invalid_input")
        with tempfile.TemporaryDirectory(prefix="runpod-ocr-") as directory:
            original = Path(directory) / "original"
            original.write_bytes(content)
            pages = engine(original, started + 600)
        output = {
            "schema_version": "runpod.raw.v1",
            "job_id": job_id,
            "document_id": data.get("document_id"),
            "organization_id": data.get("organization_id"),
            "input_sha256": digest,
            "model_version": MODEL_VERSION,
            "preprocessing_version": PREPROCESSING_VERSION,
            "pages": pages,
            "ocr_ms": round((time.monotonic() - started) * 1000),
        }
        if len(json.dumps(output, ensure_ascii=False).encode()) > 8 << 20:
            raise ValueError("invalid_result")
        return output

    return handler


if __name__ == "__main__":
    import runpod

    model = Engine(max_pages=20)
    model.start()  # One model per worker, reused across jobs.
    runpod.serverless.start({"handler": build_handler(model)})
