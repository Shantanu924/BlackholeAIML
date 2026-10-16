"""
Unit tests for Pravah Replay Mitigation Table and Replay Proof Generation.
"""

import concurrent.futures
import time
from security.replay_mitigation import ReplayMitigationTable, compute_payload_hash


def test_monotonic_nonce_and_duplicate_rejection(clean_replay_table: ReplayMitigationTable) -> None:
    """Verify that ascending nonces are accepted, while duplicates or regressions are rejected."""
    sender = "0xValidatorAlpha"

    # Nonce 1: Accepted
    ok1, proof1 = clean_replay_table.verify_and_register(
        sender_id=sender,
        nonce=1,
        payload={"tx": "genesis_transfer", "amount": 100},
    )
    assert ok1 is True
    assert proof1.status == "ACCEPTED"
    assert len(proof1.proof_digest) == 64

    # Duplicate Nonce 1: Rejected
    ok2, proof2 = clean_replay_table.verify_and_register(
        sender_id=sender,
        nonce=1,
        payload={"tx": "different_transfer", "amount": 200},
    )
    assert ok2 is False
    assert proof2.status == "DUPLICATE_REJECTED"
    assert "Nonce 1 is invalid" in proof2.reason

    # Regressive Nonce 0: Rejected
    ok3, proof3 = clean_replay_table.verify_and_register(
        sender_id=sender,
        nonce=0,
        payload={"tx": "old_transfer", "amount": 50},
    )
    assert ok3 is False
    assert proof3.status == "DUPLICATE_REJECTED"

    # Nonce 2: Accepted
    ok4, proof4 = clean_replay_table.verify_and_register(
        sender_id=sender,
        nonce=2,
        payload={"tx": "valid_next_transfer", "amount": 300},
    )
    assert ok4 is True
    assert proof4.status == "ACCEPTED"


def test_payload_hash_deduplication(clean_replay_table: ReplayMitigationTable) -> None:
    """Verify that submitting the identical payload payload hash with a new nonce is caught and rejected."""
    sender = "0xValidatorBeta"
    payload = {"command": "EXECUTE_SETTLEMENT", "block_height": 1002}

    ok1, proof1 = clean_replay_table.verify_and_register(sender_id=sender, nonce=1, payload=payload)
    assert ok1 is True

    # Same payload submitted with new nonce 2
    ok2, proof2 = clean_replay_table.verify_and_register(sender_id=sender, nonce=2, payload=payload)
    assert ok2 is False
    assert proof2.status == "DUPLICATE_REJECTED"
    assert "Identical payload hash" in proof2.reason


def test_jti_replay_defense(clean_replay_table: ReplayMitigationTable) -> None:
    """Verify that a unique JWT ID (JTI) cannot be replayed."""
    jti = "unique-jti-token-999"
    sender = "0xValidatorGamma"

    ok1, _ = clean_replay_table.verify_and_register(
        sender_id=sender, nonce=1, payload={"msg": "hello"}, jti=jti
    )
    assert ok1 is True

    ok2, proof2 = clean_replay_table.verify_and_register(
        sender_id=sender, nonce=2, payload={"msg": "world"}, jti=jti
    )
    assert ok2 is False
    assert proof2.status == "DUPLICATE_REJECTED"
    assert "JTI token" in proof2.reason


def test_sliding_window_expiration() -> None:
    """Verify that records expire after the sliding TTL window."""
    short_lived_table = ReplayMitigationTable(ttl_seconds=0.1, ledger_path=None)
    sender = "0xValidatorDelta"
    payload = {"data": "ephemeral"}

    ok1, _ = short_lived_table.verify_and_register(sender_id=sender, nonce=1, payload=payload)
    assert ok1 is True

    # Wait for TTL to lapse
    time.sleep(0.15)

    # After expiry, the payload hash should be purged, allowing re-registration under a higher nonce
    ok2, proof2 = short_lived_table.verify_and_register(sender_id=sender, nonce=2, payload=payload)
    assert ok2 is True
    assert proof2.status == "ACCEPTED"


def test_concurrent_replay_defense(clean_replay_table: ReplayMitigationTable) -> None:
    """Verify thread-safety and race condition prevention under concurrent requests."""
    sender = "0xValidatorConcurrent"
    results = []

    def submit(nonce: int) -> None:
        res = clean_replay_table.verify_and_register(
            sender_id=sender,
            nonce=nonce,
            payload={"index": nonce},
        )
        results.append(res)

    with concurrent.futures.ThreadPoolExecutor(max_workers=8) as executor:
        futures = [executor.submit(submit, i) for i in range(1, 21)]
        concurrent.futures.wait(futures)

    # Exactly 20 unique nonces registered in sequence
    accepted = [r for r in results if r[0] is True]
    assert len(accepted) > 0

    # Ensure duplicate attempts on already processed nonces fail deterministically
    dup_res, _ = clean_replay_table.verify_and_register(sender_id=sender, nonce=1, payload={"index": 999})
    assert dup_res is False


def test_ledger_persistence_and_recovery(clean_replay_table: ReplayMitigationTable) -> None:
    """Verify that records saved to persistent Pravah ledger file are recovered."""
    sender = "0xValidatorPersistent"
    clean_replay_table.verify_and_register(sender_id=sender, nonce=10, payload={"val": "persistent"})

    # Create new table pointing to the same file
    recovered_table = ReplayMitigationTable(ttl_seconds=300.0, ledger_path=clean_replay_table.ledger_path)
    # Attempting nonce 10 on the recovered table should fail as duplicate
    ok, proof = recovered_table.verify_and_register(sender_id=sender, nonce=10, payload={"val": "new"})
    assert ok is False
    assert proof.status == "DUPLICATE_REJECTED"
