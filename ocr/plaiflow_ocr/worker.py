import hashlib
import json
import os
import tempfile
import threading
import time
from pathlib import Path
from typing import Any, Callable

from .normalize import normalize_pages

MAX_FILE = 20 << 20
MAX_RESULT = 8 << 20
MAX_TEMP = 512 << 20
MODEL_VERSION = "paddleocr-3.7.0-ppocrv5-th-v2"
PREPROCESSING_VERSION = "v2"


def normalize(pages: list[dict[str, Any]]) -> list[dict[str, Any]]:
    if not 1 <= len(pages) <= 20:
        raise ValueError("invalid_input")
    canonical = normalize_pages(pages, float(os.environ.get("OCR_MIN_CONFIDENCE", "0.30")))
    result = []
    for source, page in zip(pages, canonical, strict=True):
        lines = []
        for line in page.lines:
            points = line.polygon
            # Preserve the existing normalized polygon contract for extraction.
            area = sum(
                points[i][0] * points[(i + 1) % 4][1]
                - points[(i + 1) % 4][0] * points[i][1]
                for i in range(4)
            )
            ordered = points if area >= 0 else list(reversed(points))
            item = line.model_dump()
            item["pixel_polygon"] = item.pop("polygon")
            item["polygon"] = [
                [max(0.0, min(1.0, x / page.width)), max(0.0, min(1.0, y / page.height))]
                for x, y in ordered
            ]
            lines.append(item)
        result.append({
            "page_number": page.page, "width": page.width, "height": page.height,
            "rotation": page.rotation, "duration_ms": source["duration_ms"],
            "text": page.full_text, "average_confidence": page.average_confidence,
            "low_confidence_count": page.low_confidence_count, "line_count": page.line_count,
            "lines": lines,
        })
    return result


def run_job(protocol: Any, claim: dict[str, Any], infer: Callable[[Path, float], list[dict[str, Any]]], temp_root: Path) -> dict[str, int]:
    payload = claim["job"]["payload"]
    if payload.get("model_version") != MODEL_VERSION or payload.get("preprocessing_version") != PREPROCESSING_VERSION:
        raise RuntimeError("incompatible_ocr_version")
    started = time.monotonic()
    stop = threading.Event()
    lease_errors = []

    def heartbeat() -> None:
        while not stop.wait(30):
            try:
                protocol.heartbeat(claim)
            except Exception as error:
                lease_errors.append(error)
                break

    pulse = threading.Thread(target=heartbeat, daemon=True)
    pulse.start()
    try:
        with tempfile.TemporaryDirectory(prefix="attempt-", dir=temp_root) as directory:
            os.chmod(directory, 0o700)
            if getattr(infer, "remote", False):
                # The API owns authorization and decrypts the source; only a
                # short-lived, lease-bound input capability crosses to RunPod.
                digest = payload["sha256"]
                raw_pages = infer(claim, protocol.delegation(claim), started + 900, protocol)
            else:
                original = Path(directory) / "original"
                metadata = protocol.download(claim, original)
                if not 0 < original.stat().st_size <= MAX_FILE:
                    raise ValueError("invalid_input")
                with original.open("rb") as body:
                    digest = hashlib.file_digest(body, "sha256").hexdigest()
                if digest != metadata["sha256"]:
                    raise ValueError("invalid_input")
                raw_pages = infer(original, started + 900)
            try:
                pages = normalize(raw_pages)
            except (ValueError, TypeError, KeyError) as error:
                if not getattr(infer, "remote", False):
                    raise
                raise RuntimeError("invalid_runpod_output") from error
            if lease_errors:
                raise lease_errors[0]
            if time.monotonic() > started + 900:
                raise TimeoutError()
            result = {
                "schema_version": 2,
                "input_sha256": digest,
                "model_version": payload["model_version"],
                "preprocessing_version": payload["preprocessing_version"],
                "duration_ms": round((time.monotonic() - started) * 1000),
                "pages": pages,
            }
            artifact = Path(directory) / "result.json"
            with artifact.open("w", encoding="utf-8") as output:
                json.dump(
                    result,
                    output,
                    ensure_ascii=False,
                    allow_nan=False,
                    separators=(",", ":"),
                )
            if artifact.stat().st_size > MAX_RESULT:
                raise ValueError("invalid_input")
            protocol.heartbeat(claim)
            with artifact.open("rb") as body:
                protocol.submit(claim, body)
            return {"pages": len(pages), "duration_ms": result["duration_ms"]}
    finally:
        stop.set()
        pulse.join(timeout=35)
