"""One persistent, killable inference subprocess; only paths cross the IPC boundary."""

import multiprocessing
import os
import time
from multiprocessing.connection import Connection
from pathlib import Path
from typing import Any


def create_model(download: bool = False) -> Any:
    directories = {}
    if not download:
        from .models import verify, NAMES
        verify()
        root = Path(os.environ["PADDLE_PDX_CACHE_HOME"]) / "official_models"
        for option,name in zip(("textline_orientation_model_dir", "text_detection_model_dir", "text_recognition_model_dir"), NAMES, strict=True):
            directories[option] = str(root / name)
    from paddleocr import PaddleOCR

    return PaddleOCR(
        text_detection_model_name="PP-OCRv5_mobile_det",
        text_recognition_model_name="th_PP-OCRv5_mobile_rec",
        textline_orientation_model_name="PP-LCNet_x1_0_textline_ori",
        use_doc_orientation_classify=False,
        use_doc_unwarping=False,
        use_textline_orientation=True,
        text_rec_score_thresh=0.0,
        text_recognition_batch_size=1,
        device=os.environ.get("OCR_DEVICE", "cpu"),
        cpu_threads=2,
        enable_mkldnn=False,
        **directories,
    )


def _serve(connection: Connection, max_pages: int = 20) -> None:
    # Keep native crashes/timeouts isolated; the parent enforces process-tree RSS.
    from PIL import Image, ImageOps
    import numpy as np
    import pypdfium2 as pdfium

    Image.MAX_IMAGE_PIXELS = 25_000_000
    model = create_model()
    connection.send({"ready": True})
    while True:
        command = connection.recv()
        if command is None:
            return
        path = Path(command)
        document = None
        failure_code = "invalid_input"
        try:
            with path.open("rb") as source:
                signature = source.read(12)
            if signature.startswith(b"%PDF-"):
                document = pdfium.PdfDocument(str(path))
                # Even an empty user password does not make an encrypted PDF acceptable.
                if pdfium.raw.FPDF_GetSecurityHandlerRevision(document.raw) != -1:
                    raise ValueError("invalid_input")
                count = len(document)
            elif (
                signature.startswith(b"\x89PNG\r\n\x1a\n")
                or signature.startswith(b"\xff\xd8\xff")
                or (signature.startswith(b"RIFF") and signature[8:12] == b"WEBP")
            ):
                count = 1
            else:
                raise ValueError("invalid_input")
            if not 1 <= count <= max_pages:
                raise ValueError("invalid_input")
            connection.send({"count": count})
            for number in range(count):
                failure_code = "invalid_input"
                started = time.monotonic()
                if document is not None:
                    pdfpage = document[number]
                    try:
                        width, height = pdfpage.get_size()
                        if (
                            width <= 0
                            or height <= 0
                            or width * height * (200 / 72) ** 2 > 25_000_000
                        ):
                            raise ValueError("invalid_input")
                        bitmap = pdfpage.render(scale=200 / 72)
                        image = bitmap.to_pil().copy()
                        bitmap.close()
                    finally:
                        pdfpage.close()
                else:
                    with Image.open(path) as original:
                        if (
                            original.width * original.height > 25_000_000
                            or getattr(original, "n_frames", 1) != 1
                        ):
                            raise ValueError("invalid_input")
                        image = ImageOps.exif_transpose(original).convert("RGB")
                # Coordinates refer to the EXIF-corrected image or rendered PDF page.
                failure_code = "temporary_upstream"
                original_width, original_height = image.size
                # Keep Paddle's temporary tensors within the Trial container's 1 GB limit.
                image.thumbnail((1600, 1600))
                array = np.array(image.convert("RGB"))[:, :, ::-1].copy()
                image.close()
                # The line-orientation model can flip small upright Thai text into gibberish.
                output = next(iter(model.predict(array, use_textline_orientation=False)))
                def quality(result):
                    scores = result["rec_scores"]
                    return sum(float(score) for score in scores) / max(1, len(scores))

                if len(output["rec_scores"]) and quality(output) < 0.8:
                    rotated = next(iter(model.predict(array, use_textline_orientation=True)))
                    if quality(rotated) > quality(output):
                        output = rotated
                corrected = output["doc_preprocessor_res"]["output_img"]
                if corrected.shape[:2] != array.shape[:2] or output["doc_preprocessor_res"].get("angle", 0):
                    raise RuntimeError("unexpected_coordinate_transform")
                scale_x = original_width / array.shape[1]
                scale_y = original_height / array.shape[0]
                lines = [
                    {
                        "text": str(text),
                        "confidence": float(score),
                        "polygon": [[float(x * scale_x), float(y * scale_y)] for x, y in polygon],
                    }
                    for text, score, polygon in zip(
                        output["rec_texts"],
                        output["rec_scores"],
                        output["rec_polys"],
                        strict=True,
                    )
                ]
                if (
                    len(lines) > 5000
                    or sum(len(line["text"].encode()) for line in lines) > 1_000_000
                ):
                    raise ValueError("invalid_input")
                connection.send(
                    {
                        "page": {
                            "page_number": number + 1,
                            "width": original_width,
                            "height": original_height,
                            "rotation": 0,
                            "duration_ms": round((time.monotonic() - started) * 1000),
                            "lines": lines,
                            **({"raw_result": {
                                "rec_texts": [str(text) for text in output["rec_texts"]],
                                "rec_scores": [float(score) for score in output["rec_scores"]],
                                "rec_boxes": output["rec_boxes"].tolist(),
                                "rec_polys": [poly.tolist() for poly in output["rec_polys"]],
                            }} if os.environ.get("OCR_INCLUDE_RAW_RESULT", "false").lower() == "true" else {}),
                        }
                    }
                )
            connection.send({"done": True})
        except Exception as error:
            # No decoder exceptions or document contents cross into logs.
            import traceback

            frames = traceback.extract_tb(error.__traceback__)
            connection.send(
                {
                    "error": failure_code,
                    "exception_type": type(error).__name__,
                    "frame": frames[-1].name + ":" + str(frames[-1].lineno),
                }
            )
        finally:
            if document is not None:
                document.close()


