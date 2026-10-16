"""
Integration tests for Error Boundary Protection and Malformed Input Rejection.
Verifies PARIKSHAK Gate 4 (Zero Unhandled 500s) and Gate 5 (Strict Pydantic Validation).
"""

from starlette.testclient import TestClient


def test_malformed_json_body_handled(client: TestClient) -> None:
    """Verify that unparseable raw JSON strings are intercepted safely."""
    resp = client.post(
        "/api/v1/auth/token",
        content="{\"sub\": \"broken_json, missing_quote: 123",
        headers={"Content-Type": "application/json"},
    )
    # Must be 400 or 422, NEVER 500
    assert resp.status_code in (400, 422)
    assert resp.status_code != 500
    data = resp.json()
    assert data["error"] is True


def test_extra_forbidden_fields_rejected_422(client: TestClient) -> None:
    """Verify Gate 5: extra='forbid' strictly rejects unmapped fields with 422."""
    resp = client.post(
        "/api/v1/auth/token",
        json={
            "sub": "valid_user",
            "roles": ["operator"],
            "unmapped_hostile_field": "malicious_injection",  # Extra forbidden
        },
    )
    assert resp.status_code == 422
    assert resp.status_code != 500
    data = resp.json()
    assert data["error"] is True
    assert data["code"] == "MALFORMED_INPUT"


def test_missing_required_fields_rejected_422(client: TestClient) -> None:
    """Verify that omitting required fields produces clean 422 without 500 crash."""
    resp = client.post(
        "/api/v1/auth/token",
        json={"roles": ["operator"]},  # Missing 'sub'
    )
    assert resp.status_code == 422
    assert resp.status_code != 500
    data = resp.json()
    assert data["error"] is True
    assert data["code"] == "MALFORMED_INPUT"


def test_invalid_data_types_rejected_422(client: TestClient) -> None:
    """Verify that type mismatches (string instead of integer nonce) return 422."""
    resp = client.post(
        "/api/v1/replay/verify",
        json={
            "sender_id": "0xValidatorAlpha",
            "nonce": "NOT_AN_INTEGER_NONCE",  # Type violation
            "payload": {"data": 123},
        },
    )
    assert resp.status_code == 422
    assert resp.status_code != 500
    data = resp.json()
    assert data["error"] is True
    assert data["code"] == "MALFORMED_INPUT"


def test_negative_nonce_rejected_422(client: TestClient) -> None:
    """Verify ge=0 constraint on nonce."""
    resp = client.post(
        "/api/v1/replay/verify",
        json={
            "sender_id": "0xValidatorAlpha",
            "nonce": -5,  # Constraint violation
            "payload": {"data": 123},
        },
    )
    assert resp.status_code == 422
    assert resp.status_code != 500


def test_malformed_auth_header_handled(client: TestClient) -> None:
    """Verify that garbage Bearer tokens produce 401 without unhandled 500."""
    resp = client.post(
        "/api/v1/governance/evaluate",
        json={"task_id": "T-GOV-002", "candidate_id": "Prakash Yadav"},
        headers={"Authorization": "Bearer not.a.valid.jwt.token"},
    )
    assert resp.status_code == 401
    assert resp.status_code != 500
    data = resp.json()
    assert data["error"] is True


def test_empty_bearer_token_handled(client: TestClient) -> None:
    """Verify empty Authorization header."""
    resp = client.post(
        "/api/v1/governance/evaluate",
        json={"task_id": "T-GOV-002", "candidate_id": "Prakash Yadav"},
        headers={"Authorization": "Bearer "},
    )
    assert resp.status_code == 401
    assert resp.status_code != 500


def test_invalid_target_state_enum_handled(client: TestClient, admin_token: str) -> None:
    """Verify that an invalid state string returns 422 without server panic."""
    resp = client.post(
        "/api/v1/governance/transition",
        json={
            "target_state": "NON_EXISTENT_STATE",
            "actor": "admin",
            "reason": "testing",
        },
        headers={"Authorization": f"Bearer {admin_token}"},
    )
    assert resp.status_code == 422
    assert resp.status_code != 500
    data = resp.json()
    assert data["error"] is True
