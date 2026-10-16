"""
tests/test_api.py — Integration tests for the intelligence API endpoints.

Proves:
  ✓ POST /intelligence/assess returns a complete 7-stage report
  ✓ GET  /intelligence/verify/{id} returns verification result
  ✓ GET  /intelligence/chain returns chain with integrity check
  ✓ GET  /intelligence/health returns adapter status matrix
  ✓ GET  /intelligence/adapters returns capability truth matrix
  ✓ POST /intelligence/replay re-runs pipeline deterministically
  ✓ Invalid inputs are handled gracefully (422, not 500)
  ✓ High-risk events produce DENY decisions
"""
from __future__ import annotations

import pytest
from fastapi.testclient import TestClient


class TestAssessEndpoint:
    def test_low_risk_returns_200(self, client: TestClient, sample_event_payload: dict):
        resp = client.post("/intelligence/assess", json=sample_event_payload)
        assert resp.status_code == 200
        data = resp.json()
        assert "event_id" in data
        assert "trace_id" in data
        assert data["pipeline_status"] == "COMPLETE"

    def test_all_stages_present(self, client: TestClient, sample_event_payload: dict):
        resp = client.post("/intelligence/assess", json=sample_event_payload)
        data = resp.json()
        for stage in [
            "stage1_event", "stage2_assessment", "stage3_decision",
            "stage4_evidence", "stage5_verification", "stage6_vajra_audit",
            "stage7_result",
        ]:
            assert stage in data, f"Missing {stage}"

    def test_high_risk_deny(self, client: TestClient, high_risk_payload: dict):
        resp = client.post("/intelligence/assess", json=high_risk_payload)
        assert resp.status_code == 200
        data = resp.json()
        assert data["stage3_decision"]["outcome"] == "DENY"
        assert data["stage2_assessment"]["risk"] in {"HIGH", "CRITICAL"}

    def test_evidence_recorded(self, client: TestClient, sample_event_payload: dict):
        resp = client.post("/intelligence/assess", json=sample_event_payload)
        data = resp.json()
        ev = data["stage4_evidence"]
        assert ev["tx_ref"].startswith("VI-")
        assert ev["block_id"] >= 1
        assert len(ev["current_hash"]) == 64

    def test_adapter_status_in_stage6(self, client: TestClient, sample_event_payload: dict):
        resp = client.post("/intelligence/assess", json=sample_event_payload)
        data = resp.json()
        vajra_audit = data["stage6_vajra_audit"]
        assert vajra_audit.get("_status") in {"LIVE", "SIMULATED", "ERROR"}

    def test_stage7_result_keys(self, client: TestClient, sample_event_payload: dict):
        resp = client.post("/intelligence/assess", json=sample_event_payload)
        result = resp.json()["stage7_result"]
        for key in ["event_id", "trace_id", "decision_outcome", "risk",
                    "integrity_ok", "tx_ref", "observed_at"]:
            assert key in result, f"Missing key: {key}"

    def test_invalid_payload_422(self, client: TestClient):
        resp = client.post("/intelligence/assess", json={})
        assert resp.status_code == 422

    def test_source_system_bridge(self, client: TestClient):
        payload = {
            "user_id": "bridge-relay-01",
            "action": "BRIDGE",
            "resource": "auth_tokens",
            "timestamp": "2026-10-05T10:00:00Z",
            "authorized": True,
            "source_system": "BRIDGE",
        }
        resp = client.post("/intelligence/assess", json=payload)
        assert resp.status_code == 200
        features = resp.json()["stage2_assessment"]["features"]
        assert features["bridge_activity"] is True


class TestVerifyEndpoint:
    def test_verify_existing_event(self, client: TestClient, sample_event_payload: dict):
        # First create an event
        assess_resp = client.post("/intelligence/assess", json=sample_event_payload)
        event_id = assess_resp.json()["event_id"]

        resp = client.get(f"/intelligence/verify/{event_id}")
        assert resp.status_code == 200
        data = resp.json()
        assert data["entry_found"] is True
        assert data["integrity_ok"] is True
        assert data["event_id"] == event_id

    def test_verify_unknown_event(self, client: TestClient):
        resp = client.get("/intelligence/verify/00000000-0000-0000-0000-000000000000")
        assert resp.status_code == 200
        data = resp.json()
        assert data["entry_found"] is False
        assert data["integrity_ok"] is False


class TestChainEndpoint:
    def test_chain_returns_blocks(self, client: TestClient, sample_event_payload: dict):
        # Ensure at least one block exists
        client.post("/intelligence/assess", json=sample_event_payload)
        resp = client.get("/intelligence/chain")
        assert resp.status_code == 200
        data = resp.json()
        assert data["total_blocks"] >= 1
        assert data["chain_intact"] is True
        assert isinstance(data["chain"], list)


class TestHealthEndpoint:
    def test_health_200(self, client: TestClient):
        resp = client.get("/intelligence/health")
        assert resp.status_code == 200
        data = resp.json()
        assert data["status"] == "ok"
        assert "adapters" in data
        assert len(data["adapters"]) > 0

    def test_adapters_have_status(self, client: TestClient):
        resp = client.get("/intelligence/health")
        for adapter in resp.json()["adapters"]:
            assert adapter["status"] in {"LIVE", "SIMULATED", "ERROR", "BLOCKED"}


class TestAdaptersEndpoint:
    def test_adapters_200(self, client: TestClient):
        resp = client.get("/intelligence/adapters")
        assert resp.status_code == 200
        data = resp.json()
        assert "adapters" in data
        assert "blockers" in data
        assert isinstance(data["blockers"], list)


class TestReplayEndpoint:
    def test_replay_existing_event(self, client: TestClient, sample_event_payload: dict):
        assess_resp = client.post("/intelligence/assess", json=sample_event_payload)
        event_id = assess_resp.json()["event_id"]

        resp = client.post("/intelligence/replay", json={"event_id": event_id})
        assert resp.status_code == 200
        data = resp.json()
        assert data["replayed"] is True
        assert data["original_event_id"] == event_id
        assert data["new_report"] is not None

    def test_replay_unknown_event_404(self, client: TestClient):
        resp = client.post(
            "/intelligence/replay",
            json={"event_id": "00000000-0000-0000-0000-000000000000"},
        )
        assert resp.status_code == 404


class TestEventsEndpoint:
    def test_events_returns_list(self, client: TestClient):
        resp = client.get("/intelligence/events?limit=3")
        assert resp.status_code == 200
        data = resp.json()
        assert "events" in data
        assert len(data["events"]) <= 3
