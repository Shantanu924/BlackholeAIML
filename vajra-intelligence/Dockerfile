# syntax=docker/dockerfile:1
FROM python:3.12-slim

LABEL maintainer="BHIV vNEXT — VAJRA Intelligence Lane"
LABEL description="VAJRA Intelligence Layer — AI/ML advisory layer for the VAJRA blockchain runtime."

WORKDIR /app

# Install deps first (layer-cached)
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY . .

# Create data directories
RUN mkdir -p data/evidence data/runtime_logs test_results

# Non-root user for security
RUN useradd -m -u 1001 vajra && chown -R vajra:vajra /app
USER vajra

EXPOSE 8100

# Healthcheck
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8100/intelligence/health')" || exit 1

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8100", "--workers", "1"]
