import hashlib
import json
import tempfile
import unittest
from pathlib import Path

from plaiflow_ocr.worker import run_job


class Protocol:
    def __init__(self, content=b"image"):
        self.content = content
        self.result = None

    def download(self, claim, destination):
        destination.write_bytes(self.content)
        return {"sha256": hashlib.sha256(self.content).hexdigest(), "mime": "image/png"}

    def heartbeat(self, claim):
        pass

    def submit(self, claim, body):
        self.result = json.load(body)


class WorkerTest(unittest.TestCase):
    def test_download_ocr_normalize_submit_cleans_original(self):
        protocol = Protocol()
        claim = {
            "job": {
                "id": "job",
                "payload": {"model_version": "v1", "preprocessing_version": "v1"},
            }
        }

        def infer(path, deadline):
            self.assertEqual(path.read_bytes(), b"image")
            return [
                {
                    "page_number": 1,
                    "width": 100,
                    "height": 200,
                    "rotation": 0,
                    "duration_ms": 12,
                    "lines": [
                        {
                            "text": "ทดสอบ 123",
                            "confidence": 0.9,
                            "polygon": [[10, 20], [90, 20], [90, 40], [10, 40]],
                        }
                    ],
                }
            ]

        with tempfile.TemporaryDirectory() as root:
            run_job(protocol, claim, infer, Path(root))
            self.assertEqual(list(Path(root).iterdir()), [])
        page = protocol.result["pages"][0]
        self.assertEqual(page["text"], "ทดสอบ 123")
        self.assertEqual(
            page["lines"][0]["polygon"],
            [[0.1, 0.1], [0.9, 0.1], [0.9, 0.2], [0.1, 0.2]],
        )
        self.assertEqual(protocol.result["schema_version"], 1)

    def test_failed_inference_never_submits_partial_result(self):
        protocol = Protocol()

        def fail(path, deadline):
            raise TimeoutError()

        with tempfile.TemporaryDirectory() as root:
            with self.assertRaises(TimeoutError):
                run_job(protocol, {"job": {"payload": {}}}, fail, Path(root))
            self.assertEqual(list(Path(root).iterdir()), [])
        self.assertIsNone(protocol.result)


if __name__ == "__main__":
    unittest.main()
