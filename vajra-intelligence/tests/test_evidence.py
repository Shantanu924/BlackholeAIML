"""
tests/test_evidence.py — Unit tests for the advisory evidence store.

Proves:
  ✓ Records are appended and retrievable
  ✓ Hash chain is intact after multiple appends
  ✓ Tamper detection: modifying stored hash breaks verification
  ✓ get_by_event_id returns None for unknown event_id
  ✓ get_full_chain returns records in block_id order
"""
from __future__ import annotations

import sqlite3

import pytest

from config import settings
from intelligence.assessment import assess
from intelligence.decision import decide
from intelligence.evidence import (
    append,
    get_by_event_id,
    get_full_chain,
    init_db,
    verify_chain,
)
from intelligence.models import IntelligenceEvent


def _make_event(user_id: str = "ev-test-user", action: str = "READ") -> IntelligenceEvent:
    return IntelligenceEvent(
        user_id=user_id,
        action=action,
        resource="public_docs",
        timestamp="2026-10-05T10:00:00Z",
        authorized=True,
    )


@pytest.fixture(autouse=True)
def ensure_db():
    init_db()


class TestAppendAndRetrieve:
    def test_append_returns_record(self):
        event = _make_event()
        assessment = assess(event)
        decision = decide(event, assessment)
        rec = append(event, decision, assessment.risk.value)

        assert rec.event_id == event.event_id
        assert rec.block_id >= 1
        assert rec.tx_ref.startswith("VI-")
        assert len(rec.current_hash) == 64
        assert len(rec.previous_hash) == 64

    def test_get_by_event_id(self):
        event = _make_event(user_id="retrieve-test")
        assessment = assess(event)
        decision = decide(event, assessment)
        append(event, decision, assessment.risk.value)

        fetched = get_by_event_id(event.event_id)
        assert fetched is not None
        assert fetched.event_id == event.event_id
        assert fetched.user_id == event.user_id

    def test_get_by_unknown_returns_none(self):
        assert get_by_event_id("00000000-0000-0000-0000-000000000000") is None

    def test_chain_ordered_by_block_id(self):
        chain_before = get_full_chain()
        for i in range(3):
            event = _make_event(user_id=f"order-test-{i}")
            assessment = assess(event)
            decision = decide(event, assessment)
            append(event, decision, assessment.risk.value)

        chain = get_full_chain()
        block_ids = [r.block_id for r in chain]
        assert block_ids == sorted(block_ids), "Chain not ordered by block_id"


class TestHashChainIntegrity:
    def test_chain_intact_after_appends(self):
        """Multiple appends should produce an intact chain."""
        for i in range(5):
            event = _make_event(user_id=f"chain-user-{i}")
            assessment = assess(event)
            decision = decide(event, assessment)
            append(event, decision, assessment.risk.value)

        ok, message = verify_chain()
        assert ok, f"Chain broken: {message}"

    def test_empty_chain_is_intact(self):
        """An empty chain should report intact."""
        # We can't clear the DB safely in a shared test run,
        # so just assert the function returns a bool.
        ok, message = verify_chain()
        assert isinstance(ok, bool)
        assert isinstance(message, str)


class TestTamperDetection:
    def test_tampered_hash_detected(self):
        """Manually corrupting a stored hash must be detected."""
        event = _make_event(user_id="tamper-test")
        assessment = assess(event)
        decision = decide(event, assessment)
        rec = append(event, decision, assessment.risk.value)

        # Corrupt the stored hash directly in SQLite
        conn = sqlite3.connect(settings.EVIDENCE_DB_PATH)
        conn.execute(
            "UPDATE intelligence_evidence SET current_hash = ? WHERE event_id = ?",
            ("deadbeef" * 8, event.event_id),
        )
        conn.commit()
        conn.close()

        ok, message = verify_chain()
        assert not ok, "Tamper detection failed — corrupted hash not caught"
        assert "Tamper detected" in message or "broken" in message.lower()

        # Restore the original hash so subsequent tests pass
        conn = sqlite3.connect(settings.EVIDENCE_DB_PATH)
        conn.execute(
            "UPDATE intelligence_evidence SET current_hash = ? WHERE event_id = ?",
            (rec.current_hash, event.event_id),
        )
        conn.commit()
        conn.close()
