"""
api/routes.py — FastAPI routes for the VAJRA Intelligence Layer.

Endpoints:
  POST /intelligence/assess          → Full 6-stage intelligence pipeline
  GET  /intelligence/verify/{id}     → Re-verify evidence chain for event_id
  GET  /intelligence/chain           → Full advisory evidence chain
  GET  /intelligence/health          → Service liveness + adapter status matrix
  GET  /intelligence/adapters        → Adapter capability truth matrix
  POST /intelligence/replay          → Replay pipeline on a past event
  GET  /intelligence/events          → Pull recent events from VAJRA (adapter)
"""
from __future__ import annotations

import json
import logging
import time
import uuid
from datetime import datetime, timezone

from fastapi import APIRouter, HTTPException

from adapters import ALL_ADAPTERS, vajra_audit_adapter, vajra_events_adapter
from api.schemas import (
    AdapterHealthItem,
    AssessRequest,
    AssessResponse,
    ChainResponse,
    HealthResponse,
    ReplayRequest,
    ReplayResponse,
    VerifyResponse,
)
from config import RUNTIME_LOGS_DIR, settings
from intelligence.assessment import assess
from intelligence.decision import decide
from intelligence.evidence import append, get_by_event_id, get_full_chain, init_db, verify_chain
from intelligence.models import IntelligenceEvent, SourceSystem
from intelligence.verification import verify_record

router = APIRouter(prefix="/intelligence", tags=["Intelligence"])
logger = logging.getLogger("vajra.intelligence.api")


# ── Helpers ───────────────────────────────────────────────────────────────────

def _write_replay_log(payload: dict) -> str:
    """Append an ObservableResult to the JSONL replay log. Returns path."""
    log_path = RUNTIME_LOGS_DIR / "intelligence_replay.jsonl"
    try:
        with open(log_path, "a", encoding="utf-8") as f:
            f.write(json.dumps(payload) + "\n")
    except Exception as exc:
        logger.warning("Failed to write replay log: %s", exc)
    return str(log_path)


def _build_event(req: AssessRequest) -> IntelligenceEvent:
    """Normalise an AssessRequest into an IntelligenceEvent."""
    try:
        source = SourceSystem(req.source_system.upper())
    except ValueError:
        source = SourceSystem.EXTERNAL

    return IntelligenceEvent(
        user_id=req.user_id,
        action=req.action.upper(),
        resource=req.resource,
        timestamp=req.timestamp,
        reason=req.reason,
        authorized=req.authorized,
        auth_reason=req.auth_reason,
        granted_scopes=req.granted_scopes,
        source_system=source,
        metadata=req.metadata,
    )


def _run_pipeline(event: IntelligenceEvent) -> AssessResponse:
    """Execute the full intelligence pipeline for a given event."""
    logger.info("[Stage 1] Event ingested: event_id=%s user=%s", event.event_id, event.user_id)

    # Stage 2 — AI/ML Assessment
    assessment = assess(event)
    logger.info(
        "[Stage 2] Assessment: risk=%s action=%s trace_id=%s",
        assessment.risk.value,
        assessment.recommended_action.value,
        assessment.trace_id,
    )

    # Stage 3 — Decision Gate
    decision = decide(event, assessment)
    logger.info("[Stage 3] Decision: outcome=%s event_id=%s", decision.outcome.value, event.event_id)

    # Stage 4 — Advisory Evidence Recording
    evidence = append(event, decision, assessment.risk.value)
    logger.info("[Stage 4] Evidence: block_id=%s tx_ref=%s", evidence.block_id, evidence.tx_ref)

    # Stage 5 — Verification
    verification = verify_record(event.event_id)
    logger.info(
        "[Stage 5] Verification: integrity_ok=%s event_id=%s",
        verification.integrity_ok,
        event.event_id,
    )

    # Stage 6 — VAJRA Audit Emit (adapter: LIVE or SIMULATED)
    vajra_audit = vajra_audit_adapter.call(
        event=event,
        assessment=assessment,
        decision=decision,
    )
    logger.info("[Stage 6] VAJRA audit emit: status=%s", vajra_audit.get("_status"))

    # Stage 7 — Observable Result
    result_payload = {
        "event_id": event.event_id,
        "trace_id": assessment.trace_id,
        "decision_outcome": decision.outcome.value,
        "risk": assessment.risk.value,
        "confidence": assessment.confidence,
        "integrity_ok": verification.integrity_ok,
        "tx_ref": evidence.tx_ref,
        "block_id": evidence.block_id,
        "model_id": assessment.model_id,
        "adapter_status": vajra_audit.get("_status"),
        "timestamp": event.timestamp,
        "observed_at": datetime.now(timezone.utc).isoformat(),
    }
    replay_log = _write_replay_log(result_payload)
    result_payload["replay_log"] = replay_log
    logger.info("[Stage 7] Result written: %s", replay_log)

    return AssessResponse(
        event_id=event.event_id,
        trace_id=assessment.trace_id,
        stage1_event=event.model_dump(),
        stage2_assessment=assessment.model_dump(),
        stage3_decision=decision.model_dump(),
        stage4_evidence=evidence.model_dump(),
        stage5_verification=verification.model_dump(),
        stage6_vajra_audit=vajra_audit,
        stage7_result=result_payload,
        pipeline_status="COMPLETE",
    )


# ── Routes ────────────────────────────────────────────────────────────────────

