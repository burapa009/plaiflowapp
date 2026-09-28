import io
import json
import os
import tempfile
import time
import unittest
from pathlib import Path
from unittest.mock import patch

from plaiflow_ocr.client import ServiceClient
from plaiflow_ocr.worker import normalize


class ClientTest(unittest.TestCase):
    def test_private_url_and_structured_response_bridge(self):
        response = {"success": True, "pages": [{
            "page": 1, "width": 100, "height": 100, "rotation": 0, "duration_ms": 3,
            "lines": [{"text": "ไทย Invoice", "confidence": .9,
                       "polygon": [[1, 2], [40, 2], [40, 12], [1, 12]]}],
        }]}
        with patch.dict(os.environ, {"OCR_SERVICE_URL": "http://ocr.railway.internal:8000", "OCR_SERVICE_TOKEN": "t" * 32}):
            client = ServiceClient()
            case = self
            class Opener:
                def open(self, request, timeout):
                    case.assertEqual(request.get_header("Authorization"), "Bearer " + "t" * 32)
                    case.assertIn(b"Content-Type: image/png", request.data)
                    return io.BytesIO(json.dumps(response).encode())
            client.opener = Opener()
            with tempfile.TemporaryDirectory() as directory:
                path = Path(directory) / "image"
                path.write_bytes(b"\x89PNG\r\n\x1a\n" + b"data")
                pages = client(path, time.monotonic() + 30)
            self.assertEqual(pages[0]["lines"][0]["polygon"][0], [1, 2])
            artifact_page = normalize(pages)[0]
            self.assertEqual(artifact_page["duration_ms"], 3)
            self.assertEqual(artifact_page["lines"][0]["bbox"]["x_min"], 1)
            class RedirectingOpener:
                def open(self, request, timeout):
                    raise ValueError("unauthorized_input")
            client.opener = RedirectingOpener()
            with tempfile.TemporaryDirectory() as directory:
                path = Path(directory) / "image"
                path.write_bytes(b"\x89PNG\r\n\x1a\n" + b"data")
                with self.assertRaises(RuntimeError):
                    client(path, time.monotonic() + 30)
            class MalformedOpener:
                def open(self, request, timeout):
                    return io.BytesIO(b'{"success":true,"pages":[{}]}')
            client.opener = MalformedOpener()
            with tempfile.TemporaryDirectory() as directory:
                path = Path(directory) / "image"
                path.write_bytes(b"\x89PNG\r\n\x1a\n" + b"data")
                with self.assertRaisesRegex(RuntimeError, "temporary_upstream"):
                    client(path, time.monotonic() + 30)
        with patch.dict(os.environ, {"OCR_SERVICE_URL": "http://public.example:8000", "OCR_SERVICE_TOKEN": "t" * 32}):
            with self.assertRaises(ValueError):
                ServiceClient()


if __name__ == "__main__":
    unittest.main()
