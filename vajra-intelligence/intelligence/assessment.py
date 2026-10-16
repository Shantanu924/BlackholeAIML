"""
intelligence/assessment.py — AI/ML Risk Assessment (Stage 2).

Wraps the deterministic rule engine and returns a fully structured AssessmentResult.
"""
from __future__ import annotations

from intelligence.models import (
    AssessmentResult,
    IntelligenceEvent,
    RecommendedAction,
    RiskLevel,
    StatusFlag,
)
from intelligence.rules import build_reason, map_score_to_result, score_event


def assess(event: IntelligenceEvent) -> AssessmentResult:
    """
    Assess an IntelligenceEvent and return a structured AssessmentResult.

    The assessment is:
    - Deterministic: same input always produces same output
    - Explainable: every contributing signal is in 'features'
    - Auditable: trace_id + model_id logged for model cards
    - Replayable: re-run assess(event) to reproduce the exact result

    Args:
        event: Normalised IntelligenceEvent from any source system.

    Returns:
        AssessmentResult with risk tier, recommended action, and full feature trace.
    """
    score, features = score_event(event)
    risk_str, status_str, action_str, confidence = map_score_to_result(score)
    reason = build_reason(features, risk_str)

    return AssessmentResult(
        event_id=event.event_id,
        risk=RiskLevel(risk_str),
        status=StatusFlag(status_str),
        reason=reason,
        features=features,
        confidence=confidence,
        recommended_action=RecommendedAction(action_str),
        model_id="rule-engine-v1",
    )
