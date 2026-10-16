"""
Pytest configuration and test fixtures for TANTRA / VAJRA Governance testing.
"""

import os
import tempfile
import pytest
from starlette.testclient import TestClient

from api.production import app
from security.audit_emitter import PravahAuditEmitter
from security.jwt_handler import JWTHandler, RS256SecurityEngine, get_default_security_engine
from security.replay_mitigation import ReplayMitigationTable


@pytest.fixture(scope="session")
def security_engine() -> RS256SecurityEngine:
    """Session-wide RS256 security engine sharing keys with production app."""
    return get_default_security_engine()


@pytest.fixture(scope="session")
def jwt_handler(security_engine: RS256SecurityEngine) -> JWTHandler:
    """JWT Handler backed by session RS256 engine."""
    return JWTHandler(engine=security_engine)


@pytest.fixture
def clean_replay_table() -> ReplayMitigationTable:
    """Isolated replay mitigation table using a temporary ledger file."""
    with tempfile.NamedTemporaryFile(suffix=".jsonl", delete=False) as tmp:
        tmp_path = tmp.name
    table = ReplayMitigationTable(ttl_seconds=300.0, ledger_path=tmp_path)
    yield table
    if os.path.exists(tmp_path):
        try:
            os.remove(tmp_path)
        except Exception:
            pass


@pytest.fixture
def clean_audit_emitter() -> PravahAuditEmitter:
    """Isolated Pravah audit emitter using a temporary ledger file."""
    with tempfile.NamedTemporaryFile(suffix=".jsonl", delete=False) as tmp:
        tmp_path = tmp.name
    emitter = PravahAuditEmitter(ledger_file=tmp_path)
    yield emitter
    if os.path.exists(tmp_path):
        try:
            os.remove(tmp_path)
        except Exception:
            pass


@pytest.fixture
def admin_token(jwt_handler: JWTHandler) -> str:
    """Valid RS256 token for an administrator."""
    return jwt_handler.issue_token(
        sub="admin-prakash",
        roles=["admin", "auditor"],
        permissions=["governance:evaluate", "governance:write", "replay:verify"],
        candidate_id="Prakash Yadav",
    )


@pytest.fixture
def auditor_token(jwt_handler: JWTHandler) -> str:
    """Valid RS256 token for an auditor."""
    return jwt_handler.issue_token(
        sub="auditor-niyantran",
        roles=["auditor"],
        permissions=["governance:evaluate", "replay:verify"],
        candidate_id="Prakash Yadav",
    )


@pytest.fixture
def operator_token(jwt_handler: JWTHandler) -> str:
    """Valid RS256 token for an operator."""
    return jwt_handler.issue_token(
        sub="operator-node1",
        roles=["operator"],
        permissions=["replay:verify"],
        candidate_id="Prakash Yadav",
    )


@pytest.fixture
def client() -> TestClient:
    """Synchronous test client for FastAPI production app."""
    return TestClient(app, raise_server_exceptions=False)