@router.post("/assess", response_model=AssessResponse, summary="Run full intelligence pipeline")
def assess_endpoint(req: AssessRequest) -> AssessResponse:
    """
    Run the complete VAJRA Intelligence pipeline:

    1. Normalise event
    2. AI/ML risk assessment (deterministic rule engine)
    3. Decision Gate (ALLOW / FLAG / DENY)
    4. Advisory evidence recording (hash-chained SQLite)
    5. Hash-chain verification
    6. VAJRA audit emit (LIVE or SIMULATED)
    7. Observable result (logged to replay JSONL)

    AI decision is **ADVISORY** — Cybersecurity enforcement authority is VAJRA.
    """
    init_db()
    event = _build_event(req)
    try:
        return _run_pipeline(event)
    except Exception as exc:
        logger.exception("Pipeline error for event_id=%s: %s", event.event_id, exc)
        raise HTTPException(status_code=500, detail=f"Pipeline error: {exc}") from exc


@router.get(
    "/verify/{event_id}",
    response_model=VerifyResponse,
    summary="Verify advisory evidence chain for an event",
)
def verify_endpoint(event_id: str) -> VerifyResponse:
    """Re-run Stage 5 verification on an existing advisory evidence record."""
    result = verify_record(event_id)
    return VerifyResponse(
        event_id=result.event_id,
        entry_found=result.entry_found,
        integrity_ok=result.integrity_ok,
        detail=result.detail,
        verified_at=result.verified_at,
        evidence_record=result.evidence_record.model_dump() if result.evidence_record else None,
    )


@router.get("/chain", response_model=ChainResponse, summary="Inspect full advisory evidence chain")
def chain_endpoint() -> ChainResponse:
    """Return all advisory evidence records and verify the full hash chain."""
    init_db()
    chain = get_full_chain()
    intact, detail = verify_chain()
    return ChainResponse(
        total_blocks=len(chain),
        chain_intact=intact,
        integrity_detail=detail,
        chain=[r.model_dump() for r in chain],
    )


@router.get("/health", response_model=HealthResponse, summary="Service health and adapter status")
def health_endpoint() -> HealthResponse:
    """
    Returns intelligence service liveness + the integration health matrix
    showing LIVE / SIMULATED / ERROR status for each VAJRA adapter.
    """
    adapter_items = [
        AdapterHealthItem(
            name=a.name,
            status=a.status.value,
            endpoint=a.endpoint,
        )
        for a in ALL_ADAPTERS
    ]
    return HealthResponse(
        status="ok",
        service=settings.SERVICE_NAME,
        version=settings.SERVICE_VERSION,
        timestamp=time.time(),
        vajra_live_mode=settings.vajra_live,
        adapters=adapter_items,
    )


@router.get("/adapters", summary="Capability truth matrix")
def adapters_endpoint() -> dict:
    """
    Full capability truth matrix — shows the status of every VAJRA adapter.
    Status: LIVE | SIMULATED | ERROR | BLOCKED
    """
    return {
        "vajra_live_mode": settings.vajra_live,
        "vajra_base_url": settings.VAJRA_BASE_URL,
        "vajra_internal_url": bool(settings.VAJRA_INTERNAL_URL),  # masked — don't leak URL
        "adapters": [a.health() for a in ALL_ADAPTERS],
        "blockers": [
            {
                "id": "BLOCKER-001",
                "description": (
                    "No public VAJRA API endpoints accessible externally. "
                    "Set VAJRA_INTERNAL_URL in .env to flip all adapters to LIVE."
                ),
                "affected_adapters": ["vajra-event-stream", "vajra-audit-emit", "vajra-replay-verify"],
            }
        ],
    }


@router.post("/replay", response_model=ReplayResponse, summary="Replay pipeline on past event")
def replay_endpoint(req: ReplayRequest) -> ReplayResponse:
    """
    Re-fetch a past event from the advisory evidence store and re-run
    the assessment + decision pipeline to verify deterministic reproducibility.

    Note: the replay will create a NEW evidence record with a new block_id.
    The original record is not modified.
    """
    original = get_by_event_id(req.event_id)
    if original is None:
        raise HTTPException(
            status_code=404,
            detail=f"No advisory evidence record found for event_id={req.event_id!r}.",
        )

    # Reconstruct the event from the stored evidence
    replayed_event = IntelligenceEvent(
        event_id=str(uuid.uuid4()),  # new ID for the replay record
        user_id=original.user_id,
        action=original.action,
        resource=original.resource,
        timestamp=original.timestamp,
        authorized=(original.decision_outcome != "DENY"),
        source_system=SourceSystem.SIMULATED,
        metadata={"replay_of": req.event_id},
    )

    try:
        report = _run_pipeline(replayed_event)
        return ReplayResponse(
            original_event_id=req.event_id,
            replayed=True,
            detail=f"Replay complete. New event_id={replayed_event.event_id}.",
            new_report=report.model_dump(),
        )
    except Exception as exc:
        logger.exception("Replay error: %s", exc)
        return ReplayResponse(
            original_event_id=req.event_id,
            replayed=False,
            detail=f"Replay failed: {exc}",
        )


@router.get("/events", summary="Pull recent events from VAJRA (adapter)")
def events_endpoint(limit: int = 20) -> dict:
    """
    Pull recent events from VAJRA's event stream.
    Returns SIMULATED events until VAJRA_INTERNAL_URL is configured.
    """
    return vajra_events_adapter.call(limit=limit)
