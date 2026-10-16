"""
intelligence/decision.py — Decision Gate (Stage 3).

Combines VAJRA's own authorization result with the AI recommendation to produce
a final advisory Decision. AI is ADVISORY ONLY — Cybersecurity enforcement
consumes this output and remains the enforcement authority.

Decision logic:
  DENY  → if VAJRA says unauthorized OR AI recommends DENY
  FLAG  → if VAJRA says authorized but AI recommends REVIEW
  ALLOW → if VAJRA says authorized AND AI recommends ALLOW
"""
from __future__ import annotations

from intelligence.models import (
    AssessmentResult,
    Decision,
    IntelligenceEvent,
    Outcome,
    RecommendedAction,
)


def decide(event: IntelligenceEvent, assessment: AssessmentResult) -> Decision:
    """
    Combine VAJRA's permission authorization with the AI recommendation.

    Args:
        event:      The normalised IntelligenceEvent (carries `authorized` flag).
        assessment: The AI/ML AssessmentResult from the risk engine.

    Returns:
        Decision with outcome ALLOW / FLAG / DENY and a human-readable reason.
    """
    ai_rec = assessment.recommended_action
    authorized = event.authorized

    if not authorized or ai_rec == RecommendedAction.DENY:
        outcome = Outcome.DENY
        if not authorized and ai_rec == RecommendedAction.DENY:
            reason = (
                "DENY: both VAJRA authorization and AI assessment reject this request. "
                f"{assessment.reason}"
            )
        elif not authorized:
            reason = (
                f"DENY: VAJRA authorization failed. {event.auth_reason or 'No auth reason provided.'}"
            )
        else:
            reason = (
                f"DENY: AI assessment recommends DENY. {assessment.reason}"
            )

    elif ai_rec == RecommendedAction.REVIEW:
        outcome = Outcome.FLAG
        reason = (
            "FLAG: request is permitted by VAJRA but AI assessment recommends human review. "
            f"{assessment.reason}"
        )

    else:
        outcome = Outcome.ALLOW
        reason = (
            f"ALLOW: VAJRA authorized and AI assessment is clear. "
            f"Risk={assessment.risk.value}, confidence={assessment.confidence:.2f}."
        )

    return Decision(
        event_id=event.event_id,
        outcome=outcome,
        reason=reason,
        ai_recommendation=ai_rec.value,
        permission_authorized=authorized,
        confidence=assessment.confidence,
    )
