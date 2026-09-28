"""RunPod queue adapter; the existing PlaiFlow lease remains authoritative."""

import json
import os
import re
import time
import urllib.error
import urllib.request

from .protocol import NoRedirect


class RunPodClient:
    remote = True
    peak_rss = 0

    def __init__(self):
        endpoint = os.environ["RUNPOD_ENDPOINT_ID"]
        key = os.environ["RUNPOD_API_KEY"]
        if not re.fullmatch(r"[A-Za-z0-9_-]{3,100}", endpoint) or not key:
            raise ValueError("invalid_runpod_configuration")
        self.base = "https://api.runpod.ai/v2/" + endpoint
        self.key = key
        self.provider_job_id = ""
        self.provider_execution_ms = 0
        self.provider_queue_ms = 0
        self.opener = urllib.request.build_opener(NoRedirect())

    def start(self):
        pass

    def close(self):
        pass

    def _request(self, method, path, body=None):
        data = None if body is None else json.dumps(body, separators=(",", ":")).encode()
        request = urllib.request.Request(
            self.base + path,
            data=data,
            method=method,
            headers={
                "Authorization": "Bearer " + self.key,
                "Content-Type": "application/json",
            },
        )
        try:
            with self.opener.open(request, timeout=30) as response:
                try:
                    return json.loads(response.read(9 << 20))
                except ValueError:
                    raise RuntimeError("invalid_runpod_response") from None
        except urllib.error.HTTPError as error:
            # Provider bodies can contain document data or credentials.
            raise RuntimeError("runpod_http_" + str(error.code)) from None

    def __call__(self, claim, delegation, deadline, protocol):
        job = claim["job"]
        payload = job["payload"]
        self.provider_job_id = ""
        self.provider_execution_ms = 0
        self.provider_queue_ms = 0
        result = self._request("POST", "/run", {
            "input": {
                "job_id": job["id"],
                "document_id": payload["document_id"],
                "organization_id": job["organization_id"],
                "sha256": payload["sha256"],
                "model_version": payload["model_version"],
                "preprocessing_version": payload["preprocessing_version"],
                "correlation_id": job["id"],
                **delegation,
            },
            "policy": {"executionTimeout": 600000, "ttl": 840000},
        })
        provider_id = result.get("id")
        if not isinstance(provider_id, str) or not re.fullmatch(r"[A-Za-z0-9_-]{1,128}", provider_id):
            raise RuntimeError("invalid_runpod_job_id")
        self.provider_job_id = provider_id
        try:
            protocol.record_provider(claim, provider_id)
        except Exception:
            # Keep polling the already-billed provider job after a brief DB outage.
            pass
        while time.monotonic() < deadline:
            status = self._request("GET", "/status/" + provider_id)
            state = status.get("status")
            if state == "COMPLETED":
                output = status.get("output")
                if not isinstance(output, dict) or output.get("schema_version") != "runpod.raw.v1" or output.get("job_id") != job["id"] or output.get("input_sha256") != payload["sha256"] or output.get("model_version") != payload["model_version"] or output.get("preprocessing_version") != payload["preprocessing_version"] or not isinstance(output.get("pages"), list):
                    raise RuntimeError("invalid_runpod_output")
                try:
                    execution_ms = max(0, int(status.get("executionTime") or 0))
                    queue_ms = max(0, int(status.get("delayTime") or 0))
                except (ValueError, TypeError, OverflowError):
                    raise RuntimeError("invalid_runpod_metrics") from None
                self.provider_execution_ms = execution_ms
                self.provider_queue_ms = queue_ms
                protocol.record_provider(claim, provider_id, execution_ms, queue_ms,
                                         os.environ.get("RUNPOD_GPU_CLASS", ""))
                return output["pages"]
            if state in {"FAILED", "CANCELLED", "TIMED_OUT"}:
                raise RuntimeError("runpod_" + state.lower())
            if state not in {"IN_QUEUE", "IN_PROGRESS", "RUNNING"}:
                raise RuntimeError("invalid_runpod_status")
            time.sleep(min(3, max(0, deadline - time.monotonic())))
        raise TimeoutError("runpod_timeout")