class Engine:
    def __init__(self, max_pages: int = 20) -> None:
        self.max_pages = max_pages
        self.process = None
        self.connection = None
        self.peak_rss = 0

    def close(self) -> None:
        if self.process is not None:
            self.process.terminate()
            self.process.join(timeout=5)
            if self.process.is_alive():
                self.process.kill()
                self.process.join(timeout=5)
            self.process = None
        if self.connection is not None:
            self.connection.close()
            self.connection = None

    def _receive(self, deadline: float) -> dict[str, Any]:
        import psutil

        while time.monotonic() < deadline:
            rss = psutil.Process(os.getpid()).memory_info().rss
            if self.process is None or not self.process.is_alive():
                raise RuntimeError("process_crash")
            child = psutil.Process(self.process.pid)
            rss += child.memory_info().rss + sum(
                p.memory_info().rss for p in child.children(recursive=True)
            )
            self.peak_rss = max(self.peak_rss, rss)
            if rss > 4 * 1024**3:
                raise ValueError("resource_limit")
            if self.connection.poll(0.1):
                return self.connection.recv()
        raise TimeoutError("ocr_timeout")

    def start(self) -> None:
        if self.process is None:
            context = multiprocessing.get_context("spawn")
            self.connection, child = context.Pipe()
            self.process = context.Process(target=_serve, args=(child, self.max_pages), daemon=True)
            self.process.start()
            child.close()
            try:
                if not self._receive(time.monotonic() + 180).get("ready"):
                    raise RuntimeError("model_initialization_failed")
            except BaseException:
                self.close()
                raise

    def __call__(self, path: Path, deadline: float) -> list[dict[str, Any]]:
        self.start()
        self.peak_rss = 0
        pages = []
        self.connection.send(str(path))
        try:
            header = self._receive(min(deadline, time.monotonic() + 45))
            if "error" in header:
                error_type = ValueError if header["error"] == "invalid_input" else RuntimeError
                raise error_type(header["error"])
            for _ in range(header["count"]):
                message = self._receive(min(deadline, time.monotonic() + 45))
                if "error" in message:
                    error_type = ValueError if message["error"] == "invalid_input" else RuntimeError
                    raise error_type(
                        message["error"]
                        + ":"
                        + message.get("exception_type", "")
                        + ":"
                        + message.get("frame", "")
                    )
                pages.append(message["page"])
            if not self._receive(min(deadline, time.monotonic() + 45)).get("done"):
                raise RuntimeError("invalid_protocol_response")
            return pages
        except ValueError as error:
            if not str(error).startswith("invalid_input"):
                self.close()
            raise
        except BaseException:
            self.close()
            raise
