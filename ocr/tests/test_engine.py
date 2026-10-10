import tempfile
import time
import unittest
from pathlib import Path
from unittest.mock import Mock, patch

import numpy as np
from PIL import Image

from plaiflow_ocr.engine import Engine, _serve


class EngineFailureTest(unittest.TestCase):
    def test_upright_text_is_not_rotated_and_low_quality_gets_one_orientation_retry(self):
        for first_score, retry_score, expected_calls in ((0.95, 0.99, 1), (0.4, 0.9, 2), (0.6, 0.4, 2)):
            with self.subTest(first_score=first_score), tempfile.TemporaryDirectory() as root:
                path = Path(root) / 'upright.png'
                Image.new('RGB', (100, 100), 'white').save(path)
                connection = Mock()
                connection.recv.side_effect = [str(path), None]
                model = Mock()

                def predict(array, use_textline_orientation=True):
                    score = retry_score if use_textline_orientation else first_score
                    return iter([{'doc_preprocessor_res': {'output_img': array, 'angle': 0},
                                  'rec_texts': ['rotated' if use_textline_orientation else 'upright'],
                                  'rec_scores': [score], 'rec_polys': [np.array([[0, 0], [90, 0], [90, 10], [0, 10]])]}])

                model.predict.side_effect = predict
                with patch('plaiflow_ocr.engine.create_model', return_value=model):
                    _serve(connection)
                self.assertFalse(model.predict.call_args_list[0].kwargs.get('use_textline_orientation', True))
                self.assertEqual(model.predict.call_count, expected_calls)
                page = next(call.args[0]['page'] for call in connection.send.call_args_list if 'page' in call.args[0])
                expected = 'rotated' if expected_calls == 2 and retry_score > first_score else 'upright'
                self.assertEqual(page['lines'][0]['text'], expected)

    def test_resize_maps_polygons_back_and_pdf_renders_each_page(self):
        def predict(array, **options):
            height, width = array.shape[:2]
            return iter([{
                "doc_preprocessor_res": {"output_img": array, "angle": 0},
                "rec_texts": ["ไทย Invoice"], "rec_scores": [0.9],
                "rec_polys": [np.array([[0, 0], [width / 2, 0], [width / 2, height / 2], [0, height / 2]])],
            }])

        with tempfile.TemporaryDirectory() as directory:
            image = Image.new("RGB", (4000, 100), "white")
            paths = [Path(directory) / "large.png", Path(directory) / "two.pdf"]
            image.save(paths[0])
            small = Image.new("RGB", (100, 100), "white")
            small.save(paths[1], format="PDF", save_all=True, append_images=[small])
            for path, expected_count in ((paths[0], 1), (paths[1], 2)):
                with self.subTest(path=path.name):
                    connection = Mock()
                    connection.recv.side_effect = [str(path), None]
                    model = Mock()
                    model.predict.side_effect = predict
                    with patch("plaiflow_ocr.engine.create_model", return_value=model):
                        _serve(connection)
                    messages = [call.args[0] for call in connection.send.call_args_list]
                    self.assertEqual(messages[1]["count"], expected_count)
                    pages = [message["page"] for message in messages if "page" in message]
                    self.assertEqual([page["page_number"] for page in pages], list(range(1, expected_count + 1)))
                    if expected_count == 1:
                        self.assertEqual(pages[0]["width"], 4000)
                        self.assertEqual(model.predict.call_args.args[0].shape[1], 1600)
                        self.assertAlmostEqual(pages[0]["lines"][0]["polygon"][1][0], 2000)

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
                        self.assertEqual(close.call_count, 0 if invalid else 1)


if __name__ == "__main__":
    unittest.main()
