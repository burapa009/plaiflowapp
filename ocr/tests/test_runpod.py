import hashlib
import io
import json
import os
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from plaiflow_ocr.runpod_client import RunPodClient
from plaiflow_ocr.worker import MODEL_VERSION, PREPROCESSING_VERSION, run_job
from runpod_handler import build_handler


class FakeProtocol:
    def __init__(self):
        self.result = None
        self.provider_job_id = None

    def delegation(self, claim):
        return {"api_url": "https://api.example.test", "input_auth": "a", "original_auth": "b", "lease_token": "lease", "attempt_id": "a" * 36}

    def heartbeat(self, claim):
        pass

    def record_provider(self, claim, provider_job_id, *metrics):
        self.provider_job_id = provider_job_id

    def submit(self, claim, body):
        self.result = json.load(body)


class RunPodTest(unittest.TestCase):
    def setUp(self):
        self.digest = hashlib.sha256(b"image").hexdigest()
        self.claim = {"job": {"id": "1" * 36, "organization_id": "2" * 36, "attempt_id": "a" * 36,
            "payload": {"document_id": "3" * 36, "sha256": self.digest,
                "model_version": MODEL_VERSION, "preprocessing_version": PREPROCESSING_VERSION}}}
        self.pages = [{"page_number": 1, "width": 100, "height": 100, "rotation": 0,
            "duration_ms": 10, "lines": [{"text": "ใบเสร็จ 100.00", "confidence": .9,
                "polygon": [[0, 0], [50, 0], [50, 10], [0, 10]]}]}]

    def test_remote_result_uses_existing_v2_artifact_and_is_idempotent_at_api_boundary(self):
        protocol = FakeProtocol()
        with patch.dict(os.environ, {"RUNPOD_ENDPOINT_ID": "endpoint1", "RUNPOD_API_KEY": "secret"}):
            client = RunPodClient()
        output = {"schema_version": "runpod.raw.v1", "job_id": self.claim["job"]["id"], "input_sha256": self.digest,
            "model_version": MODEL_VERSION, "preprocessing_version": PREPROCESSING_VERSION,
            "pages": self.pages}
        with patch.object(client, "_request", side_effect=[{"id": "provider123"}, {"status": "COMPLETED", "output": output}]):
            with tempfile.TemporaryDirectory() as directory:
                run_job(protocol, self.claim, client, Path(directory))
                self.assertEqual(list(Path(directory).iterdir()), [])
        self.assertEqual(protocol.provider_job_id, "provider123")
        self.assertEqual(protocol.result["schema_version"], 2)
        self.assertEqual(protocol.result["input_sha256"], self.digest)
        self.assertEqual(protocol.result["pages"][0]["lines"][0]["text"], "ใบเสร็จ 100.00")

    def test_malformed_provider_result_never_publishes(self):
        protocol = FakeProtocol()
        with patch.dict(os.environ, {"RUNPOD_ENDPOINT_ID": "endpoint1", "RUNPOD_API_KEY": "secret"}):
            client = RunPodClient()
        with patch.object(client, "_request", side_effect=[{"id": "provider123"},
            {"status": "COMPLETED", "output": {"job_id": "other", "pages": self.pages}}]):
            with tempfile.TemporaryDirectory() as directory:
                with self.assertRaises(RuntimeError):
                    run_job(protocol, self.claim, client, Path(directory))
        self.assertIsNone(protocol.result)

    def test_malformed_provider_pages_are_retryable(self):
        protocol = FakeProtocol()
        with patch.dict(os.environ, {"RUNPOD_ENDPOINT_ID": "endpoint1", "RUNPOD_API_KEY": "secret"}):
            client = RunPodClient()
        output = {"schema_version": "runpod.raw.v1", "job_id": self.claim["job"]["id"], "input_sha256": self.digest,
            "model_version": MODEL_VERSION, "preprocessing_version": PREPROCESSING_VERSION,
            "pages": [{"page_number": 1, "lines": "broken"}]}
        with patch.object(client, "_request", side_effect=[{"id": "provider123"}, {"status": "COMPLETED", "output": output}]):
            with tempfile.TemporaryDirectory() as directory:
                with self.assertRaises(RuntimeError):
                    run_job(protocol, self.claim, client, Path(directory))
        self.assertIsNone(protocol.result)

    def test_handler_rejects_untrusted_api_and_fetches_only_lease_bound_source(self):
        class FakeEngine:
            def __call__(self, path, deadline):
                self_content = path.read_bytes()
                if self_content != b"image":
                    raise AssertionError("wrong input")
                return self_pages

        self_pages = self.pages
        metadata = json.dumps({"download_url": "/internal/v1/ocr/jobs/" + self.claim["job"]["id"] + "/original?token=signed",
            "mime": "image/png", "size": 5, "sha256": self.digest}).encode()

        class FakeOpener:
            def __init__(self):
                self.calls = []

            def open(self, request, timeout):
                self.calls.append(request.full_url)
                return io.BytesIO(metadata if request.full_url.endswith("/input") else b"image")

        data = {"job_id": self.claim["job"]["id"], "document_id": "3" * 36, "organization_id": "2" * 36,
            "model_version": MODEL_VERSION, "preprocessing_version": PREPROCESSING_VERSION,
            "api_url": "https://api.example.test", "sha256": self.digest,
            "attempt_id": "a" * 36, "lease_token": "lease", "input_auth": "a", "original_auth": "b"}
        with patch.dict(os.environ, {"OCR_INPUT_API_URL": "https://api.example.test"}):
            opener = FakeOpener()
            with patch("runpod_handler.urllib.request.build_opener", return_value=opener):
                handler = build_handler(FakeEngine())
            with self.assertRaises(ValueError):
                handler({"input": {**data, "api_url": "https://attacker.test"}})
            result = handler({"input": data})
        self.assertEqual(result["pages"], self.pages)
        self.assertEqual(len(opener.calls), 2)
        self.assertTrue(all(url.startswith("https://api.example.test/internal/v1/ocr/jobs/") for url in opener.calls))


if __name__ == "__main__":
    unittest.main()
