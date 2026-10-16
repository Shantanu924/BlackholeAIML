"""
main.py — FastAPI application entry point for the VAJRA Intelligence Layer.

Run with:
    uvicorn main:app --reload --port 8100

API docs:  http://localhost:8100/docs
Health:    http://localhost:8100/intelligence/health
Adapters:  http://localhost:8100/intelligence/adapters
"""
from __future__ import annotations

import logging
import sys
from contextlib import asynccontextmanager
from pathlib import Path

import uvicorn
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

# Ensure the package root is on sys.path so sub-packages resolve correctly
sys.path.insert(0, str(Path(__file__).parent))

from api.routes import router
from config import settings
from intelligence.evidence import init_db

logging.basicConfig(
    level=getattr(logging, settings.LOG_LEVEL.upper(), logging.INFO),
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
    datefmt="%Y-%m-%dT%H:%M:%SZ",
)
logger = logging.getLogger("vajra.intelligence.main")


@asynccontextmanager
async def lifespan(app: FastAPI):
    logger.info(
        "VAJRA Intelligence Layer starting (mode=%s).",
        "LIVE" if settings.vajra_live else "SIMULATED",
    )
    init_db()
    logger.info("Advisory evidence DB initialised.")
    yield
    logger.info("VAJRA Intelligence Layer shutting down.")


app = FastAPI(
    title="VAJRA Intelligence Layer",
    description=(
        "AI/ML + Blockchain intelligence layer that attaches to the real VAJRA runtime.\n\n"
        "**AI decisions are ADVISORY only.** Cybersecurity enforcement authority is VAJRA.\n\n"
        "All adapters show **LIVE** or **SIMULATED** status — never fakes live data.\n\n"
        "Set `VAJRA_INTERNAL_URL` in `.env` to flip from SIMULATED → LIVE mode."
    ),
    version=settings.SERVICE_VERSION,
    lifespan=lifespan,
)

# CORS — allow dashboard and external clients
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(router)


@app.get("/", include_in_schema=False)
def root() -> dict:
    return {
        "service": settings.SERVICE_NAME,
        "version": settings.SERVICE_VERSION,
        "docs": "/docs",
        "health": "/intelligence/health",
        "adapters": "/intelligence/adapters",
        "mode": "LIVE" if settings.vajra_live else "SIMULATED",
    }


if __name__ == "__main__":
    uvicorn.run(
        "main:app",
        host=settings.HOST,
        port=settings.PORT,
        reload=True,
    )
