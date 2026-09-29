"""Keep Ollama local to the GPU worker; terminate both children together."""
import subprocess
import time
import urllib.request

server = subprocess.Popen(["ollama", "serve"])
try:
    deadline = time.monotonic() + 90
    while True:
        if server.poll() is not None:
            raise RuntimeError("openthai_server_failed")
        try:
            with urllib.request.urlopen("http://127.0.0.1:11434/api/version", timeout=2):
                break
        except OSError:
            if time.monotonic() >= deadline:
                raise RuntimeError("openthai_server_timeout") from None
            time.sleep(1)
    import runpy
    runpy.run_path("runpod_handler.py", run_name="__main__")
finally:
    server.terminate()
    try:
        server.wait(timeout=10)
    except subprocess.TimeoutExpired:
        server.kill()
        server.wait()
