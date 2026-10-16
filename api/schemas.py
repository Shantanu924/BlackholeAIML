"""
Strict Pydantic schemas for the TANTRA Production API.
All models strictly enforce extra='forbid' to satisfy PARIKSHAK Gate 5.
"""

from typing import Any, Dict, List, Optional
from pydantic import BaseModel, Field


class TokenRequest(BaseModel):
    """Request payload to generate an RS256 enterprise JWT."""
    sub: str = Field(..., description="Subject identity or service name")
    roles: Optional[List[str]] = Field(default_factory=lambda: ["operator"])
    permissions: Optional[List[str]] = Field(default_factory=list)
    tenant_id: str = Field(default="tantra-mainnet")
    candidate_id: Optional[str] = Field(default="Prakash Yadav")
    ttl_seconds: int = Field(default=3600, ge=60, le=86400)

    model_config = {"extra": "forbid"}


class TokenResponse(BaseModel):
    """Response payload containing issued RS256 token."""
    access_token: str
    token_type: str = "Bearer"
    expires_in: int
    claims: Dict[str, Any]

    model_config = {"extra": "forbid"}


class VerifyTokenRequest(BaseModel):
    """Request to verify an RS256 token."""
    token: str

    model_config = {"extra": "forbid"}


class VerifyTokenResponse(BaseModel):
    """Response with verified claims."""
    valid: bool
    claims: Dict[str, Any]

    model_config = {"extra": "forbid"}


class ReplayVerifyRequest(BaseModel):
    """Request to register and verify transaction against Pravah replay ledger."""
    sender_id: str
    nonce: int = Field(..., ge=0)
    payload: Dict[str, Any]
    trace_id: Optional[str] = None

    model_config = {"extra": "forbid"}


class ReplayVerifyResponse(BaseModel):
    """Verification verdict and cryptographic replay proof."""
    valid: bool
    status: str
    reason: str
    proof_digest: str
    trace_id: str

    model_config = {"extra": "forbid"}


class AuditEmitRequest(BaseModel):
    """Request to emit an event into the Pravah audit stream."""
    event_type: str
    actor: str
    action: str
    resource: str
    status: str
    metadata: Dict[str, Any] = Field(default_factory=dict)
    payload: Optional[Dict[str, Any]] = None

    model_config = {"extra": "forbid"}


class AuditEventResponse(BaseModel):
    """Audit event record with cryptographic hash chain link."""
    event_id: str
    timestamp: float
    event_type: str
    actor: str
    action: str
    resource: str
    status: str
    current_hash: str
    previous_hash: str

    model_config = {"extra": "forbid"}


class AuditVerifyChainResponse(BaseModel):
    """Cryptographic chain verification outcome."""
    valid: bool
    message: str

    model_config = {"extra": "forbid"}


class RuleEvaluateRequest(BaseModel):
    """Request to execute governance rule evaluation."""
    task_id: str = Field(default="T-GOV-002")
    candidate_id: str = Field(default="Prakash Yadav")
    actor: str = Field(default="Prakash Yadav")
    token: Optional[str] = None
    sender_id: Optional[str] = None
    nonce: Optional[int] = None
    payload: Optional[Dict[str, Any]] = None
    run_50_determinism_check: bool = Field(default=True)

    model_config = {"extra": "forbid"}


class StateTransitionRequest(BaseModel):
    """Request to execute a deterministic state transition."""
    target_state: str
    actor: str
    reason: str = Field(default="")
    metadata: Dict[str, Any] = Field(default_factory=dict)

    model_config = {"extra": "forbid"}


class StateTransitionResponse(BaseModel):
    """Result of state transition."""
    success: bool
    current_state: str
    step_hash: str
    state_hash: str

    model_config = {"extra": "forbid"}


class StateHashResponse(BaseModel):
    """Current state hash."""
    state_hash: str
    current_state: str
    step_count: int

    model_config = {"extra": "forbid"}


class TaskReviewRequest(BaseModel):
    """Request for PARIKSHAK review evaluation."""
    task_id: str = Field(default="T-GOV-002")
    candidate_id: str = Field(default="Prakash Yadav")
    requirements_path: str = Field(default="requirements.txt")

    model_config = {"extra": "forbid"}


class TaskReviewResponse(BaseModel):
    """Full task review evaluation response."""
    task_id: str
    candidate_id: str
    overall_status: str
    total_score: float
    certified_proof_hash: str
    llm_summary: Dict[str, Any]

    model_config = {"extra": "forbid"}


class ErrorEnvelope(BaseModel):
    """Standardized error boundary payload."""
    error: bool = True
    code: str
    message: str
    trace_id: str
    timestamp: float

    model_config = {"extra": "forbid"}
