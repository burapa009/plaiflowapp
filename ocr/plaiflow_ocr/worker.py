import hashlib
import json
import math
import os
import tempfile
import threading
import time
from pathlib import Path

MAX_FILE = 20 << 20
MAX_RESULT = 8 << 20
MAX_TEMP = 512 << 20


def normalize(pages):
    if not 1 <= len(pages) <= 20:
        raise ValueError("invalid_input")
    for number, page in enumerate(pages, 1):
        width, height = page["width"], page["height"]
        if (
            width <= 0
            or height <= 0
            or width * height > 25_000_000
            or len(page["lines"]) > 5000
        ):
            raise ValueError("invalid_input")
        page["page_number"] = number
        for line in page["lines"]:
            if (
                len(line["text"].encode("utf-8")) > 16384
                or not 0 <= line["confidence"] <= 1
            ):
                raise ValueError("invalid_input")
            points = line["polygon"]
            if len(points) != 4 or any(
                not math.isfinite(v) for point in points for v in point
            ):
                raise ValueError("invalid_input")
            # Image coordinates have y pointing down; positive signed area is clockwise.
            area = sum(
                points[i][0] * points[(i + 1) % 4][1]
                - points[(i + 1) % 4][0] * points[i][1]
                for i in range(4)
            )
            if area < 0:
                points = list(reversed(points))
            line["polygon"] = [
                [max(0.0, min(1.0, x / width)), max(0.0, min(1.0, y / height))]
                for x, y in points
            ]
        page["text"] = "\n".join(line["text"] for line in page["lines"])
    return pages


def run_job(protocol, claim, infer, temp_root):
    started = time.monotonic()
    stop = threading.Event()
    lease_errors = []

    def heartbeat():
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
            original = Path(directory) / "original"
            metadata = protocol.download(claim, original)
            if not 0 < original.stat().st_size <= MAX_FILE:
                raise ValueError("invalid_input")
            with original.open("rb") as body:
                digest = hashlib.file_digest(body, "sha256").hexdigest()
            if digest != metadata["sha256"]:
                raise ValueError("invalid_input")
            pages = normalize(infer(original, started + 900))
            if lease_errors:
                raise lease_errors[0]
            if time.monotonic() > started + 900:
                raise TimeoutError()
            payload = claim["job"]["payload"]
            result = {
                "schema_version": 1,
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
