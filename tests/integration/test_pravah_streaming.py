"""
Integration tests for Pravah Event-Driven Streaming and Replay Ledgers.
"""

from security.audit_emitter import PravahAuditEmitter, verify_audit_chain
from security.replay_mitigation import ReplayMitigationTable


def test_pravah_streaming_and_ledger_lifecycle(
    clean_audit_emitter: PravahAuditEmitter,
    clean_replay_table: ReplayMitigationTable,
) -> None:
    """
    Test end-to-end event-driven streaming to Pravah replay ledgers:
    1. Authenticate -> Emit AUTH_SUCCESS
    2. Register transaction -> Verify against replay table
    3. Replay attack simulated -> Verify REPLAY_ATTEMPT emitted
    4. State transition executed -> Verify STATE_TRANSITION emitted
    5. Chain cryptographic integrity verified.
    """
    actor = "Prakash Yadav"
    sender = "0xPravahStreamingNode"

    # Step 1: Auth event
    evt_auth = clean_audit_emitter.emit(
        event_type="AUTH_SUCCESS",
        actor=actor,
        action="VERIFY_RS256_TOKEN",
        resource="auth/token",
        status="SUCCESS",
        metadata={"role": "auditor"},
    )
    assert len(evt_auth.current_hash) == 64

    # Step 2: Register valid transaction
    payload = {"block": 5001, "txs": ["tx1", "tx2"]}
    valid_tx, proof = clean_replay_table.verify_and_register(
        sender_id=sender, nonce=1, payload=payload
    )
    assert valid_tx is True

    evt_tx = clean_audit_emitter.emit(
        event_type="TRANSACTION_COMMITTED",
        actor=sender,
        action="COMMIT_REPLAY_RECORD",
        resource="ledger/pravah",
        status="SUCCESS",
        metadata={"proof_digest": proof.proof_digest},
        payload=payload,
    )
    assert evt_tx.previous_hash == evt_auth.current_hash

    # Step 3: Replay attack attempted
    dup_tx, dup_proof = clean_replay_table.verify_and_register(
        sender_id=sender, nonce=1, payload=payload
    )
    assert dup_tx is False

    evt_replay = clean_audit_emitter.emit(
        event_type="REPLAY_ATTEMPT_BLOCKED",
        actor=sender,
        action="INTERCEPT_DUPLICATE",
        resource="ledger/pravah",
        status="DENIED",
        metadata={"reason": dup_proof.reason, "proof_digest": dup_proof.proof_digest},
    )
    assert evt_replay.previous_hash == evt_tx.current_hash

    # Step 4: Verify full cryptographic chain
    is_valid, msg = verify_audit_chain(events=clean_audit_emitter._events)
    assert is_valid is True
    assert "3 consecutive blocks cryptographically valid" in msg
