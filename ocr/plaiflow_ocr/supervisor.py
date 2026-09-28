"""Run the private OCR API beside the leased worker when Railway has one slot."""

import os
import signal
import subprocess
import sys
import time
import urllib.error
import urllib.request


def main() -> None:
    worker_env = os.environ.copy()
    if worker_env.get("OCR_SERVICE_URL"):
        from .__main__ import main as run_worker

        run_worker()
        return

    worker_env["OCR_SERVICE_URL"] = "http://127.0.0.1:8000"
    api_env = worker_env.copy()
    api_env["OCR_BIND_HOST"] = "127.0.0.1"
    api_env.pop("OCR_SERVICE_URL")
    stopping = False

    def stop(*_: object) -> None:
        nonlocal stopping
        stopping = True

    signal.signal(signal.SIGTERM, stop)
    signal.signal(signal.SIGINT, stop)
    processes: list[subprocess.Popen[bytes]] = []
    try:
        api = subprocess.Popen([sys.executable, "-m", "plaiflow_ocr.api"], env=api_env)
        processes.append(api)
        deadline = time.monotonic() + 240
        while not stopping and time.monotonic() < deadline and api.poll() is None:
            try:
                with urllib.request.urlopen("http://127.0.0.1:8000/health", timeout=2) as response:
                    if response.status == 200:
                        break
            except (urllib.error.URLError, TimeoutError):
                time.sleep(1)
        else:
            raise RuntimeError("ocr_api_not_ready")
        worker = subprocess.Popen([sys.executable, "-m", "plaiflow_ocr"], env=worker_env)
        processes.append(worker)
        while not stopping and all(process.poll() is None for process in processes):
            time.sleep(1)
        if not stopping:
            raise RuntimeError("ocr_process_exited")
    finally:
        for process in processes:
            if process.poll() is None:
                process.terminate()
        for process in processes:
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()


if __name__ == "__main__":
    main()
