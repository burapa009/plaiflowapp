"""Bridge the existing durable OCR job worker to the private HTTP service."""

import json
import os
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid
from pathlib import Path

from .protocol import NoRedirect
from .worker import MAX_FILE, MAX_RESULT


class ServiceClient:
    def __init__(self) -> None:
        self.url = os.environ["OCR_SERVICE_URL"].rstrip("/")
        self.token = os.environ["OCR_SERVICE_TOKEN"]
        parsed = urllib.parse.urlsplit(self.url)
        private_http = parsed.scheme == "http" and (parsed.hostname or "").endswith(".railway.internal") and parsed.port is not None
        if (parsed.scheme != "https" and not private_http) or not parsed.hostname or parsed.username or parsed.password or parsed.query or parsed.fragment or len(self.token) < 32:
            raise ValueError("invalid_service_configuration")
        self.opener = urllib.request.build_opener(NoRedirect())
        self.peak_rss = 0

    def start(self) -> None:
        return None

    def close(self) -> None:
        return None

    def __call__(self, path: Path, deadline: float) -> list[dict]:
        if time.monotonic() >= deadline or path.stat().st_size > MAX_FILE:
            raise ValueError("invalid_input")
        with path.open("rb") as source:
            signature = source.read(12)
            source.seek(0)
            # ponytail: bounded 20 MiB body in memory; stream multipart if intake limit grows.
            content = source.read(MAX_FILE + 1)
        if signature.startswith(b"%PDF-"):
            mime = "application/pdf"
        elif signature.startswith(b"\x89PNG\r\n\x1a\n"):
            mime = "image/png"
        elif signature.startswith(b"\xff\xd8\xff"):
            mime = "image/jpeg"
        elif signature.startswith(b"RIFF") and signature[8:12] == b"WEBP":
            mime = "image/webp"
        else:
            raise ValueError("invalid_input")
        boundary = uuid.uuid4().hex
        body = (
            f"--{boundary}\r\nContent-Disposition: form-data; name=\"file\"; filename=\"original\"\r\n"
            f"Content-Type: {mime}\r\n\r\n"
        ).encode() + content + f"\r\n--{boundary}--\r\n".encode()
        request = urllib.request.Request(
            self.url + "/v1/ocr", data=body, method="POST",
            headers={"Authorization": "Bearer " + self.token,
                     "Content-Type": "multipart/form-data; boundary=" + boundary},
        )
        try:
            with self.opener.open(request, timeout=max(1, deadline - time.monotonic())) as response:
                body = response.read(MAX_RESULT + 1)
                if len(body) > MAX_RESULT:
                    raise RuntimeError("oversized_ocr_response")
                result = json.loads(body)
        except urllib.error.HTTPError as error:
            if error.code == 400:
                raise ValueError("invalid_input") from error
            raise RuntimeError("temporary_upstream") from error
        except ValueError as error:
            raise RuntimeError("temporary_upstream") from error
        try:
            if not result["success"] or not 1 <= len(result["pages"]) <= 20:
                raise RuntimeError("temporary_upstream")
            return [{
                "page_number": page["page"], "width": page["width"], "height": page["height"],
                "rotation": page["rotation"], "duration_ms": page["duration_ms"],
                "lines": [{
                    "text": line["text"], "confidence": line["confidence"],
                    "polygon": line["polygon"],
                } for line in page["lines"]],
            } for page in result["pages"]]
        except (KeyError, TypeError, ValueError) as error:
            raise RuntimeError("temporary_upstream") from error
