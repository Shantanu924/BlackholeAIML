"""
api/schemas.py — FastAPI request/response schemas for the intelligence API.
"""
from __future__ import annotations

from typing import Any, Optional

from pydantic import BaseModel, Field


# ── Request Schemas ──────────────────────────────────────────────────────────

class AssessRequest(BaseModel):
    """POST /intelligence/assess — run the full intelligence pipeline."""

    user_id: str = Field(..., description="User or wallet address initiating the event.")
    action: str = Field(
        ...,
        description="Action verb: READ, WRITE, DELETE, EXPORT, TRANSFER, STAKE, UNSTAKE, VOTE, BRIDGE.",
        examples=["TRANSFER"],
    )
    resource: str = Field(..., description="Target resource or contract.", examples=["financial_data"])
    timestamp: str = Field(..., description="ISO 8601 UTC timestamp.", examples=["2026-10-05T04:00:00Z"])
    reason: str = Field(default="", description="Human-readable reason for the request.")
    authorized: bool = Field(default=True, description="Whether VAJRA's own auth check passed.")
    auth_reason: str = Field(default="", description="VAJRA auth decision reason.")
    granted_scopes: list[str] = Field(default_factory=list)
    source_system: str = Field(default="VAJRA", description="Origin system: VAJRA, BRIDGE, WALLET, OTC, EXTERNAL.")
    metadata: dict[str, Any] = Field(default_factory=dict, description="Arbitrary source-specific payload.")


# ── Response Schemas ─────────────────────────────────────────────────────────

class AssessResponse(BaseModel):
    """Full 7-stage intelligence pipeline response."""

    event_id: str
    trace_id: str
    stage1_event: dict[str, Any]
    stage2_assessment: dict[str, Any]
    stage3_decision: dict[str, Any]
    stage4_evidence: dict[str, Any]
    stage5_verification: dict[str, Any]
    stage6_vajra_audit: dict[str, Any]
    stage7_result: dict[str, Any]
    pipeline_status: str


class VerifyResponse(BaseModel):
    """GET /intelligence/verify/{event_id} response."""

    event_id: str
    entry_found: bool
    integrity_ok: bool
    detail: str
    verified_at: str
    evidence_record: Optional[dict[str, Any]] = None


class ChainResponse(BaseModel):
    """GET /intelligence/chain response."""

    total_blocks: int
    chain_intact: bool
    integrity_detail: str
    chain: list[dict[str, Any]]


class AdapterHealthItem(BaseModel):
    name: str
    status: str
    endpoint: str
    detail: str = ""


class HealthResponse(BaseModel):
    """GET /intelligence/health response."""

    status: str
    service: str
    version: str
    timestamp: float
    vajra_live_mode: bool
    adapters: list[AdapterHealthItem]


class ReplayRequest(BaseModel):
    """POST /intelligence/replay — re-run pipeline on a previously seen event_id."""

    event_id: str = Field(..., description="event_id of a previously assessed event.")


class ReplayResponse(BaseModel):
    """POST /intelligence/replay response."""

    original_event_id: str
    replayed: bool
    detail: str
    new_report: Optional[dict[str, Any]] = None
