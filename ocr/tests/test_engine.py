import tempfile
import time
import unittest
from pathlib import Path
from unittest.mock import Mock, patch

from PIL import Image

from plaiflow_ocr.engine import Engine, _serve


class EngineFailureTest(unittest.TestCase):
    def test_invalid_files_are_terminal_but_model_failures_are_retryable(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "original"
            for invalid, failure in ((True, None), (False, RuntimeError), (False, ValueError)):
                with self.subTest(invalid=invalid, failure=failure):
                    if invalid:
                        path.write_bytes(b"not an image")
                    else:
                        Image.new("RGB", (10, 10), "white").save(path, format="PNG")
                    connection = Mock()
                    connection.recv.side_effect = [str(path), None]
                    model = Mock()
                    model.predict.side_effect = failure("private model details") if failure else None
                    with patch("plaiflow_ocr.engine.create_model", return_value=model):
                        _serve(connection)
                    messages = [call.args[0] for call in connection.send.call_args_list]
                    error = messages[-1]
                    self.assertEqual(error["error"], "invalid_input" if invalid else "temporary_upstream")
                    self.assertNotIn("private model details", str(messages))
                    engine = Engine()
                    with (
                        patch.object(engine, "start"),
                        patch.object(engine, "close") as close,
                        patch.object(engine, "_receive", side_effect=messages[1:]),
                    ):
                        engine.connection = Mock()
                        with self.assertRaises(ValueError if invalid else RuntimeError):
                            engine(path, time.monotonic() + 60)
                        close.assert_called_once()


if __name__ == "__main__":
    unittest.main()
