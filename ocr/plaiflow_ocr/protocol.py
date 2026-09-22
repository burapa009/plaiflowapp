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

    def request(self, method, path, scope, claim=None, data=None):
        if not path.startswith("/internal/v1/ocr/jobs/") or path.startswith("//"):
            raise ValueError("unauthorized_input")
        claims = {
            "worker_id": self.worker,
            "environment": self.environment,
            "scopes": ["ocr:" + scope],
            "iat": int(time.time()),
            "exp": int(time.time()) + 120,
            "nonce": secrets.token_hex(16),
        }
        encoded = base64.urlsafe_b64encode(
            json.dumps(claims, separators=(",", ":")).encode()
        ).rstrip(b"=")
        signature = base64.urlsafe_b64encode(
            hmac.digest(self.key, encoded, "sha256")
        ).rstrip(b"=")
        headers = {
            "Authorization": "Bearer " + (encoded + b"." + signature).decode(),
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

    def claim(self):
        with self.request(
            "POST", "/internal/v1/ocr/jobs/claim", "claim", data=b"{}"
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
