import unittest

from plaiflow_ocr.generative import FIELDS, normalize_pages, validate_fields


class GenerativeTest(unittest.TestCase):
    def test_evidence_and_transcription_boundary(self):
        fields = dict.fromkeys(FIELDS)
        lines = ["Total 11,875.00", "<figure>", "Tax ID 1234567890123", "</figure>"]
        fields["total_amount"] = {"raw": "11,875.00", "line": 1}
        fields["seller_tax_id"] = {"raw": "1234567890123", "line": 3}
        fields["seller_name"] = {"raw": "invented", "line": 1}
        proposals = validate_fields(fields, lines)
        self.assertEqual(proposals, [{"field": "total_amount", "raw": "11,875.00", "line": 1}])
        page = dict(page_number=1, width=100, height=100, rotation=0, duration_ms=15,
                    transcription_ms=10, extraction_ms=5, text="\n".join(lines), lines=[], proposals=proposals)
        self.assertEqual(normalize_pages([page]), [page])
        for bad in ({"proposals": proposals * 2}, {"duration_ms": 600001}, {"lines": [{}]},
                    {"proposals": [{"field": "total_amount", "raw": "9999", "line": 1}]}):
            with self.assertRaises(ValueError):
                normalize_pages([{**page, **bad}])


if __name__ == "__main__":
    unittest.main()
