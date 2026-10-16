"""
Unit tests for RS256 Cryptographic Token Verification and Enterprise RBAC.
"""

import time
import pytest

from security.jwt_handler import JWTHandler, RS256SecurityEngine, TokenPayload


def test_rs256_token_issuance_and_verification(security_engine: RS256SecurityEngine) -> None:
    """Verify that an RS256 JWT is signed and verified with asymmetric RSA keys."""
    handler = JWTHandler(security_engine)
    token = handler.issue_token(
        sub="candidate-prakash",
        roles=["auditor", "validator"],
        permissions=["governance:evaluate", "replay:verify"],
        tenant_id="tantra-mainnet",
        candidate_id="Prakash Yadav",
        ttl_seconds=3600,
    )

    assert isinstance(token, str)
    assert len(token.split(".")) == 3

    payload = handler.verify_token(token)
    assert isinstance(payload, TokenPayload)
    assert payload.sub == "candidate-prakash"
    assert "auditor" in payload.roles
    assert "validator" in payload.roles
    assert "governance:evaluate" in payload.permissions
    assert payload.tenant_id == "tantra-mainnet"
    assert payload.candidate_id == "Prakash Yadav"


def test_rs256_tampered_payload_rejected(security_engine: RS256SecurityEngine) -> None:
    """Verify that any modification to payload bytes triggers cryptographic rejection."""
    import json
    from security.jwt_handler import _b64url_decode, _b64url_encode

    handler = JWTHandler(security_engine)
    token = handler.issue_token(sub="legitimate-user", roles=["operator"])

    header_b64, payload_b64, sig_b64 = token.split(".")
    claims = json.loads(_b64url_decode(payload_b64))
    claims["sub"] = "tampered-attacker"
    tampered_payload = _b64url_encode(json.dumps(claims).encode("utf-8"))
    tampered_token = f"{header_b64}.{tampered_payload}.{sig_b64}"

    with pytest.raises(ValueError, match="Cryptographic verification failed|Invalid RS256 signature"):
        handler.verify_token(tampered_token)


def test_rs256_tampered_signature_rejected(security_engine: RS256SecurityEngine) -> None:
    """Verify that an invalid or mutated signature is rejected."""
    handler = JWTHandler(security_engine)
    token = handler.issue_token(sub="test-user")
    header_b64, payload_b64, sig_b64 = token.split(".")
    tampered_sig = sig_b64[:-4] + "AAAA"
    tampered_token = f"{header_b64}.{payload_b64}.{tampered_sig}"

    with pytest.raises(ValueError, match="Cryptographic verification failed|Invalid RS256 signature"):
        handler.verify_token(tampered_token)


def test_rs256_expired_token_rejected(security_engine: RS256SecurityEngine) -> None:
    """Verify that an expired token raises ValueError."""
    claims = {
        "sub": "expired-user",
        "roles": ["operator"],
        "permissions": [],
        "exp": int(time.time()) - 100,  # Expired in past
        "iat": int(time.time()) - 200,
    }
    expired_token = security_engine.sign_jwt(claims)

    with pytest.raises(ValueError, match="Token has expired"):
        security_engine.verify_jwt(expired_token)


def test_rs256_future_nbf_token_rejected(security_engine: RS256SecurityEngine) -> None:
    """Verify that a token with future nbf (not-before) is rejected."""
    claims = {
        "sub": "future-user",
        "roles": ["operator"],
        "permissions": [],
        "exp": int(time.time()) + 3600,
        "nbf": int(time.time()) + 1000,  # Not valid yet
    }
    future_token = security_engine.sign_jwt(claims)

    with pytest.raises(ValueError, match="Token not valid before"):
        security_engine.verify_jwt(future_token)


def test_audience_mismatch_rejected(security_engine: RS256SecurityEngine) -> None:
    """Verify audience claim enforcement."""
    claims = {
        "sub": "user1",
        "aud": "wrong.audience",
        "exp": int(time.time()) + 3600,
    }
    token = security_engine.sign_jwt(claims)

    with pytest.raises(ValueError, match="Audience mismatch"):
        security_engine.verify_jwt(token, expected_audience="vajra.production.runtime")


def test_rbac_permissions_and_roles(security_engine: RS256SecurityEngine) -> None:
    """Verify RBAC role checks, permission checks, and admin override."""
    handler = JWTHandler(security_engine)

    # 1. Standard Operator
    op_payload = handler.verify_token(
        handler.issue_token(sub="op1", roles=["operator"], permissions=["read:status"])
    )
    ok, _ = handler.check_permissions(op_payload, required_roles=["auditor"])
    assert not ok, "Operator should not pass auditor check"

    ok, _ = handler.check_permissions(op_payload, required_roles=["operator"])
    assert ok, "Operator should pass operator check"

    ok, _ = handler.check_permissions(op_payload, required_permissions=["write:governance"])
    assert not ok, "Missing permission should fail"

    # 2. Administrator (Override All)
    admin_payload = handler.verify_token(
        handler.issue_token(sub="admin1", roles=["admin"])
    )
    ok, msg = handler.check_permissions(
        admin_payload,
        required_roles=["super-auditor"],
        required_permissions=["unknown:privileged:perm"],
    )
    assert ok, f"Admin override must bypass all checks: {msg}"


def test_jwks_and_public_key_export(security_engine: RS256SecurityEngine) -> None:
    """Verify public key PEM format and JWKS structure."""
    pem = security_engine.get_public_key_pem()
    assert "-----BEGIN PUBLIC KEY-----" in pem
    assert "-----END PUBLIC KEY-----" in pem

    jwks = security_engine.get_jwks()
    assert "keys" in jwks
    assert len(jwks["keys"]) == 1
    key = jwks["keys"][0]
    assert key["kty"] == "RSA"
    assert key["alg"] == "RS256"
    assert "n" in key and "e" in key
