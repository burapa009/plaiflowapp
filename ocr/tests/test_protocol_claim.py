import json
import unittest
from unittest.mock import MagicMock, patch

from plaiflow_ocr.protocol import Protocol


class TargetedClaimTests(unittest.TestCase):
    def test_target_is_sent_without_changing_provider(self):
        protocol = Protocol.__new__(Protocol)
        response = MagicMock()
        response.__enter__.return_value.read.return_value = b'{"jobs": []}'
        protocol.request = MagicMock(return_value=response)
        with patch.dict("os.environ", {"OCR_PROVIDER": "runpod"}):
            self.assertEqual(protocol.claim("ca18ce81-4b0d-4428-83ba-516c06ea528e"), [])
        self.assertEqual(json.loads(protocol.request.call_args.kwargs["data"]), {
            "provider": "runpod", "job_id": "ca18ce81-4b0d-4428-83ba-516c06ea528e",
        })

    def test_normal_claim_keeps_existing_contract(self):
        protocol = Protocol.__new__(Protocol)
        response = MagicMock()
        response.__enter__.return_value.read.return_value = b'{"jobs": []}'
        protocol.request = MagicMock(return_value=response)
        with patch.dict("os.environ", {"OCR_PROVIDER": "railway"}):
            protocol.claim()
        self.assertEqual(json.loads(protocol.request.call_args.kwargs["data"]), {"provider": "railway"})
