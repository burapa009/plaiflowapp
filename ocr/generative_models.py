"""Bake the publisher's pinned model artifacts into the worker image."""
import hashlib
import json
import os
import sys
import urllib.request
from pathlib import Path

from plaiflow_ocr.generative import OPENTHAI_MODEL, OPENTHAI_WEIGHTS, TYPHOON_REVISION

LAYERS = [
    ("application/vnd.ollama.image.projector", "7eec6ab6043a1fbc5ab54fea0fe7db53f704b5e0b62c6974404f2e8bd50160de", 931145760),
    ("application/vnd.ollama.image.model", "8a53bc6612576dffeb1f988bb5eede4321b004bf23f28a6f0e6173b4a0780f52", 16810714240),
    ("application/vnd.ollama.image.template", "ae370d884f108d16e7cc8fd5259ebc5773a0afa6e078b11f4ed7e39a27e0dfc4", 1723),
    ("application/vnd.ollama.image.params", "1d5b971ca7607cae0fa648c8bda410938c3420f767170e07d3557efd879cc441", 81),
]
CONFIG = ("application/vnd.docker.container.image.v1+json", "592e46f09edad2305e6601ca240ed7e3c814d392c1f22c032cb284f624471ced", 490)


def assemble_weights(root=Path("/opt")):
    destination = root / "ollama" / "blobs" / ("sha256-" + OPENTHAI_WEIGHTS)
    if destination.exists():
        return
    temporary = destination.with_suffix(".assembling")
    digest = hashlib.sha256()
    try:
        with temporary.open("wb") as output:
            for suffix in ("aa", "ab", "ac"):
                with (root / ("openthai-part-" + suffix)).open("rb") as source:
                    while chunk := source.read(8 << 20):
                        digest.update(chunk)
                        output.write(chunk)
        if digest.hexdigest() != OPENTHAI_WEIGHTS:
            raise ValueError("model_hash_mismatch")
        temporary.replace(destination)
    finally:
        temporary.unlink(missing_ok=True)


def models(download=False):
    root = Path(os.environ.get("OLLAMA_MODELS", "/opt/ollama"))
    model, tag = OPENTHAI_MODEL.split(":")
    for _, digest, size in [CONFIG, *LAYERS]:
        path = root / "blobs" / ("sha256-" + digest)
        if download:
            path.parent.mkdir(parents=True, exist_ok=True)
            with urllib.request.urlopen(f"https://registry.ollama.ai/v2/{model}/blobs/sha256:{digest}", timeout=300) as source, path.open("wb") as output:
                while chunk := source.read(8 << 20):
                    output.write(chunk)
        paths = [path]
        if digest == OPENTHAI_WEIGHTS and not path.exists():
            paths = [Path("/opt/openthai-part-" + suffix) for suffix in ("aa", "ab", "ac")]
        if sum(part.stat().st_size for part in paths) != size:
            raise ValueError("model_size_mismatch")
        actual = hashlib.sha256()
        for part in paths:
            with part.open("rb") as source:
                while chunk := source.read(8 << 20):
                    actual.update(chunk)
        if actual.hexdigest() != digest:
            raise ValueError("model_hash_mismatch")
    entry = lambda item: {"mediaType": item[0], "digest": "sha256:" + item[1], "size": item[2]}
    manifest = {"schemaVersion": 2, "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
                "config": entry(CONFIG), "layers": [entry(item) for item in LAYERS]}
    path = root / "manifests" / "registry.ollama.ai" / model / tag
    if download:
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(manifest), encoding="utf-8")
        from huggingface_hub import snapshot_download
        snapshot_download("typhoon-ai/typhoon-ocr1.5-2b", revision=TYPHOON_REVISION,
                          local_dir="/opt/typhoon", allow_patterns=["*.json", "*.safetensors", "*.txt", "*.jinja"])
    if json.loads(path.read_text()) != manifest or not list(Path("/opt/typhoon").glob("*.safetensors")):
        raise ValueError("model_manifest_mismatch")


if __name__ == "__main__":
    models("--download" in sys.argv)
