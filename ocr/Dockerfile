FROM python:3.12-slim
ENV PYTHONUNBUFFERED=1 OMP_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2 \
    PADDLE_PDX_CACHE_HOME=/opt/models PADDLE_PDX_DISABLE_MODEL_SOURCE_CHECK=True \
    OCR_TEMP_DIR=/tmp/plaiflow-ocr
RUN apt-get update && apt-get install -y --no-install-recommends libgomp1 libgl1 libglib2.0-0 && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY ocr/requirements.lock ./requirements.lock
RUN pip install --no-cache-dir -r requirements.lock
COPY ocr/plaiflow_ocr ./plaiflow_ocr
COPY ocr/model-manifest.json ./model-manifest.json
RUN python -m plaiflow_ocr.models --download
RUN useradd --uid 10001 --create-home ocr && mkdir -p /tmp/plaiflow-ocr && chown ocr:ocr /tmp/plaiflow-ocr
USER ocr
CMD ["python", "-m", "plaiflow_ocr"]
