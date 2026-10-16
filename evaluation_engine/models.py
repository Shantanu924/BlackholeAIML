"""
Strict Pydantic models for Evaluation Engine and Rule Definitions.
Enforces zero unhandled extra fields with extra='forbid'.
"""

from typing import Any, Dict, List, Optional
from pydantic import BaseModel, Field


class RuleDefinition(BaseModel):
    """Definition of an architectural or governance rule."""
    rule_id: str = Field(..., description="Unique rule identifier (e.g. RULE-RBAC-01)")
    name: str = Field(..., description="Human-readable rule name")
    category: str = Field(..., description="SECURITY, CRYPTO, REPLAY, DETERMINISM, ROBUSTNESS")
    weight: float = Field(default=20.0, ge=0.0, le=100.0, description="Weight percentage in scoring")
    description: str = Field(..., description="Detailed criteria for passing the rule")

    model_config = {"extra": "forbid"}


class RuleEvaluationResult(BaseModel):
    """Result of evaluating a single rule against a context."""
    rule_id: str
    name: str
    category: str
    passed: bool
    score: float
    max_score: float
    details: str
    evidence_digest: str

    model_config = {"extra": "forbid"}


class EvaluationContext(BaseModel):
    """Context data provided for rule evaluation."""
    task_id: str = Field(default="T-GOV-002")
    candidate_id: str = Field(default="Prakash Yadav")
    actor: str = Field(default="Prakash Yadav")
    token: Optional[str] = Field(default=None)
    sender_id: Optional[str] = Field(default=None)
    nonce: Optional[int] = Field(default=None)
    payload: Optional[Dict[str, Any]] = Field(default=None)
    required_roles: Optional[List[str]] = Field(default=None)
    required_permissions: Optional[List[str]] = Field(default=None)
    run_50_determinism_check: bool = Field(default=True)
    trace_id: Optional[str] = Field(default=None)

    model_config = {"extra": "forbid"}


class GateVerdict(BaseModel):
    """Overall PARIKSHAK quality gate evaluation verdict."""
    task_id: str
    candidate_id: str
    verdict: str  # "PASS" or "FAIL"
    total_score: float = Field(..., ge=0.0, le=100.0)
    passed_rules: int
    failed_rules: int
    rule_results: List[RuleEvaluationResult]
    proof_digest: str
    timestamp: float

    model_config = {"extra": "forbid"}
