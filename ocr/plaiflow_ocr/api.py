"""Authenticated OCR HTTP boundary; inference stays in the existing killable process."""

import asyncio
import hmac
import json
import logging
import os
import tempfile
import time
import uuid
from contextlib import asynccontextmanager
from importlib.metadata import version
from pathlib import Path
from typing import AsyncIterator, Awaitable, Callable

import pypdfium2 as pdfium
from fastapi import FastAPI, File, HTTPException, Request, Response, UploadFile
from fastapi.concurrency import run_in_threadpool
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse

from .engine import Engine
from .normalize import (
    OCRDocumentStats, OCREngineInfo, OCRProcessingInfo, OCRResponse, normalize_pages,
)
from .worker import MAX_RESULT

logger = logging.getLogger("plaiflow_ocr")
logger.setLevel(os.environ.get("OCR_LOG_LEVEL", "INFO").upper())
if not logger.handlers:
    logger.addHandler(logging.StreamHandler())


def _limit(name: str, default: int) -> int:
    value = int(os.environ.get(name, str(default)))
    if value <= 0:
        raise ValueError(f"invalid_{name}")
    return value


def _file_type(signature: bytes) -> str:
    if signature.startswith(b"%PDF-"):
        return "application/pdf"
    if signature.startswith(b"\x89PNG\r\n\x1a\n"):
        return "image/png"
    if signature.startswith(b"\xff\xd8\xff"):
        return "image/jpeg"
    if signature.startswith(b"RIFF") and signature[8:12] == b"WEBP":
        return "image/webp"
    raise HTTPException(400, "UNSUPPORTED_FILE_TYPE")


def _validate_pdf(path: Path, maximum: int) -> None:
    document = None
    try:
        document = pdfium.PdfDocument(str(path))
        if pdfium.raw.FPDF_GetSecurityHandlerRevision(document.raw) != -1:
            raise HTTPException(400, "INVALID_FILE")
        if not 1 <= len(document) <= maximum:
            raise HTTPException(400, "PDF_PAGE_LIMIT_EXCEEDED")
    except HTTPException:
        raise
    except Exception as error:
        raise HTTPException(400, "INVALID_FILE") from error
    finally:
        if document is not None:
            document.close()


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    token = os.environ.get("OCR_SERVICE_TOKEN", "")
    if len(token) < 32:
        raise RuntimeError("OCR_SERVICE_TOKEN must contain at least 32 characters")
    app.state.token = token
    app.state.max_file_size = _limit("OCR_MAX_FILE_SIZE_MB", 20) * 1024 * 1024
    app.state.max_pdf_pages = _limit("OCR_MAX_PDF_PAGES", 50)
    app.state.min_confidence = float(os.environ.get("OCR_MIN_CONFIDENCE", "0.30"))
    if not 0 <= app.state.min_confidence <= 1:
        raise RuntimeError("invalid OCR_MIN_CONFIDENCE")
    app.state.lock = asyncio.Lock()
    app.state.engine = Engine(max_pages=app.state.max_pdf_pages)
    await run_in_threadpool(app.state.engine.start)
    try:
        yield
    finally:
        await run_in_threadpool(app.state.engine.close)


app = FastAPI(lifespan=lifespan, docs_url=None, redoc_url=None, openapi_url=None)


def _error(status: int, code: str, request_id: str) -> JSONResponse:
    messages = {
        "INVALID_FILE": "Invalid document",
        "UNSUPPORTED_FILE_TYPE": "Unsupported file type",
        "FILE_TOO_LARGE": "File exceeds upload limit",
        "PDF_PAGE_LIMIT_EXCEEDED": "PDF exceeds page limit",
        "UNAUTHORIZED": "Access denied",
        "OCR_BUSY": "OCR service is busy",
        "OCR_PROCESSING_FAILED": "Unable to process document",
        "INTERNAL_ERROR": "Internal error",
    }
    return JSONResponse(
        status_code=status,
        content={"success": False, "error": {"code": code, "message": messages[code]}, "request_id": request_id},
    )


@app.middleware("http")
async def authorize_upload(request: Request, call_next: Callable[[Request], Awaitable[Response]]) -> Response:
    if request.url.path == "/v1/ocr":
        request.state.request_id = str(uuid.uuid4())
        supplied = request.headers.get("authorization", "").removeprefix("Bearer ")
        if not hmac.compare_digest(supplied, request.app.state.token):
            return _error(401, "UNAUTHORIZED", request.state.request_id)
        try:
            declared = int(request.headers["content-length"])
        except ValueError:
            return _error(400, "INVALID_FILE", request.state.request_id)
        if declared > request.app.state.max_file_size + (1 << 20):
            return _error(413, "FILE_TOO_LARGE", request.state.request_id)
    return await call_next(request)


