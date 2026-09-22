"""Record real measurements; absent human reference text leaves accuracy unverified."""

import argparse
import hashlib
import json
import math
import platform
import statistics
import time
from pathlib import Path

from .engine import Engine
from .worker import normalize


def edit_distance(reference, actual):
    previous = list(range(len(actual) + 1))
    for i, a in enumerate(reference, 1):
        current = [i]
        for j, b in enumerate(actual, 1):
            current.append(
                min(previous[j] + 1, current[-1] + 1, previous[j - 1] + (a != b))
            )
        previous = current
    return previous[-1]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("directory", type=Path)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    engine = Engine()
    measurements = []
    seen = set()
    engine.start()
    try:
        for path in sorted(args.directory.iterdir()):
            if path.suffix.lower() not in (".jpg", ".jpeg", ".png", ".pdf"):
                continue
            with path.open("rb") as source:
                digest = hashlib.file_digest(source, "sha256").hexdigest()
            if digest in seen:
                continue
            seen.add(digest)
            started = time.monotonic()
            record = {"sha256": digest, "format": path.suffix.lower()}
            try:
                pages = normalize(engine(path, time.monotonic() + 900))
                record.update(
                    {
                        "status": "completed",
                        "pages": len(pages),
                        "seconds": time.monotonic() - started,
                        "peak_rss_bytes": engine.peak_rss,
                        "empty_pages": sum(not p["text"] for p in pages),
                    }
                )
                reference = path.with_suffix(path.suffix + ".txt")
                if reference.is_file():
                    truth = (
                        reference.read_text(encoding="utf-8")
                        .replace("\r\n", "\n")
                        .strip()
                    )
                    actual = "\n".join(p["text"] for p in pages).strip()
                    record["cer"] = edit_distance(truth, actual) / max(1, len(truth))
            except Exception as error:
                record.update(
                    {
                        "status": "failed",
                        "error_type": type(error).__name__,
                        "seconds": time.monotonic() - started,
                    }
                )
            measurements.append(record)
            print(
                json.dumps({"index": len(measurements), "status": record["status"]}),
                flush=True,
            )
    finally:
        engine.close()
    seconds = sorted(
        m["seconds"] / m["pages"] for m in measurements if m["status"] == "completed"
    )
    report = {
        "platform": platform.platform(),
        "runtime": "Python 3.12 / PaddleOCR 3.7.0 / PaddlePaddle 3.2.2",
        "cpu_threads": 2,
        "container_cpu_limit_verified": False,
        "acceptance_passed": False,
        "accuracy_status": "unverified unless independently reviewed reference files are supplied",
        "unique_files": len(measurements),
        "human_reference_count": sum("cer" in m for m in measurements),
        "p50_seconds_per_page": statistics.median(seconds) if seconds else None,
        "p95_seconds_per_page": seconds[math.ceil(len(seconds) * 0.95) - 1]
        if seconds
        else None,
        "measurements": measurements,
    }
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2), encoding="utf-8")


if __name__ == "__main__":
    main()
