"""
adapters/vajra_replay.py — VAJRA Replay Verification Adapter.

Target (internal):  POST {VAJRA_INTERNAL_URL}/api/v1/replay/verify

Checks that an intelligence-layer nonce hasn't been replayed in VAJRA's
Pravah replay mitigation table.

Status: SIMULATED (VAJRA_INTERNAL_URL not yet configured)
"""
from __future__ import annotations

import hashlib
import time
import uuid
from typing import Any

import httpx

from adapters.base import BaseVajraAdapter
from config import settings
from intelligence.models import AdapterStatus


class VajraReplayAdapter(BaseVajraAdapter):
    """
    Verifies transaction uniqueness against VAJRA's Pravah replay ledger.

    LIVE:      POST /api/v1/replay/verify — rejects replays with HTTP 409.
    SIMULATED: Always returns valid=True with a SIMULATED label.
    """

    @property
    def name(self) -> str:
        return "vajra-replay-verify"

    @property
    def endpoint(self) -> str:
        base = settings.VAJRA_INTERNAL_URL or settings.VAJRA_BASE_URL
        return f"{base}/api/v1/replay/verify"

    def call(  # type: ignore[override]
        self,
        sender_id: str,
        nonce: str,
        payload: dict[str, Any],
        trace_id: str | None = None,
        **kwargs: Any,
    ) -> dict[str, Any]:
        if self.status == AdapterStatus.SIMULATED:
            return self._simulated(sender_id, nonce, payload, trace_id)
        return self._live(sender_id, nonce, payload, trace_id)

    def _live(
        self,
        sender_id: str,
        nonce: str,
        payload: dict[str, Any],
        trace_id: str | None,
    ) -> dict[str, Any]:
        url = f"{settings.VAJRA_INTERNAL_URL.rstrip('/')}/api/v1/replay/verify"
        headers: dict[str, str] = {"Content-Type": "application/json"}
        if settings.VAJRA_API_KEY:
            headers["Authorization"] = f"Bearer {settings.VAJRA_API_KEY}"

        body = {
            "sender_id": sender_id,
            "nonce": nonce,
            "payload": payload,
            "trace_id": trace_id or str(uuid.uuid4()),
        }
        try:
            response = httpx.post(
                url, json=body, headers=headers, timeout=settings.VAJRA_TIMEOUT_SECONDS
            )
            response.raise_for_status()
            return {
                **response.json(),
                "_adapter": self.name,
                "_status": "LIVE",
                "_called_at": time.time(),
            }
        except httpx.HTTPStatusError as exc:
            # 409 = replay detected — this is a valid business rejection
            if exc.response.status_code == 409:
                return {
                    "valid": False,
                    "reason": "REPLAY_DETECTED",
                    "detail": exc.response.text[:200],
                    "_adapter": self.name,
                    "_status": "LIVE",
                    "_called_at": time.time(),
                }
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

    def _simulated(
        self,
        sender_id: str,
        nonce: str,
        payload: dict[str, Any],
        trace_id: str | None,
    ) -> dict[str, Any]:
        proof_digest = hashlib.sha256(
            f"{sender_id}:{nonce}:{time.time()}".encode()
        ).hexdigest()
        return {
            "valid": True,
            "status": "ACCEPTED",
            "reason": "Nonce accepted (SIMULATED — not checked against VAJRA Pravah ledger).",
            "proof_digest": f"simulated-{proof_digest[:32]}",
            "trace_id": trace_id or str(uuid.uuid4()),
            "_adapter": self.name,
            "_status": "SIMULATED",
            "_note": (
                "SIMULATED — set VAJRA_INTERNAL_URL in .env to enable LIVE replay verification."
            ),
        }


# Module-level singleton
vajra_replay_adapter = VajraReplayAdapter()
