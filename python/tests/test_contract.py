import asyncio
import json
import unittest
from pathlib import Path

from plaiflow_export import submit


class ContractTest(unittest.TestCase):
    def test_shared_request_and_result(self):
        root = Path(__file__).parents[2]
        request = json.loads((root / "contracts/export/v1/request.json").read_text())
        expected = json.loads((root / "contracts/export/v1/result.json").read_text())
        result = asyncio.run(submit(request))
        self.assertEqual(result["version"], expected["version"])
        self.assertTrue(result["job_ref"])


if __name__ == "__main__":
    unittest.main()
