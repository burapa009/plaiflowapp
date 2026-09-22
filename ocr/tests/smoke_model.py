"""Synthetic runtime smoke; never presented as the 60-page acceptance benchmark."""

import json
import os
from pathlib import Path
import time

from PIL import Image, ImageDraw, ImageFont
from plaiflow_ocr.engine import Engine
from plaiflow_ocr.worker import normalize


def main():
    output = Path(os.environ.get("OCR_SMOKE_DIR", ".scratch/phase-6/smoke"))
    output.mkdir(parents=True, exist_ok=True)
    font = ImageFont.truetype(
        os.environ.get("OCR_SMOKE_FONT", "C:/Windows/Fonts/tahoma.ttf"), 40
    )
    image = Image.new("RGB", (1200, 800), "white")
    draw = ImageDraw.Draw(image)
    draw.text((60, 80), "ใบเสร็จรับเงิน ทดสอบ 12345", font=font, fill="black")
    draw.text((60, 160), "PlaiFlow Invoice 2026 Total 1000", font=font, fill="black")
    image.save(output / "image.png")
    image.save(output / "document.pdf", "PDF", resolution=200)
    engine = Engine()
    start = time.monotonic()
    try:
        engine.start()
        cold = time.monotonic() - start
        measurements = []
        for name in ("image.png", "document.pdf"):
            start = time.monotonic()
            pages = normalize(engine(output / name, time.monotonic() + 900))
            (output / (name + ".json")).write_text(
                json.dumps(pages, ensure_ascii=False, indent=2), encoding="utf-8"
            )
            measurements.append(
                {
                    "file": name,
                    "seconds": time.monotonic() - start,
                    "peak_rss_bytes": engine.peak_rss,
                    "pages": len(pages),
                    "nonempty": bool(pages[0]["text"]),
                }
            )
        report = {
            "type": "synthetic smoke, not production acceptance",
            "platform": os.name,
            "cold_start_seconds": cold,
            "measurements": measurements,
        }
        (output / "report.json").write_text(
            json.dumps(report, indent=2), encoding="utf-8"
        )
        print(json.dumps(report, indent=2))
    finally:
        engine.close()


if __name__ == "__main__":
    main()
