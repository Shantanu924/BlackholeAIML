"""
tests/conftest.py — Shared pytest fixtures for the intelligence layer.
"""
from __future__ import annotations

import pytest
from fastapi.testclient import TestClient

from main import app
from intelligence.evidence import init_db


@pytest.fixture(scope="session", autouse=True)
def initialise_db():
    """Ensure evidence DB is initialised before any test runs."""
    init_db()


@pytest.fixture(scope="session")
def client() -> TestClient:
    """Reusable FastAPI test client."""
    return TestClient(app)


@pytest.fixture
def sample_event_payload() -> dict:
    """A valid low-risk event payload."""
    return {
        "user_id": "test-user-01",
        "action": "READ",
        "resource": "public_report",
        "timestamp": "2026-10-05T06:00:00Z",
        "authorized": True,
        "auth_reason": "Read scope granted.",
        "source_system": "VAJRA",
    }


@pytest.fixture
def high_risk_payload() -> dict:
    """A high-risk event payload (sensitive resource + high-value action + unauthorized)."""
    return {
        "user_id": "attacker-99",
        "action": "DELETE",
        "resource": "personal_data",
        "timestamp": "2026-10-05T02:00:00Z",   # off-hours UTC
        "authorized": False,
        "auth_reason": "User not found.",
        "source_system": "EXTERNAL",
    }
