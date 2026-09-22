import argparse
import hashlib
import json
import os
from pathlib import Path

NAMES = (
    "PP-LCNet_x1_0_doc_ori",
    "PP-LCNet_x1_0_textline_ori",
    "PP-OCRv5_mobile_det",
    "th_PP-OCRv5_mobile_rec",
)


def verify():
    root = Path(os.environ["PADDLE_PDX_CACHE_HOME"]) / "official_models"
    manifest = Path(__file__).resolve().parent.parent / "model-manifest.json"
    expected = json.loads(manifest.read_text(encoding="utf-8"))
    for relative, digest in expected.items():
        path = root / relative
        if not path.is_file() or path.is_symlink():
            raise RuntimeError("model_asset_missing")
        with path.open("rb") as source:
            if hashlib.file_digest(source, "sha256").hexdigest() != digest:
                raise RuntimeError("model_asset_checksum_mismatch")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--download", action="store_true")
    args = parser.parse_args()
    if args.download:
        from .engine import create_model

        create_model(download=True)
    verify()


if __name__ == "__main__":
    main()
