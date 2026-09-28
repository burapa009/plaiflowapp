"""Canonical, document-agnostic OCR geometry and reading order."""

import math
from typing import Any

from pydantic import BaseModel, Field


class Point(BaseModel):
    x: float
    y: float


class BoundingBox(BaseModel):
    x_min: int
    y_min: int
    x_max: int
    y_max: int


class NormalizedBoundingBox(BaseModel):
    x_min: float = Field(ge=0, le=1)
    y_min: float = Field(ge=0, le=1)
    x_max: float = Field(ge=0, le=1)
    y_max: float = Field(ge=0, le=1)


class OCRLine(BaseModel):
    id: str
    page_number: int
    text: str
    confidence: float = Field(ge=0, le=1)
    is_low_confidence: bool
    bbox: BoundingBox
    normalized_bbox: NormalizedBoundingBox
    polygon: list[list[float]]
    center: Point
    width: int
    height: int
    reading_order: int


class OCRPage(BaseModel):
    page: int
    width: int
    height: int
    rotation: int
    duration_ms: int
    average_confidence: float
    low_confidence_count: int
    line_count: int
    full_text: str
    lines: list[OCRLine]


class OCREngineInfo(BaseModel):
    name: str
    version: str


class OCRProcessingInfo(BaseModel):
    pages: int
    duration_ms: int


class OCRDocumentStats(BaseModel):
    average_confidence: float
    low_confidence_count: int


class OCRResponse(BaseModel):
    success: bool
    document_id: str
    engine: OCREngineInfo
    processing: OCRProcessingInfo
    document: OCRDocumentStats
    pages: list[OCRPage]
    debug: dict[str, Any] | None = None


def reading_order(lines: list[OCRLine]) -> list[OCRLine]:
    if not lines:
        return []
    threshold = sum(max(line.height, 1) for line in lines) / len(lines) * 0.5
    rows: list[list[OCRLine]] = []
    for line in sorted(lines, key=lambda item: (item.center.y, item.bbox.x_min)):
        if rows and abs(line.center.y - sum(item.center.y for item in rows[-1]) / len(rows[-1])) <= threshold:
            rows[-1].append(line)
        else:
            rows.append([line])
    ordered = [line for row in rows for line in sorted(row, key=lambda item: item.bbox.x_min)]
    for number, line in enumerate(ordered, 1):
        line.reading_order = number
    return ordered


def normalize_pages(pages: list[dict[str, Any]], min_confidence: float = 0.30) -> list[OCRPage]:
    if not 0 <= min_confidence <= 1 or not pages:
        raise ValueError("invalid_input")
    normalized: list[OCRPage] = []
    for page_number, source in enumerate(pages, 1):
        width, height = int(source["width"]), int(source["height"])
        if width <= 0 or height <= 0 or width * height > 25_000_000 or len(source["lines"]) > 5000:
            raise ValueError("invalid_input")
        lines: list[OCRLine] = []
        for original in source["lines"]:
            text, confidence = original["text"], float(original["confidence"])
            polygon = original["polygon"]
            if (
                not isinstance(text, str)
                or len(text.encode("utf-8")) > 16384
                or not math.isfinite(confidence)
                or not 0 <= confidence <= 1
                or len(polygon) != 4
                or any(len(point) != 2 or not all(math.isfinite(value) for value in point) for point in polygon)
            ):
                raise ValueError("invalid_input")
            polygon = [[float(x), float(y)] for x, y in polygon]
            x_min = max(0, math.floor(min(point[0] for point in polygon)))
            y_min = max(0, math.floor(min(point[1] for point in polygon)))
            x_max = min(width, math.ceil(max(point[0] for point in polygon)))
            y_max = min(height, math.ceil(max(point[1] for point in polygon)))
            if x_min >= x_max or y_min >= y_max:
                raise ValueError("invalid_input")
            box = BoundingBox(x_min=x_min, y_min=y_min, x_max=x_max, y_max=y_max)
            lines.append(OCRLine(
                id=f"p{page_number}_line_0000", page_number=page_number, text=text, confidence=confidence,
                is_low_confidence=confidence < min_confidence, bbox=box,
                normalized_bbox=NormalizedBoundingBox(
                    x_min=x_min / width, y_min=y_min / height,
                    x_max=x_max / width, y_max=y_max / height,
                ),
                polygon=polygon,
                center=Point(x=(x_min + x_max) / 2, y=(y_min + y_max) / 2),
                width=x_max - x_min, height=y_max - y_min, reading_order=0,
            ))
        ordered = reading_order(lines)
        for number, line in enumerate(ordered, 1):
            line.id = f"p{page_number}_line_{number:04d}"
        normalized.append(OCRPage(
            page=page_number, width=width, height=height, rotation=int(source.get("rotation", 0)),
            duration_ms=int(source.get("duration_ms", 0)),
            average_confidence=sum(line.confidence for line in ordered) / len(ordered) if ordered else 0,
            low_confidence_count=sum(line.is_low_confidence for line in ordered),
            line_count=len(ordered), full_text="\n".join(line.text for line in ordered), lines=ordered,
        ))
    return normalized


def build_ai_payload(page: OCRPage) -> dict[str, Any]:
    return {
        "page": page.page, "width": page.width, "height": page.height,
        "lines": [{
            "text": line.text, "confidence": line.confidence,
            "bbox": [line.bbox.x_min, line.bbox.y_min, line.bbox.x_max, line.bbox.y_max],
        } for line in page.lines],
    }


def build_ai_text(page: OCRPage) -> str:
    chunks = [f"PAGE_SIZE: {page.width}x{page.height}"]
    for line in page.lines:
        box = line.bbox
        chunks.append(
            f"[{line.reading_order:03d}]\ntext: {line.text}\nconfidence: {line.confidence:.3f}"
            f"\nbbox: [{box.x_min},{box.y_min},{box.x_max},{box.y_max}]"
        )
    return "\n\n".join(chunks)
