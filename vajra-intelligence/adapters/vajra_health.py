"""
adapters/vajra_health.py — VAJRA Health Check Adapter.

Target (internal):  GET {VAJRA_INTERNAL_URL}/health
                    GET {VAJRA_INTERNAL_URL}/api/v1/health

Status: SIMULATED (VAJRA_INTERNAL_URL not yet configured)
        → will flip to LIVE automatically once internal URL is provided.
"""
from __future__ import annotations

import time
from typing import Any

import httpx

from adapters.base import BaseVajraAdapter
from config import settings
from intelligence.models import AdapterStatus


class VajraHealthAdapter(BaseVajraAdapter):
    """
    Checks VAJRA runtime health.

    LIVE:      Calls GET /health on the internal VAJRA API.
    SIMULATED: Returns a clearly-labelled synthetic health response.
    """

    @property
    def name(self) -> str:
        return "vajra-health"

    @property
    def endpoint(self) -> str:
        base = settings.VAJRA_INTERNAL_URL or settings.VAJRA_BASE_URL
        return f"{base}/health"

    def call(self, **kwargs: Any) -> dict[str, Any]:
        if self.status == AdapterStatus.SIMULATED:
            return self._simulated()
        return self._live()

    def _live(self) -> dict[str, Any]:
        """Call the real VAJRA health endpoint."""
        url = f"{settings.VAJRA_INTERNAL_URL.rstrip('/')}/health"
        headers: dict[str, str] = {}
        if settings.VAJRA_API_KEY:
            headers["Authorization"] = f"Bearer {settings.VAJRA_API_KEY}"

        try:
            response = httpx.get(url, headers=headers, timeout=settings.VAJRA_TIMEOUT_SECONDS)
            response.raise_for_status()
            return {
                **response.json(),
                "_adapter": self.name,
                "_status": "LIVE",
                "_called_at": time.time(),
            }
        except httpx.HTTPStatusError as exc:
            return {
                "_adapter": self.name,
                "_status": "ERROR",
                "_error": f"HTTP {exc.response.status_code}: {exc.response.text[:200]}",
                "_called_at": time.time(),
            }
        except Exception as exc:
            return {
                "_adapter": self.name,
                "_status": "ERROR",
                "_error": str(exc)[:200],
                "_called_at": time.time(),
            }

    def _simulated(self) -> dict[str, Any]:
        """Return a clearly-labelled SIMULATED health response."""
        return {
            "status": "HEALTHY",
            "service": "TANTRA Governance & VAJRA Trust Layer",
            "version": "2.0.0",
            "timestamp": time.time(),
            "_adapter": self.name,
            "_status": "SIMULATED",
            "_note": (
                "SIMULATED — set VAJRA_INTERNAL_URL in .env to enable LIVE mode."
            ),
        }


# Module-level singleton
vajra_health_adapter = VajraHealthAdapter()
