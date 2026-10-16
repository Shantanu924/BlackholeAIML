"""
Unit tests for Pravah Immutable Audit Emitter and Cryptographic Hash Chain.
"""

import copy
from security.audit_emitter import AuditEvent, PravahAuditEmitter, verify_audit_chain


def test_audit_event_emission_and_chaining(clean_audit_emitter: PravahAuditEmitter) -> None:
    """Verify sequential SHA-256 hash chaining of audit events."""
    evt1 = clean_audit_emitter.emit(
        event_type="AUTH_SUCCESS",
        actor="Prakash Yadav",
        action="LOGIN_PRAVESH",
        resource="auth/sso",
        status="SUCCESS",
        metadata={"tenant": "tantra"},
    )
    assert evt1.previous_hash == "0" * 64
    assert len(evt1.current_hash) == 64

    evt2 = clean_audit_emitter.emit(
        event_type="RULE_EVALUATION",
        actor="RuleEngine",
        action="EVALUATE_RBAC",
        resource="governance/t-gov-002",
        status="SUCCESS",
        metadata={"rule": "RULE-RBAC-01"},
    )
    assert evt2.previous_hash == evt1.current_hash
    assert len(evt2.current_hash) == 64

    evt3 = clean_audit_emitter.emit(
        event_type="STATE_TRANSITION",
        actor="NIYANTRAN",
        action="TRANSITION_SUBMITTED",
        resource="state/t-gov-002",
        status="SUCCESS",
    )
    assert evt3.previous_hash == evt2.current_hash

    # Verify overall chain integrity
    valid, message = verify_audit_chain(events=clean_audit_emitter._events)
    assert valid is True
    assert "3 consecutive blocks cryptographically valid" in message


def test_audit_chain_tamper_detection(clean_audit_emitter: PravahAuditEmitter) -> None:
    """Verify that tampering with any audit event's content breaks chain verification."""
    clean_audit_emitter.emit(
        event_type="AUTH_SUCCESS", actor="Alice", action="READ", resource="res1", status="SUCCESS"
    )
    clean_audit_emitter.emit(
        event_type="AUTH_SUCCESS", actor="Bob", action="WRITE", resource="res2", status="SUCCESS"
    )
    clean_audit_emitter.emit(
        event_type="AUTH_SUCCESS", actor="Charlie", action="ADMIN", resource="res3", status="SUCCESS"
    )

    # 1. Verify pristine chain
    ok, _ = verify_audit_chain(events=clean_audit_emitter._events)
    assert ok is True

    # 2. Tamper with event 1 actor
    tampered_events = [copy.deepcopy(e) for e in clean_audit_emitter._events]
    tampered_events[1].actor = "Mallory"  # Malicious replacement

    tampered_ok, tampered_msg = verify_audit_chain(events=tampered_events)
    assert tampered_ok is False
    assert "Tampered block detected at index 1" in tampered_msg

    # 3. Tamper with previous_hash link
    tampered_events2 = [copy.deepcopy(e) for e in clean_audit_emitter._events]
    tampered_events2[2].previous_hash = "f" * 64

    link_ok, link_msg = verify_audit_chain(events=tampered_events2)
    assert link_ok is False
    assert "Broken chain link at index 2" in link_msg


def test_empty_audit_chain_verification() -> None:
    """Verify that an empty chain verifies trivially without exceptions."""
    ok, msg = verify_audit_chain(events=[])
    assert ok is True
    assert "Verified trivially" in msg
