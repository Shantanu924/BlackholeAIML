"""
intelligence/models.py — Core Pydantic schemas for the VAJRA Intelligence Layer.

These are the canonical data contracts for the intelligence pipeline.
All stages communicate via these types. Never import VAJRA root models here.
"""
from __future__ import annotations

import uuid
from enum import Enum
from typing import Any, Optional

from pydantic import BaseModel, Field


# ── Enumerations ────────────────────────────────────────────────────────────

class RiskLevel(str, Enum):
    LOW = "LOW"
    MEDIUM = "MEDIUM"
    HIGH = "HIGH"
    CRITICAL = "CRITICAL"


class StatusFlag(str, Enum):
    CLEAR = "CLEAR"
    FLAGGED = "FLAGGED"
    BLOCKED = "BLOCKED"


class RecommendedAction(str, Enum):
    ALLOW = "ALLOW"
    REVIEW = "REVIEW"
    DENY = "DENY"


class Outcome(str, Enum):
    ALLOW = "ALLOW"
    FLAG = "FLAG"
    DENY = "DENY"


class SourceSystem(str, Enum):
    VAJRA = "VAJRA"
    BRIDGE = "BRIDGE"
    WALLET = "WALLET"
    OTC = "OTC"
    EXTERNAL = "EXTERNAL"
    SIMULATED = "SIMULATED"


# ── Stage 1: Normalised Event ────────────────────────────────────────────────

class IntelligenceEvent(BaseModel):
    """
    Normalised VAJRA event ingested by the intelligence layer.

    Events may originate from VAJRA's event stream, bridge activity,
    wallet transactions, OTC flows, or external systems.
    All sources are normalised into this single schema before assessment.
    """

    event_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    user_id: str = Field(..., description="User or wallet address initiating the event.")
    action: str = Field(
        ...,
        description="Normalised action verb: READ, WRITE, DELETE, EXPORT, TRANSFER, STAKE, UNSTAKE, VOTE, BRIDGE.",
    )
    resource: str = Field(..., description="Target resource or contract.")
    timestamp: str = Field(..., description="ISO 8601 UTC timestamp of the originating event.")
    reason: str = Field(default="", description="Human-readable reason provided with the request.")
    authorized: bool = Field(default=True, description="Whether the event passed VAJRA's own auth check.")
    auth_reason: str = Field(default="", description="VAJRA's auth decision reason (pass-through).")
    granted_scopes: list[str] = Field(default_factory=list)
    source_system: SourceSystem = Field(default=SourceSystem.VAJRA)
    metadata: dict[str, Any] = Field(default_factory=dict, description="Arbitrary source-specific payload.")
    schema_version: str = "1.0.0"


# ── Stage 2: Assessment Result ───────────────────────────────────────────────

class AssessmentResult(BaseModel):
    """
    Output of the AI/ML risk assessment engine (Stage 2).

    The 'features' dict captures every signal that contributed to the score,
    enabling full auditability and replay determinism.
    """

    event_id: str
    risk: RiskLevel
    status: StatusFlag
    reason: str = Field(..., description="Human-readable explanation of the risk verdict.")
    features: dict[str, Any] = Field(..., description="Full feature trace for audit and replay.")
    confidence: float = Field(..., ge=0.0, le=1.0)
    recommended_action: RecommendedAction
    trace_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    model_id: str = Field(default="rule-engine-v1", description="Model/version identifier for model cards.")
    schema_version: str = "1.0.0"


# ── Stage 3: Decision ────────────────────────────────────────────────────────

class Decision(BaseModel):
    """
    Decision Gate output (Stage 3).

    Combines VAJRA's own authorization result with the AI recommendation.
    AI is ADVISORY — Cybersecurity enforcement consumes this and acts.
    """

    event_id: str
    outcome: Outcome
    reason: str
    ai_recommendation: str
    permission_authorized: bool
    confidence: float


# ── Stage 4: Advisory Evidence Record ───────────────────────────────────────

class EvidenceRecord(BaseModel):
    """
    Advisory evidence record — stored in the intelligence layer's own SQLite store.

    This is NOT the VAJRA blockchain. It is an immutable advisory log that the
    intelligence layer maintains for its own audit trail and hash verification.
    VAJRA remains the single source of truth for on-chain evidence.
    """

    record_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    event_id: str
    user_id: str
    action: str
    resource: str
    timestamp: str
    decision_outcome: str
    risk_level: str
    block_id: int = Field(..., description="Sequential block number in intelligence evidence chain.")
    tx_ref: str = Field(..., description="Intelligence-layer transaction reference (NOT a VAJRA tx hash).")
    current_hash: str = Field(..., description="SHA-256 of this record's content + previous_hash.")
    previous_hash: str = Field(..., description="Hash of the preceding evidence record (genesis = '0' * 64).")


# ── Stage 5: Verification ────────────────────────────────────────────────────

class VerificationResult(BaseModel):
    """Result of hash-chain verification for an evidence record."""

    event_id: str
    entry_found: bool
    integrity_ok: bool
    detail: str
    verified_at: str
    evidence_record: Optional[EvidenceRecord] = None


# ── Full Pipeline Output ─────────────────────────────────────────────────────

class IntelligenceReport(BaseModel):
    """
    Complete output of the VAJRA Intelligence 7-stage pipeline.

    stage1_event        → Normalised event
    stage2_assessment   → AI/ML risk assessment
    stage3_decision     → Decision Gate result
    stage4_evidence     → Advisory evidence record
    stage5_verification → Hash-chain verification
    stage6_vajra_audit  → VAJRA adapter emit result (LIVE or SIMULATED)
    stage7_result       → Observable result summary
    """

    event_id: str
    trace_id: str
    stage1_event: dict[str, Any]
    stage2_assessment: dict[str, Any]
    stage3_decision: dict[str, Any]
    stage4_evidence: dict[str, Any]
    stage5_verification: dict[str, Any]
    stage6_vajra_audit: dict[str, Any]
    stage7_result: dict[str, Any]
    pipeline_status: str = "COMPLETE"


# ── Adapter Status ────────────────────────────────────────────────────────────

class AdapterStatus(str, Enum):
    LIVE = "LIVE"
    SIMULATED = "SIMULATED"
    ERROR = "ERROR"
    BLOCKED = "BLOCKED"


class AdapterHealth(BaseModel):
    """Health record for a single VAJRA adapter."""

    name: str
    status: AdapterStatus
    endpoint: str
    detail: str = ""
