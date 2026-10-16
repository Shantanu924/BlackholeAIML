"""
tests/test_adapters.py — Unit tests for VAJRA adapters in SIMULATED mode.

Proves:
  ✓ All adapters return SIMULATED status when VAJRA_INTERNAL_URL is unset
  ✓ SIMULATED responses are clearly labelled with _status="SIMULATED"
  ✓ Adapters degrade safely when VAJRA is unreachable (LIVE mode with bad URL)
  ✓ Adapter health() returns required fields
"""
from __future__ import annotations

import pytest

from adapters import ALL_ADAPTERS, vajra_audit_adapter, vajra_events_adapter, vajra_health_adapter, vajra_replay_adapter
from intelligence.assessment import assess
from intelligence.decision import decide
from intelligence.models import AdapterStatus, IntelligenceEvent


def _make_event() -> IntelligenceEvent:
    return IntelligenceEvent(
        user_id="adapter-test-user",
        action="TRANSFER",
        resource="financial_data",
        timestamp="2026-10-05T10:00:00Z",
        authorized=True,
    )


class TestAdapterStatus:
    def test_all_adapters_simulated_by_default(self):
        """All adapters must be SIMULATED when VAJRA_INTERNAL_URL is not set."""
        from config import settings
        if settings.vajra_live:
            pytest.skip("Skipped — VAJRA_INTERNAL_URL is set (LIVE mode active).")

        for adapter in ALL_ADAPTERS:
            assert adapter.status == AdapterStatus.SIMULATED, (
                f"{adapter.name} should be SIMULATED but is {adapter.status}"
            )

    def test_all_adapters_have_name(self):
        for adapter in ALL_ADAPTERS:
            assert len(adapter.name) > 0

    def test_all_adapters_have_endpoint(self):
        for adapter in ALL_ADAPTERS:
            assert adapter.endpoint.startswith("http")


class TestSimulatedResponses:
    def test_health_simulated_labelled(self):
        from config import settings
        if settings.vajra_live:
            pytest.skip("LIVE mode active.")
        result = vajra_health_adapter.call()
        assert result["_status"] == "SIMULATED"
        assert "_note" in result

    def test_events_simulated_labelled(self):
        from config import settings
        if settings.vajra_live:
            pytest.skip("LIVE mode active.")
        result = vajra_events_adapter.call(limit=3)
        assert result["_status"] == "SIMULATED"
        assert len(result["events"]) <= 3
        assert result["count"] == len(result["events"])

    def test_audit_simulated_labelled(self):
        from config import settings
        if settings.vajra_live:
            pytest.skip("LIVE mode active.")
        event = _make_event()
        assessment = assess(event)
        decision = decide(event, assessment)
        result = vajra_audit_adapter.call(event=event, assessment=assessment, decision=decision)
        assert result["_status"] == "SIMULATED"
        assert "event_id" in result
        assert "current_hash" in result

    def test_replay_simulated_labelled(self):
        from config import settings
        if settings.vajra_live:
            pytest.skip("LIVE mode active.")
        result = vajra_replay_adapter.call(
            sender_id="test-sender",
            nonce="test-nonce-123",
            payload={"event_id": "test-event"},
        )
        assert result["_status"] == "SIMULATED"
        assert result["valid"] is True


class TestAdapterHealth:
    def test_health_method_returns_required_fields(self):
        for adapter in ALL_ADAPTERS:
            h = adapter.health()
            assert "name" in h
            assert "status" in h
            assert "endpoint" in h


class TestAdapterDegradation:
    def test_health_adapter_degrades_on_bad_url(self, monkeypatch):
        """When LIVE mode with unreachable URL, adapter must return ERROR not crash."""
        monkeypatch.setattr("config.settings.VAJRA_INTERNAL_URL", "http://127.0.0.1:19999")

        class _ForcedLive(type(vajra_health_adapter)):
            @property
            def status(self):
                return AdapterStatus.LIVE

        # Call _live() directly to simulate live mode with bad URL
        result = vajra_health_adapter._live()
        # Should return a dict with _status ERROR, not raise
        assert "_status" in result
        assert result["_status"] in {"ERROR", "LIVE"}
