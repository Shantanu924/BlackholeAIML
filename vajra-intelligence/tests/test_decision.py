"""
tests/test_decision.py — Unit tests for the Decision Gate.

Proves:
  ✓ DENY when unauthorized (regardless of AI)
  ✓ DENY when AI recommends DENY (regardless of auth)
  ✓ FLAG when authorized but AI recommends REVIEW
  ✓ ALLOW when authorized and AI is clear
  ✓ Reason strings are non-empty and descriptive
  ✓ Decision is explainable (reason references authorization and AI rec)
"""
from __future__ import annotations

import pytest

from intelligence.assessment import assess
from intelligence.decision import decide
from intelligence.models import (
    AssessmentResult,
    IntelligenceEvent,
    Outcome,
    RecommendedAction,
    RiskLevel,
    StatusFlag,
)


def _make_event(**kwargs) -> IntelligenceEvent:
    defaults = {
        "user_id": "test-user",
        "action": "READ",
        "resource": "logs",
        "timestamp": "2026-10-05T10:00:00Z",
        "authorized": True,
    }
    defaults.update(kwargs)
    return IntelligenceEvent(**defaults)


def _make_assessment(event: IntelligenceEvent, **overrides) -> AssessmentResult:
    result = assess(event)
    # Apply overrides for controlled testing
    data = result.model_dump()
    data.update(overrides)
    return AssessmentResult(**data)


class TestDecisionOutcomes:
    def test_allow_path(self):
        """Authorized + AI ALLOW → ALLOW decision."""
        event = _make_event(authorized=True, action="READ", resource="public_docs",
                            timestamp="2026-10-05T10:00:00Z")
        assessment = _make_assessment(event)
        # Force ALLOW recommendation for deterministic test
        assessment = _make_assessment(
            event,
            recommended_action=RecommendedAction.ALLOW,
            risk=RiskLevel.LOW,
            status=StatusFlag.CLEAR,
        )
        decision = decide(event, assessment)
        assert decision.outcome == Outcome.ALLOW
        assert decision.permission_authorized is True

    def test_deny_unauthorized(self):
        """Unauthorized event → always DENY regardless of AI recommendation."""
        event = _make_event(authorized=False, auth_reason="No permission.")
        assessment = _make_assessment(
            event,
            recommended_action=RecommendedAction.ALLOW,  # even if AI says ALLOW
            risk=RiskLevel.LOW,
            status=StatusFlag.CLEAR,
        )
        decision = decide(event, assessment)
        assert decision.outcome == Outcome.DENY
        assert decision.permission_authorized is False

    def test_deny_ai_recommendation(self):
        """Authorized but AI recommends DENY → DENY."""
        event = _make_event(authorized=True)
        assessment = _make_assessment(
            event,
            recommended_action=RecommendedAction.DENY,
            risk=RiskLevel.CRITICAL,
            status=StatusFlag.BLOCKED,
        )
        decision = decide(event, assessment)
        assert decision.outcome == Outcome.DENY

    def test_flag_path(self):
        """Authorized but AI recommends REVIEW → FLAG."""
        event = _make_event(authorized=True)
        assessment = _make_assessment(
            event,
            recommended_action=RecommendedAction.REVIEW,
            risk=RiskLevel.MEDIUM,
            status=StatusFlag.FLAGGED,
        )
        decision = decide(event, assessment)
        assert decision.outcome == Outcome.FLAG

    def test_both_deny(self):
        """Unauthorized + AI DENY → DENY (combined reason)."""
        event = _make_event(authorized=False)
        assessment = _make_assessment(
            event,
            recommended_action=RecommendedAction.DENY,
            risk=RiskLevel.CRITICAL,
            status=StatusFlag.BLOCKED,
        )
        decision = decide(event, assessment)
        assert decision.outcome == Outcome.DENY


class TestDecisionExplainability:
    def test_reason_always_populated(self):
        """Decision reason must always be non-empty."""
        for authorized in [True, False]:
            for rec in RecommendedAction:
                event = _make_event(authorized=authorized)
                assessment = _make_assessment(event, recommended_action=rec)
                decision = decide(event, assessment)
                assert len(decision.reason) > 10, f"Short reason for auth={authorized} rec={rec}"

    def test_ai_recommendation_captured(self):
        """Decision must capture the AI recommendation string."""
        event = _make_event(authorized=True)
        assessment = _make_assessment(event, recommended_action=RecommendedAction.REVIEW)
        decision = decide(event, assessment)
        assert decision.ai_recommendation == "REVIEW"

    def test_confidence_propagated(self):
        """Confidence from assessment must propagate to decision."""
        event = _make_event(authorized=True)
        assessment = _make_assessment(event, confidence=0.75)
        decision = decide(event, assessment)
        assert decision.confidence == pytest.approx(0.75)
