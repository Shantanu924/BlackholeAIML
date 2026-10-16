"""
adapters/vajra_audit.py — VAJRA Audit Emit Adapter.

Target (internal):  POST {VAJRA_INTERNAL_URL}/api/v1/audit/emit

Publishes intelligence-layer decisions into VAJRA's Pravah immutable audit stream.
This is the primary integration point — Cybersecurity reads from the audit stream
to perform enforcement.

Status: SIMULATED (VAJRA_INTERNAL_URL not yet configured)
"""
from __future__ import annotations

import time
import uuid
from typing import Any

import httpx

from adapters.base import BaseVajraAdapter
from config import settings
from intelligence.models import AdapterStatus, AssessmentResult, Decision, IntelligenceEvent


class VajraAuditAdapter(BaseVajraAdapter):
    """
    Emits an intelligence decision event into VAJRA's Pravah audit stream.

    Payload follows VAJRA's AuditEmitRequest schema:
        event_type, actor, action, resource, status, metadata, payload

    LIVE:      POST /api/v1/audit/emit
    SIMULATED: Logs a labelled local record; nothing reaches VAJRA.
    """

    @property
    def name(self) -> str:
        return "vajra-audit-emit"

    @property
    def endpoint(self) -> str:
        base = settings.VAJRA_INTERNAL_URL or settings.VAJRA_BASE_URL
        return f"{base}/api/v1/audit/emit"

    def call(  # type: ignore[override]
        self,
        event: IntelligenceEvent,
        assessment: AssessmentResult,
        decision: Decision,
        **kwargs: Any,
    ) -> dict[str, Any]:
        if self.status == AdapterStatus.SIMULATED:
            return self._simulated(event, assessment, decision)
        return self._live(event, assessment, decision)

    def _build_payload(
        self,
        event: IntelligenceEvent,
        assessment: AssessmentResult,
        decision: Decision,
    ) -> dict[str, Any]:
        """Build VAJRA AuditEmitRequest-compatible payload."""
        return {
            "event_type": "INTELLIGENCE_DECISION",
            "actor": event.user_id,
            "action": event.action,
            "resource": event.resource,
            "status": decision.outcome.value,
            "metadata": {
                "source_system": event.source_system.value,
                "trace_id": assessment.trace_id,
                "model_id": assessment.model_id,
            },
            "payload": {
                "risk": assessment.risk.value,
                "confidence": assessment.confidence,
                "recommended_action": assessment.recommended_action.value,
                "decision_outcome": decision.outcome.value,
                "reason": decision.reason,
                "features": assessment.features,
            },
        }

    def _live(
        self,
        event: IntelligenceEvent,
        assessment: AssessmentResult,
        decision: Decision,
    ) -> dict[str, Any]:
        """POST to VAJRA /api/v1/audit/emit."""
        url = f"{settings.VAJRA_INTERNAL_URL.rstrip('/')}/api/v1/audit/emit"
        headers: dict[str, str] = {"Content-Type": "application/json"}
        if settings.VAJRA_API_KEY:
            headers["Authorization"] = f"Bearer {settings.VAJRA_API_KEY}"

        payload = self._build_payload(event, assessment, decision)
        try:
            response = httpx.post(
                url,
                json=payload,
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

    def _simulated(
        self,
        event: IntelligenceEvent,
        assessment: AssessmentResult,
        decision: Decision,
    ) -> dict[str, Any]:
        """Return a clearly-labelled SIMULATED audit emit response."""
        payload = self._build_payload(event, assessment, decision)
        return {
            "event_id": str(uuid.uuid4()),
            "timestamp": time.time(),
            "event_type": payload["event_type"],
            "actor": payload["actor"],
            "action": payload["action"],
            "resource": payload["resource"],
            "status": payload["status"],
            "current_hash": "simulated-" + uuid.uuid4().hex[:32],
            "previous_hash": "simulated-" + "0" * 32,
            "_adapter": self.name,
            "_status": "SIMULATED",
            "_note": (
                "SIMULATED — no event was emitted to VAJRA Pravah audit stream. "
                "Set VAJRA_INTERNAL_URL in .env to enable LIVE mode."
            ),
        }


# Module-level singleton
vajra_audit_adapter = VajraAuditAdapter()