@app.exception_handler(HTTPException)
async def http_error(request: Request, error: HTTPException) -> JSONResponse:
    code = str(error.detail)
    return _error(error.status_code, code if code in {
        "INVALID_FILE", "UNSUPPORTED_FILE_TYPE", "FILE_TOO_LARGE", "PDF_PAGE_LIMIT_EXCEEDED",
        "UNAUTHORIZED", "OCR_BUSY", "OCR_PROCESSING_FAILED",
    } else "INTERNAL_ERROR", request.state.request_id if hasattr(request.state, "request_id") else str(uuid.uuid4()))


@app.exception_handler(RequestValidationError)
async def validation_error(request: Request, error: RequestValidationError) -> JSONResponse:
    return _error(400, "INVALID_FILE", request.state.request_id if hasattr(request.state, "request_id") else str(uuid.uuid4()))


@app.exception_handler(Exception)
async def internal_error(request: Request, error: Exception) -> JSONResponse:
    logger.error(json.dumps({"event": "ocr.internal_error", "request_id": getattr(request.state, "request_id", ""), "error_type": type(error).__name__}))
    return _error(500, "INTERNAL_ERROR", getattr(request.state, "request_id", str(uuid.uuid4())))


@app.get("/health")
async def health() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/v1/ocr", response_model=OCRResponse, response_model_exclude_none=True)
async def ocr(request: Request, file: UploadFile = File(...)) -> OCRResponse:
    request_id = request.state.request_id
    document_id = str(uuid.uuid4())
    maximum = request.app.state.max_file_size
    page_limit = request.app.state.max_pdf_pages
    started = time.monotonic()
    size = 0
    mime = ""
    try:
        with tempfile.TemporaryDirectory(prefix="ocr-api-") as directory:
            path = Path(directory) / "original"
            with path.open("xb") as output:
                while chunk := await file.read(65536):
                    size += len(chunk)
                    if size > maximum:
                        raise HTTPException(413, "FILE_TOO_LARGE")
                    output.write(chunk)
            with path.open("rb") as source:
                mime = _file_type(source.read(12))
            if mime != file.content_type:
                raise HTTPException(400, "INVALID_FILE")
            if mime == "application/pdf":
                await run_in_threadpool(_validate_pdf, path, page_limit)
            try:
                await asyncio.wait_for(request.app.state.lock.acquire(), timeout=30)
            except TimeoutError as error:
                raise HTTPException(503, "OCR_BUSY") from error
            try:
                try:
                    raw_pages = await run_in_threadpool(request.app.state.engine, path, time.monotonic() + 900)
                except ValueError as error:
                    raise HTTPException(400, "INVALID_FILE") from error
                except Exception as error:
                    logger.error(json.dumps({"event": "ocr.inference_failed", "request_id": request_id, "error_type": type(error).__name__}))
                    raise HTTPException(500, "OCR_PROCESSING_FAILED") from error
            finally:
                request.app.state.lock.release()
    finally:
        await file.close()
    pages = normalize_pages(raw_pages, request.app.state.min_confidence)
    scores = [line.confidence for page in pages for line in page.lines]
    duration_ms = round((time.monotonic() - started) * 1000)
    response = OCRResponse(
        success=True, document_id=document_id,
        engine=OCREngineInfo(name="paddleocr", version=version("paddleocr")),
        processing=OCRProcessingInfo(pages=len(pages), duration_ms=duration_ms),
        document=OCRDocumentStats(
            average_confidence=sum(scores) / len(scores) if scores else 0,
            low_confidence_count=sum(page.low_confidence_count for page in pages),
        ),
        pages=pages,
        debug={"raw_result": [page["raw_result"] for page in raw_pages]}
        if os.environ.get("OCR_INCLUDE_RAW_RESULT", "false").lower() == "true" else None,
    )
    if len(response.model_dump_json(exclude_none=True).encode("utf-8")) > MAX_RESULT:
        raise HTTPException(500, "OCR_PROCESSING_FAILED")
    logger.info(json.dumps({
        "request_id": request_id, "document_id": document_id, "file_type": mime,
        "file_size": size, "page_count": len(pages), "processing_ms": duration_ms,
        "average_confidence": response.document.average_confidence,
        "low_confidence_count": response.document.low_confidence_count,
    }))
    return response


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host=os.environ.get("OCR_BIND_HOST", "0.0.0.0"), port=int(os.environ.get("PORT", "8000")), workers=1)
