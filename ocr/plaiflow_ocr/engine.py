"""One persistent, killable inference subprocess; only paths cross the IPC boundary."""

import multiprocessing
import os
import time
from pathlib import Path


def create_model(download=False):
    directories = {}
    if not download:
        from .models import verify, NAMES
        verify()
        root = Path(os.environ["PADDLE_PDX_CACHE_HOME"]) / "official_models"
        for option,name in zip(("doc_orientation_classify_model_dir", "textline_orientation_model_dir", "text_detection_model_dir", "text_recognition_model_dir"), NAMES, strict=True):
            directories[option] = str(root / name)
    from paddleocr import PaddleOCR

    return PaddleOCR(
        text_detection_model_name="PP-OCRv5_mobile_det",
        text_recognition_model_name="th_PP-OCRv5_mobile_rec",
        doc_orientation_classify_model_name="PP-LCNet_x1_0_doc_ori",
        textline_orientation_model_name="PP-LCNet_x1_0_textline_ori",
        use_doc_orientation_classify=True,
        use_doc_unwarping=False,
        use_textline_orientation=True,
        text_rec_score_thresh=0.0,
        text_recognition_batch_size=1,
        device="cpu",
        cpu_threads=2,
        enable_mkldnn=False,
        **directories,
    )


def _serve(connection):
    # Keep native crashes/timeouts isolated; the parent enforces process-tree RSS.
    from PIL import Image, ImageOps
    import numpy as np
    import pypdfium2 as pdfium
    import cv2

    Image.MAX_IMAGE_PIXELS = 25_000_000
    cv2.setNumThreads(2)
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
                signature = source.read(8)
            if signature.startswith(b"%PDF-"):
                document = pdfium.PdfDocument(str(path))
                # Even an empty user password does not make an encrypted PDF acceptable.
                if pdfium.raw.FPDF_GetSecurityHandlerRevision(document.raw) != -1:
                    raise ValueError("invalid_input")
                count = len(document)
            elif signature.startswith(b"\x89PNG\r\n\x1a\n") or signature.startswith(
                b"\xff\xd8\xff"
            ):
                count = 1
            else:
                raise ValueError("invalid_input")
            if not 1 <= count <= 20:
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
                # Decoding is complete; processing failures must remain retryable.
                failure_code = "temporary_upstream"
                image.thumbnail((3500, 3500))
                array = np.array(image.convert("RGB"))[:, :, ::-1].copy()
                image.close()
                # Mild skew correction only; retain this corrected image's coordinate frame.
                gray = cv2.cvtColor(array, cv2.COLOR_BGR2GRAY)
                edges = cv2.Canny(gray, 50, 150)
                segments = cv2.HoughLinesP(
                    edges, 1, np.pi / 180, 80, minLineLength=100, maxLineGap=10
                )
                if segments is not None:
                    angles = [
                        np.degrees(np.arctan2(y2 - y1, x2 - x1))
                        for x1, y1, x2, y2 in segments[:, 0]
                    ]
                    small = [angle for angle in angles if abs(angle) <= 5]
                    if len(small) >= 5:
                        angle = float(np.median(small))
                        h, w = array.shape[:2]
                        array = cv2.warpAffine(
                            array,
                            cv2.getRotationMatrix2D((w / 2, h / 2), angle, 1),
                            (w, h),
                            borderValue=(255, 255, 255),
                        )
                output = next(iter(model.predict(array)))
                corrected = output["doc_preprocessor_res"]["output_img"]
                height, width = corrected.shape[:2]
                angle = int(output["doc_preprocessor_res"].get("angle", 0))
                lines = [
                    {
                        "text": str(text),
                        "confidence": float(score),
                        "polygon": polygon.tolist(),
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
                            "width": width,
                            "height": height,
                            "rotation": angle % 360,
                            "duration_ms": round((time.monotonic() - started) * 1000),
                            "lines": lines,
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
    def __init__(self):
        self.process = None
        self.connection = None
        self.peak_rss = 0

    def close(self):
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

    def _receive(self, deadline):
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

    def start(self):
        if self.process is None:
            context = multiprocessing.get_context("spawn")
            self.connection, child = context.Pipe()
            self.process = context.Process(target=_serve, args=(child,), daemon=True)
            self.process.start()
            child.close()
            try:
                if not self._receive(time.monotonic() + 180).get("ready"):
                    raise RuntimeError("model_initialization_failed")
            except BaseException:
                self.close()
                raise

    def __call__(self, path, deadline):
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
                raise ValueError("invalid_input")
            return pages
        except BaseException:
            self.close()
            raise
