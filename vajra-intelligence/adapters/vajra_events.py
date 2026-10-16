"""
adapters/vajra_events.py — VAJRA Event Stream Adapter.

Target (internal):  GET {VAJRA_INTERNAL_URL}/api/v1/audit/logs
                    GET {VAJRA_INTERNAL_URL}/api/v1/replay/ledger

Pulls recent events from VAJRA for the intelligence layer to assess.
In LIVE mode, queries VAJRA's Pravah audit log.
In SIMULATED mode, generates a set of synthetic demo events.

Status: SIMULATED (VAJRA_INTERNAL_URL not yet configured)

BLOCKER-001: No public VAJRA event-stream endpoint is accessible externally.
             Requires VAJRA_INTERNAL_URL to be provided.
"""
from __future__ import annotations

import time
import uuid
from datetime import datetime, timezone
from typing import Any

import httpx

from adapters.base import BaseVajraAdapter
from config import settings
from intelligence.models import AdapterStatus


# ── Simulated demo events ─────────────────────────────────────────────────────

_SIMULATED_EVENTS: list[dict[str, Any]] = [
    {
        "event_id": str(uuid.uuid4()),
        "user_id": "wallet-0xABCD1234",
        "action": "TRANSFER",
        "resource": "financial_data",
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "authorized": True,
        "auth_reason": "Scopes match — transfer permitted.",
        "source_system": "VAJRA",
        "reason": "Token transfer to counterparty",
    },
    {
        "event_id": str(uuid.uuid4()),
        "user_id": "wallet-0xDEADBEEF",
        "action": "STAKE",
        "resource": "validator_keys",
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "authorized": True,
        "auth_reason": "Validator staking permitted.",
        "source_system": "VAJRA",
        "reason": "Epoch staking operation",
    },
    {
        "event_id": str(uuid.uuid4()),
        "user_id": "unknown-actor-99",
        "action": "DELETE",
        "resource": "personal_data",
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "authorized": False,
        "auth_reason": "User not found in VAJRA identity store.",
        "source_system": "EXTERNAL",
        "reason": "Data deletion request",
    },
    {
        "event_id": str(uuid.uuid4()),
        "user_id": "bridge-relay-01",
        "action": "BRIDGE",
        "resource": "auth_tokens",
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "authorized": True,
        "auth_reason": "Bridge relay authorized.",
        "source_system": "BRIDGE",
        "reason": "Cross-chain asset relay",
    },
    {
        "event_id": str(uuid.uuid4()),
        "user_id": "governance-actor-07",
        "action": "VOTE",
        "resource": "governance_keys",
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "authorized": True,
        "auth_reason": "Governance vote permitted.",
        "source_system": "VAJRA",
        "reason": "Proposal vote submission",
    },
]


class VajraEventsAdapter(BaseVajraAdapter):
    """
    Pulls recent events from VAJRA's Pravah audit log.

    LIVE:      GET /api/v1/audit/logs?limit=N
    SIMULATED: Returns a fixed set of labelled synthetic demo events.
    """

    @property
    def name(self) -> str:
        return "vajra-event-stream"

    @property
    def endpoint(self) -> str:
        base = settings.VAJRA_INTERNAL_URL or settings.VAJRA_BASE_URL
        return f"{base}/api/v1/audit/logs"

    def call(self, limit: int = 20, **kwargs: Any) -> dict[str, Any]:  # type: ignore[override]
        if self.status == AdapterStatus.SIMULATED:
            return self._simulated(limit)
        return self._live(limit)

    def _live(self, limit: int) -> dict[str, Any]:
        url = f"{settings.VAJRA_INTERNAL_URL.rstrip('/')}/api/v1/audit/logs"
        headers: dict[str, str] = {}
        if settings.VAJRA_API_KEY:
            headers["Authorization"] = f"Bearer {settings.VAJRA_API_KEY}"

        try:
            response = httpx.get(
                url,
                params={"limit": limit},
                headers=headers,
                timeout=settings.VAJRA_TIMEOUT_SECONDS,
            )
            response.raise_for_status()
            return {
                **response.json(),
                "_adapter": self.name,
                "_status": "LIVE",
                "_called_at": time.time(),
            }
        except Exception as exc:
            return {
                "_adapter": self.name,
                "_status": "ERROR",
                "_error": str(exc)[:200],
                "_called_at": time.time(),
                "events": [],
                "count": 0,
            }

    def _simulated(self, limit: int) -> dict[str, Any]:
        events = _SIMULATED_EVENTS[:limit]
        return {
            "count": len(events),
            "events": events,
            "_adapter": self.name,
            "_status": "SIMULATED",
            "_note": (
                "SIMULATED — displaying synthetic demo events. "
                "Set VAJRA_INTERNAL_URL in .env to pull live VAJRA events."
            ),
            "_blocker": "BLOCKER-001: No public VAJRA event-stream endpoint accessible.",
        }


# Module-level singleton
vajra_events_adapter = VajraEventsAdapter()
