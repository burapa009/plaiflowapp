import io
import os
import unittest
from unittest.mock import patch

from fastapi.testclient import TestClient
from PIL import Image
import pypdfium2 as pdfium

from plaiflow_ocr.api import app


TOKEN = "t" * 32


class FakeEngine:
    def __init__(self, max_pages=20):
        self.max_pages = max_pages

    def start(self):
        pass

    def close(self):
        pass

    def __call__(self, path, deadline):
        with path.open("rb") as source:
            is_pdf = source.read(5) == b"%PDF-"
        document = pdfium.PdfDocument(str(path)) if is_pdf else None
        try:
            count = len(document) if document is not None else 1
        finally:
            if document is not None:
                document.close()
        return [{"width": 100, "height": 100, "rotation": 0, "duration_ms": 3, "lines": [
            {"text": "ไทย Invoice", "confidence": 0.8, "polygon": [[1, 2], [40, 2], [40, 12], [1, 12]]},
        ]} for _ in range(count)]


class APITest(unittest.TestCase):
    def test_auth_upload_response_and_invalid_files(self):
        picture = io.BytesIO()
        Image.new("RGB", (1, 1), "white").save(picture, format="WEBP")
        with patch.dict(os.environ, {"OCR_SERVICE_TOKEN": TOKEN, "OCR_MAX_FILE_SIZE_MB": "1"}), patch("plaiflow_ocr.api.Engine", FakeEngine), TestClient(app) as client:
            file = {"file": ("invoice.webp", picture.getvalue(), "image/webp")}
            self.assertEqual(client.post("/v1/ocr", files=file).status_code, 401)
            self.assertEqual(client.post("/v1/ocr", content=b"not multipart").status_code, 401)
            response = client.post("/v1/ocr", files=file, headers={"Authorization": "Bearer " + TOKEN})
            self.assertEqual(response.status_code, 200, response.text)
            result = response.json()
            self.assertEqual(result["pages"][0]["lines"][0]["polygon"], [[1, 2], [40, 2], [40, 12], [1, 12]])
            self.assertEqual(result["pages"][0]["full_text"], "ไทย Invoice")
            self.assertEqual(result["engine"]["name"], "paddleocr")
            self.assertNotIn("debug", result)
            bad = client.post("/v1/ocr", files={"file": ("bad.png", b"garbage", "image/png")}, headers={"Authorization": "Bearer " + TOKEN})
            self.assertEqual(bad.json()["error"]["code"], "UNSUPPORTED_FILE_TYPE")
            mismatch = client.post("/v1/ocr", files={"file": ("bad.png", picture.getvalue(), "image/png")}, headers={"Authorization": "Bearer " + TOKEN})
            self.assertEqual(mismatch.json()["error"]["code"], "INVALID_FILE")
            large = client.post("/v1/ocr", files={"file": ("big.webp", picture.getvalue() + b"x" * (1 << 20), "image/webp")}, headers={"Authorization": "Bearer " + TOKEN})
            self.assertEqual(large.status_code, 413)
            self.assertEqual(client.get("/health").json(), {"status": "ok"})

    def test_pdf_pages_and_limit(self):
        document = io.BytesIO()
        first = Image.new("RGB", (100, 100), "white")
        first.save(document, format="PDF", save_all=True, append_images=[first])
        file = {"file": ("two.pdf", document.getvalue(), "application/pdf")}
        with patch.dict(os.environ, {"OCR_SERVICE_TOKEN": TOKEN, "OCR_MAX_PDF_PAGES": "2"}), patch("plaiflow_ocr.api.Engine", FakeEngine), TestClient(app) as client:
            response = client.post("/v1/ocr", files=file, headers={"Authorization": "Bearer " + TOKEN})
            self.assertEqual(response.status_code, 200, response.text)
            self.assertEqual([page["page"] for page in response.json()["pages"]], [1, 2])
        with patch.dict(os.environ, {"OCR_SERVICE_TOKEN": TOKEN, "OCR_MAX_PDF_PAGES": "1"}), patch("plaiflow_ocr.api.Engine", FakeEngine), TestClient(app) as client:
            response = client.post("/v1/ocr", files=file, headers={"Authorization": "Bearer " + TOKEN})
            self.assertEqual(response.json()["error"]["code"], "PDF_PAGE_LIMIT_EXCEEDED")


if __name__ == "__main__":
    unittest.main()
