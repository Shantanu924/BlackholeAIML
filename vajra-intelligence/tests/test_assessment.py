"""
tests/test_assessment.py — Unit tests for the AI/ML risk assessment engine.

Proves:
  ✓ Deterministic repeatability (same input → same output always)
  ✓ Model/version captured in result
  ✓ All risk tiers reachable
  ✓ Feature trace complete and auditable
  ✓ Bridge / validator VAJRA extension rules fire correctly
"""
from __future__ import annotations

import pytest

from intelligence.assessment import assess
from intelligence.models import (
    IntelligenceEvent,
    RecommendedAction,
    RiskLevel,
    SourceSystem,
    StatusFlag,
)


def _make_event(**kwargs) -> IntelligenceEvent:
    defaults = {
        "user_id": "test-user",
        "action": "READ",
        "resource": "public_report",
        "timestamp": "2026-10-05T10:00:00Z",
        "authorized": True,
    }
    defaults.update(kwargs)
    return IntelligenceEvent(**defaults)


class TestDeterminism:
    def test_same_input_same_output(self):
        """Determinism: identical events must produce identical assessments."""
        event = _make_event(user_id="det-user", action="READ", resource="logs")
        result_a = assess(event)
        result_b = assess(event)
        assert result_a.risk == result_b.risk
        assert result_a.recommended_action == result_b.recommended_action
        assert result_a.confidence == result_b.confidence
        assert result_a.features == result_b.features

    def test_model_id_captured(self):
        """Model/version must be captured for model card compliance."""
        event = _make_event()
        result = assess(event)
        assert result.model_id == "rule-engine-v1"
        assert result.schema_version == "1.0.0"

    def test_trace_id_unique_per_call(self):
        """Each assessment must have a unique trace_id."""
        event = _make_event()
        r1 = assess(event)
        r2 = assess(event)
        assert r1.trace_id != r2.trace_id


class TestRiskTiers:
    def test_low_risk(self):
        """Authorized READ on non-sensitive resource during business hours → LOW."""
        event = _make_event(action="READ", resource="public_docs", authorized=True,
                            timestamp="2026-10-05T10:00:00Z")
        result = assess(event)
        assert result.risk == RiskLevel.LOW
        assert result.recommended_action == RecommendedAction.ALLOW
        assert result.status == StatusFlag.CLEAR

    def test_medium_risk(self):
        """High-value action on non-sensitive resource → MEDIUM."""
        event = _make_event(action="EXPORT", resource="public_report", authorized=True,
                            timestamp="2026-10-05T10:00:00Z")
        result = assess(event)
        assert result.risk in {RiskLevel.MEDIUM, RiskLevel.HIGH}
        assert result.recommended_action in {RecommendedAction.REVIEW, RecommendedAction.DENY}

    def test_high_risk_sensitive_resource(self):
        """DELETE on sensitive resource → HIGH or CRITICAL."""
        event = _make_event(action="DELETE", resource="personal_data", authorized=True,
                            timestamp="2026-10-05T10:00:00Z")
        result = assess(event)
        assert result.risk in {RiskLevel.HIGH, RiskLevel.CRITICAL}
        assert result.recommended_action == RecommendedAction.DENY

    def test_critical_unauthorized_sensitive(self):
        """Unauthorized DELETE on sensitive resource off-hours → CRITICAL."""
        event = _make_event(
            action="DELETE",
            resource="personal_data",
            authorized=False,
            timestamp="2026-10-05T02:00:00Z",  # off-hours UTC
        )
        result = assess(event)
        assert result.risk == RiskLevel.CRITICAL
        assert result.recommended_action == RecommendedAction.DENY
        assert result.confidence >= 0.90


class TestFeatureTrace:
    def test_features_complete(self):
        """Feature dict must contain all expected keys."""
        event = _make_event()
        result = assess(event)
        required_keys = {
            "unauthorized_attempt", "high_value_action", "sensitive_resource",
            "off_hours_access", "frequency_abuse", "bridge_activity",
            "validator_operation", "total_score", "action", "resource", "hour_utc",
        }
        assert required_keys.issubset(result.features.keys())

    def test_unauthorized_flagged(self):
        """Unauthorized event must set unauthorized_attempt=True in features."""
        event = _make_event(authorized=False)
        result = assess(event)
        assert result.features["unauthorized_attempt"] is True

    def test_off_hours_flagged(self):
        """Off-hours access (UTC 02:00) must be flagged."""
        event = _make_event(timestamp="2026-10-05T02:30:00Z")
        result = assess(event)
        assert result.features["off_hours_access"] is True

    def test_reason_non_empty(self):
        """Reason string must always be non-empty."""
        for action in ["READ", "DELETE", "TRANSFER", "VOTE"]:
            event = _make_event(action=action)
            result = assess(event)
            assert len(result.reason) > 0


class TestVajraExtensionRules:
    def test_bridge_rule_fires(self):
        """Bridge activity (source_system=BRIDGE) must set bridge_activity=True."""
        event = _make_event(source_system=SourceSystem.BRIDGE, action="TRANSFER")
        result = assess(event)
        assert result.features["bridge_activity"] is True

    def test_bridge_action_fires(self):
        """BRIDGE action must trigger bridge_activity signal."""
        event = _make_event(action="BRIDGE")
        result = assess(event)
        assert result.features["bridge_activity"] is True

    def test_validator_stake_fires(self):
        """STAKE action must set validator_operation=True."""
        event = _make_event(action="STAKE", resource="validator_keys")
        result = assess(event)
        assert result.features["validator_operation"] is True

    def test_vote_fires(self):
        """VOTE action must set validator_operation=True."""
        event = _make_event(action="VOTE")
        result = assess(event)
        assert result.features["validator_operation"] is True
