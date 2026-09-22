import json
import os
from pathlib import Path
import random
import shutil
import signal
import threading
import time

from .engine import Engine
from .protocol import Protocol
from .worker import run_job


def main():
    stop = threading.Event()
    signal.signal(signal.SIGTERM, lambda *_: stop.set())
    signal.signal(signal.SIGINT, lambda *_: stop.set())
    root = Path(os.environ.get("OCR_TEMP_DIR", "/tmp/plaiflow-ocr"))
    root.mkdir(mode=0o700, parents=True, exist_ok=True)
    for entry in root.glob("attempt-*"):
        if (
            not entry.is_symlink()
            and entry.is_dir()
            and time.time() - entry.stat().st_mtime > 3600
        ):
            shutil.rmtree(entry)
    protocol = Protocol()
    engine = Engine()
    engine.start()
    delay = 1
    try:
        while not stop.is_set():
            try:
                jobs = protocol.claim()
                if not jobs:
                    stop.wait(delay + random.random())
                    delay = min(15, delay * 2)
                    continue
                delay = 1
                claim = jobs[0]
                try:
                    metrics = run_job(protocol, claim, engine, root)
                    print(
                        json.dumps(
                            {
                                "event": "ocr.completed",
                                "job_id": claim["job"]["id"],
                                "peak_rss": engine.peak_rss,
                                **metrics,
                            }
                        ),
                        flush=True,
                    )
                except Exception as error:
                    code = (
                        "invalid_input"
                        if isinstance(error, ValueError)
                        else "temporary_upstream"
                    )
                    print(
                        json.dumps(
                            {
                                "event": "ocr.failed",
                                "job_id": claim["job"]["id"],
                                "code": code,
                            }
                        ),
                        flush=True,
                    )
                    protocol.fail(claim, code)
            except Exception:
                print(json.dumps({"event": "ocr.protocol_unavailable"}), flush=True)
                stop.wait(15)
    finally:
        engine.close()


if __name__ == "__main__":
    main()
