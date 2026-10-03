import unittest

from plaiflow_ocr.normalize import build_ai_payload, build_ai_text, normalize_pages
from plaiflow_ocr.worker import normalize


class NormalizeTest(unittest.TestCase):
    def test_geometry_order_confidence_and_unchanged_thai_english(self):
        page = {"width": 1200, "height": 1600, "rotation": 0, "duration_ms": 10, "lines": [
            {"text": "ยอดรวม 1,070.00", "confidence": 0.43, "polygon": [[710, 1000], [1100, 998], [1100, 1040], [710, 1042]]},
            {"text": "Invoice", "confidence": 0.98, "polygon": [[800, 50], [1100, 50], [1100, 90], [800, 90]]},
            {"text": "บริษัท ABC 01055581234S6", "confidence": 0.29, "polygon": [[100, 52], [600, 48], [602, 92], [102, 96]]},
        ]}
        got = normalize_pages([page])[0]
        self.assertEqual([line.text for line in got.lines], ["บริษัท ABC 01055581234S6", "Invoice", "ยอดรวม 1,070.00"])
        self.assertEqual([line.reading_order for line in got.lines], [1, 2, 3])
        self.assertEqual([line.page_number for line in got.lines], [1, 1, 1])
        self.assertEqual(got.lines[0].bbox.model_dump(), {"x_min": 100, "y_min": 48, "x_max": 602, "y_max": 96})
        self.assertEqual(got.lines[0].normalized_bbox.x_min, 100 / 1200)
        self.assertEqual(got.lines[0].center.x, 351)
        self.assertTrue(got.lines[0].is_low_confidence)
        self.assertFalse(got.lines[2].is_low_confidence)
        self.assertEqual(got.full_text, "บริษัท ABC 01055581234S6\nInvoice\nยอดรวม 1,070.00")
        self.assertEqual(build_ai_payload(got)["lines"][0]["bbox"], [100, 48, 602, 96])
        self.assertIn("text: บริษัท ABC 01055581234S6", build_ai_text(got))
        stored = normalize([page])[0]
        self.assertEqual(stored["lines"][0]["pixel_polygon"], page["lines"][2]["polygon"])
        self.assertLessEqual(max(x for line in stored["lines"] for x, _ in line["polygon"]), 1)
        self.assertEqual(stored["text"], got.full_text)

    def test_empty_page_and_invalid_geometry(self):
        page = {"width": 100, "height": 100, "duration_ms": 0, "lines": []}
        got = normalize_pages([page])[0]
        self.assertEqual((got.full_text, got.average_confidence, got.line_count), ("", 0, 0))
        page["lines"] = [{"text": "bad", "confidence": 0.5, "polygon": [[0, 0]] * 4}]
        with self.assertRaises(ValueError):
            normalize_pages([page])


if __name__ == "__main__":
    unittest.main()
