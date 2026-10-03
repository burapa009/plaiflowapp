import base64
import hashlib
import hmac
import json
import os
import secrets
import time
import urllib.parse
import urllib.request

from .worker import MAX_FILE


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise ValueError("unauthorized_input")


class Protocol:
    def __init__(self):
        self.base = os.environ["JOB_API_URL"].rstrip("/")
        self.environment = os.environ["APP_ENV"]
        self.worker = os.environ["OCR_WORKER_ID"]
        self.key = base64.b64decode(os.environ["OCR_WORKER_AUTH_KEY"], validate=True)
        if len(self.key) != 32 or urllib.parse.urlsplit(self.base).scheme != "https":
            raise ValueError("invalid_worker_configuration")
        self.opener = urllib.request.build_opener(NoRedirect())

    def token(self, scope, ttl=120):
        if scope not in {"claim", "heartbeat", "input", "submit", "fail"} or not 0 < ttl <= (900 if scope == "input" else 300):
            raise ValueError("invalid_worker_scope")
        now = int(time.time())
        claims = {
            "worker_id": self.worker,
            "environment": self.environment,
            "scopes": ["ocr:" + scope],
            "iat": now,
            "exp": now + ttl,
            "nonce": secrets.token_hex(16),
        }
        encoded = base64.urlsafe_b64encode(
            json.dumps(claims, separators=(",", ":")).encode()
        ).rstrip(b"=")
        signature = base64.urlsafe_b64encode(
            hmac.digest(self.key, encoded, "sha256")
        ).rstrip(b"=")
        return (encoded + b"." + signature).decode()

    def request(self, method, path, scope, claim=None, data=None):
        if not path.startswith("/internal/v1/ocr/jobs/") or path.startswith("//"):
            raise ValueError("unauthorized_input")
        headers = {
            "Authorization": "Bearer " + self.token(scope),
            "Content-Type": "application/json",
        }
        if claim is not None:
            headers.update(
                {
                    "X-Job-Attempt": claim["job"]["attempt_id"],
                    "X-Job-Lease": claim["lease_token"],
                }
            )
        return self.opener.open(
            urllib.request.Request(
                self.base + path, data=data, headers=headers, method=method
            ),
            timeout=30,
        )

    def delegation(self, claim):
        # Each scoped token is single-use; RunPod never receives the signing key.
        return {
            "api_url": self.base,
            "input_auth": self.token("input", 900),
            "original_auth": self.token("input", 900),
            "lease_token": claim["lease_token"],
            "attempt_id": claim["job"]["attempt_id"],
            "worker_id": self.worker,
        }

    def record_provider(self, claim, provider_job_id, execution_ms=0, queue_ms=0, gpu_class=""):
        path = "/internal/v1/ocr/jobs/" + claim["job"]["id"] + "/provider"
        with self.request(
            "POST", path, "heartbeat", claim,
            data=json.dumps({"provider_job_id": provider_job_id, "execution_ms": execution_ms,
                             "queue_ms": queue_ms, "gpu_class": gpu_class}).encode(),
        ):
            pass

    def claim(self, job_id=None):
        provider = os.environ.get("OCR_PROVIDER", "railway")
        if provider not in {"railway", "runpod"}:
            raise ValueError("invalid_ocr_provider")
        data = {"provider": provider}
        if job_id is not None:
            data["job_id"] = job_id
        with self.request(
            "POST", "/internal/v1/ocr/jobs/claim", "claim",
            data=json.dumps(data).encode(),
        ) as response:
            return json.loads(response.read(1 << 20))["jobs"]

    def download(self, claim, destination):
        prefix = "/internal/v1/ocr/jobs/" + claim["job"]["id"]
        with self.request("GET", prefix + "/input", "input", claim) as response:
            metadata = json.loads(response.read(16384))
        if (
            not 0 < metadata["size"] <= MAX_FILE
            or metadata["download_url"].split("?")[0] != prefix + "/original"
        ):
            raise ValueError("invalid_input")
        digest = hashlib.sha256()
        total = 0
        with (
            self.request("GET", metadata["download_url"], "input", claim) as response,
            destination.open("xb") as output,
        ):
            while chunk := response.read(65536):
                total += len(chunk)
                if total > metadata["size"]:
                    raise ValueError("invalid_input")
                digest.update(chunk)
                output.write(chunk)
        if total != metadata["size"] or digest.hexdigest() != metadata["sha256"]:
            raise ValueError("invalid_input")
        return metadata

    def heartbeat(self, claim):
        with self.request(
            "POST",
            "/internal/v1/ocr/jobs/" + claim["job"]["id"] + "/heartbeat",
            "heartbeat",
            claim,
            data=b"{}",
        ):
            pass

    def submit(self, claim, body):
        # Result has an 8 MiB contract bound; original files remain streamed.
        with self.request(
            "PUT",
            "/internal/v1/ocr/jobs/" + claim["job"]["id"] + "/result",
            "submit",
            claim,
            data=body.read(),
        ) as response:
            if json.loads(response.read(1024))["status"] != "Completed":
                raise RuntimeError("publication_failed")

    def fail(self, claim, code):
        with self.request(
            "POST",
            "/internal/v1/ocr/jobs/" + claim["job"]["id"] + "/fail",
            "fail",
            claim,
            data=json.dumps({"code": code}).encode(),
        ):
            pass
