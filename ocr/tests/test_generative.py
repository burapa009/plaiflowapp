import unittest
import hashlib
import tempfile
from pathlib import Path
from unittest.mock import patch

from plaiflow_ocr.generative import FIELDS, TYPHOON_GENERATION, normalize_pages, validate_fields
from generative_models import assemble_weights


class GenerativeTest(unittest.TestCase):
    def test_typhoon_uses_pinned_publisher_generation_config(self):
        self.assertEqual(TYPHOON_GENERATION, {"max_new_tokens": 10000})

    def test_split_weights_are_verified_before_atomic_publication(self):
        data = b"pinned model weights"
        digest = hashlib.sha256(data).hexdigest()
        with tempfile.TemporaryDirectory() as directory, patch("generative_models.OPENTHAI_WEIGHTS", digest):
            root = Path(directory)
            (root / "ollama/blobs").mkdir(parents=True)
            for suffix, chunk in zip(("aa", "ab", "ac"), (data[:5], data[5:10], data[10:])):
                (root / ("openthai-part-" + suffix)).write_bytes(chunk)
            destination = root / "ollama/blobs" / ("sha256-" + digest)
            (root / "openthai-part-ac").write_bytes(b"corrupted")
            with self.assertRaisesRegex(ValueError, "model_hash_mismatch"):
                assemble_weights(root)
            self.assertFalse(destination.exists())
            self.assertFalse(destination.with_suffix(".assembling").exists())
            (root / "openthai-part-ac").write_bytes(data[10:])
            assemble_weights(root)
            self.assertEqual(destination.read_bytes(), data)
            assemble_weights(root)

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
