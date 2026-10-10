import tempfile
import unittest
from unittest.mock import Mock, patch
from plaiflow_ocr import __main__ as worker


class PollingTest(unittest.TestCase):
    def test_idle_railway_worker_accepts_new_document_within_two_seconds(self):
        stop = Mock()
        stop.is_set.side_effect = [False] * 8 + [True]
        protocol = Mock()
        protocol.claim.return_value = []
        with tempfile.TemporaryDirectory() as root, patch.dict(worker.os.environ, {'OCR_TEMP_DIR': root, 'OCR_PROVIDER': 'railway', 'OCR_SERVICE_URL': ''}), patch.object(worker.threading, 'Event', return_value=stop), patch.object(worker.signal, 'signal'), patch.object(worker, 'Protocol', return_value=protocol), patch.object(worker, 'Engine'), patch.object(worker.random, 'random', return_value=0.5):
            worker.main()
        self.assertLessEqual(max(call.args[0] for call in stop.wait.call_args_list), 2, 'idle backoff delays a new document before inference even starts')
