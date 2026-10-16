"""
Integration tests for TANTRA Production API Endpoints and Security Envelopes.
"""

import time
import uuid
from starlette.testclient import TestClient


def test_health_endpoints(client: TestClient) -> None:
    """Verify system health routes."""
    resp = client.get("/health")
    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "HEALTHY"
    assert data["task_id"] == "T-GOV-002"

    resp2 = client.get("/api/v1/health")
    assert resp2.status_code == 200


def test_auth_token_lifecycle(client: TestClient) -> None:
    """Verify token generation, JWKS, and verification endpoints."""
    # 1. Public key
    pub_resp = client.get("/api/v1/auth/public-key")
    assert pub_resp.status_code == 200
    assert "-----BEGIN PUBLIC KEY-----" in pub_resp.json()["public_key"]

    # 2. Issue Token
    tok_resp = client.post(
        "/api/v1/auth/token",
        json={
            "sub": "test-engineer",
            "roles": ["operator", "auditor"],
            "permissions": ["replay:verify"],
            "tenant_id": "tantra-mainnet",
            "candidate_id": "Prakash Yadav",
            "ttl_seconds": 1800,
        },
    )
    assert tok_resp.status_code == 200
    tok_data = tok_resp.json()
    token = tok_data["access_token"]
    assert len(token.split(".")) == 3

    # 3. Verify Token
    ver_resp = client.post("/api/v1/auth/verify", json={"token": token})
    assert ver_resp.status_code == 200
    assert ver_resp.json()["valid"] is True
    assert ver_resp.json()["claims"]["sub"] == "test-engineer"


def test_replay_verification_route(client: TestClient) -> None:
    """Verify replay detection returns 200 on first try, and 409 Conflict on replay."""
    unique_tag = uuid.uuid4().hex[:8]
    sender = f"0xRouteValidator-{unique_tag}"
    nonce = int(time.time() * 1000)
    payload = {"tx_hash": f"0xabc-{unique_tag}", "value": 1000}

    # First attempt: 200 OK
    resp1 = client.post(
        "/api/v1/replay/verify",
        json={"sender_id": sender, "nonce": nonce, "payload": payload},
    )
    assert resp1.status_code == 200
    assert resp1.json()["valid"] is True
    assert resp1.json()["status"] == "ACCEPTED"

    # Replay attempt: 409 Conflict
    resp2 = client.post(
        "/api/v1/replay/verify",
        json={"sender_id": sender, "nonce": nonce, "payload": payload},
    )
    assert resp2.status_code == 409
    err = resp2.json()
    assert err["error"] is True
    assert "Replay rejected" in err["message"]


def test_audit_stream_and_chain_verification_routes(client: TestClient) -> None:
    """Verify audit emission and cryptographic chain verification endpoints."""
    # 1. Emit
    emit_resp = client.post(
        "/api/v1/audit/emit",
        json={
            "event_type": "INTEGRATION_TEST_EVENT",
            "actor": "PytestRunner",
            "action": "RUN_API_TEST",
            "resource": "api/v1/test",
            "status": "SUCCESS",
            "metadata": {"suite": "integration"},
            "payload": {"key": "val"},
        },
    )
    assert emit_resp.status_code == 200
    assert len(emit_resp.json()["current_hash"]) == 64

    # 2. Get Logs
    logs_resp = client.get("/api/v1/audit/logs?limit=10")
    assert logs_resp.status_code == 200
    assert logs_resp.json()["count"] > 0

    # 3. Verify Chain
    chain_resp = client.get("/api/v1/audit/verify-chain")
    assert chain_resp.status_code == 200
    assert chain_resp.json()["valid"] is True


def test_rbac_protected_governance_evaluate(client: TestClient, admin_token: str, operator_token: str) -> None:
    """Verify RBAC gating on governance evaluation endpoint."""
    # 1. No token -> 401
    resp_unauth = client.post(
        "/api/v1/governance/evaluate",
        json={"task_id": "T-GOV-002", "candidate_id": "Prakash Yadav"},
    )
    assert resp_unauth.status_code == 401

    # 2. Operator token (has operator role, permitted for evaluation)
    uid1 = uuid.uuid4().hex[:8]
    resp_op = client.post(
        "/api/v1/governance/evaluate",
        json={
            "task_id": "T-GOV-002",
            "candidate_id": "Prakash Yadav",
            "sender_id": f"op-sender-{uid1}",
            "nonce": int(time.time() * 1000),
            "payload": {"req": "op-eval", "uid": uid1},
        },
        headers={"Authorization": f"Bearer {operator_token}"},
    )
    assert resp_op.status_code == 200
    assert resp_op.json()["verdict"] in ("PASS", "FAIL")

    # 3. Admin token -> 200 OK with 100.0 score
    uid2 = uuid.uuid4().hex[:8]
    resp_admin = client.post(
        "/api/v1/governance/evaluate",
        json={
            "task_id": "T-GOV-002",
            "candidate_id": "Prakash Yadav",
            "sender_id": f"admin-sender-{uid2}",
            "nonce": int(time.time() * 1000) + 1,
            "payload": {"req": "admin-eval", "uid": uid2},
        },
        headers={"Authorization": f"Bearer {admin_token}"},
    )
    assert resp_admin.status_code == 200
    assert resp_admin.json()["total_score"] == 100.0


def test_orchestrator_evaluate_task_endpoint(client: TestClient, admin_token: str) -> None:
    """Verify PARIKSHAK task review endpoint."""
    resp = client.post(
        "/api/v1/orchestrator/evaluate-task",
        json={"task_id": "T-GOV-002", "candidate_id": "Prakash Yadav"},
        headers={"Authorization": f"Bearer {admin_token}"},
    )
    assert resp.status_code == 200
    data = resp.json()
    assert data["task_id"] == "T-GOV-002"
    assert data["candidate_id"] == "Prakash Yadav"
    assert data["overall_status"] == "PASS"
    assert data["total_score"] == 100.0
    assert len(data["certified_proof_hash"]) == 64
