"""
Pravah Replay Mitigation Table & Replay Proof Engine
Implements concurrency-controlled nonce tracking, payload SHA-256 fingerprinting,
sliding-window deduplication, and cryptographic proof generation.
"""

import hashlib
import json
import os
import threading
import time
import uuid
from typing import Any, Dict, List, Optional, Tuple

from pydantic import BaseModel, Field


class ReplayRecord(BaseModel):
    """Immutable record stored in the replay mitigation table."""
    sender_id: str
    nonce: int
    payload_hash: str
    trace_id: str
    timestamp: float
    status: str = "ACCEPTED"

    model_config = {"extra": "forbid"}


class ReplayProof(BaseModel):
    """Cryptographically verifiable proof for replay defense validation."""
    proof_id: str = Field(default_factory=lambda: f"proof-{uuid.uuid4().hex[:12]}")
    trace_id: str
    sender_id: str
    nonce: int
    payload_hash: str
    timestamp: float
    status: str  # "ACCEPTED" or "DUPLICATE_REJECTED"
    reason: str
    proof_digest: str

    model_config = {"extra": "forbid"}


def compute_payload_hash(payload: Any) -> str:
    """
    Produce a canonical, deterministic SHA-256 digest of any payload.
    Sorts dictionary keys recursively to guarantee determinism.
    """
    if isinstance(payload, bytes):
        raw = payload
    elif isinstance(payload, str):
        raw = payload.encode("utf-8")
    else:
        raw = json.dumps(payload, sort_keys=True, separators=(",", ":")).encode("utf-8")
    return hashlib.sha256(raw).hexdigest()


class ReplayMitigationTable:
    """
    Thread-safe, sliding-window replay mitigation engine with persistent
    Pravah replay ledger synchronization.
    """

    def __init__(
        self,
        ttl_seconds: float = 300.0,
        ledger_path: Optional[str] = "pravah_replay_ledger.jsonl",
    ) -> None:
        self.ttl_seconds = ttl_seconds
        self.ledger_path = ledger_path
        self._lock = threading.RLock()

        # sender_id -> highest monotonic nonce seen
        self._sender_highest_nonce: Dict[str, int] = {}
        # (sender_id, nonce) -> ReplayRecord
        self._seen_nonces: Dict[Tuple[str, int], ReplayRecord] = {}
        # payload_hash -> ReplayRecord
        self._seen_payloads: Dict[str, ReplayRecord] = {}
        # jti -> timestamp
        self._seen_jtis: Dict[str, float] = {}

        if self.ledger_path and os.path.exists(self.ledger_path):
            self._recover_from_ledger()

    def _recover_from_ledger(self) -> None:
        """Hydrate in-memory tables from persistent replay ledger."""
        try:
            with open(self.ledger_path, "r", encoding="utf-8") as f:
                for line in f:
                    line = line.strip()
                    if not line:
                        continue
                    data = json.loads(line)
                    record = ReplayRecord(
                        sender_id=data["sender_id"],
                        nonce=data["nonce"],
                        payload_hash=data["payload_hash"],
                        trace_id=data.get("trace_id", "recovered"),
                        timestamp=data["timestamp"],
                        status=data.get("status", "ACCEPTED"),
                    )
                    self._seen_nonces[(record.sender_id, record.nonce)] = record
                    self._seen_payloads[record.payload_hash] = record
                    if record.nonce > self._sender_highest_nonce.get(record.sender_id, -1):
                        self._sender_highest_nonce[record.sender_id] = record.nonce
        except Exception:
            pass  # Fallback to fresh state if recovery encounters corruption

    def _purge_expired(self, now: float) -> None:
        """Purge entries outside the sliding TTL window."""
        cutoff = now - self.ttl_seconds
        # Clean expired payloads
        expired_hashes = [h for h, rec in self._seen_payloads.items() if rec.timestamp < cutoff]
        for h in expired_hashes:
            del self._seen_payloads[h]

        # Clean expired JTIs
        expired_jtis = [j for j, t in self._seen_jtis.items() if t < cutoff]
        for j in expired_jtis:
            del self._seen_jtis[j]

    def verify_and_register(
        self,
        sender_id: str,
        nonce: int,
        payload: Any,
        trace_id: Optional[str] = None,
        jti: Optional[str] = None,
    ) -> Tuple[bool, ReplayProof]:
        """
        Verify payload uniqueness and monotonic nonce validity.
        Returns (is_valid, ReplayProof).
        """
        now = time.time()
        tid = trace_id or f"trace-{uuid.uuid4().hex[:12]}"
        p_hash = compute_payload_hash(payload)

        with self._lock:
            self._purge_expired(now)

            # 1. JTI Check (if provided)
            if jti:
                if jti in self._seen_jtis:
                    proof_digest = hashlib.sha256(f"REPLAY:{jti}:{p_hash}".encode("utf-8")).hexdigest()
                    return False, ReplayProof(
                        trace_id=tid,
                        sender_id=sender_id,
                        nonce=nonce,
                        payload_hash=p_hash,
                        timestamp=now,
                        status="DUPLICATE_REJECTED",
                        reason=f"JTI token '{jti}' has already been processed (replay detected).",
                        proof_digest=proof_digest,
                    )
                self._seen_jtis[jti] = now

            # 2. Monotonic Nonce & Duplicate Nonce Check
            last_nonce = self._sender_highest_nonce.get(sender_id, -1)
            if (sender_id, nonce) in self._seen_nonces or nonce <= last_nonce:
                proof_digest = hashlib.sha256(f"REPLAY:{sender_id}:{nonce}:{p_hash}".encode("utf-8")).hexdigest()
                return False, ReplayProof(
                    trace_id=tid,
                    sender_id=sender_id,
                    nonce=nonce,
                    payload_hash=p_hash,
                    timestamp=now,
                    status="DUPLICATE_REJECTED",
                    reason=f"Nonce {nonce} is invalid for sender '{sender_id}'. Current height: {last_nonce}.",
                    proof_digest=proof_digest,
                )

            # 3. Payload Hash Duplicate Check
            if p_hash in self._seen_payloads:
                existing = self._seen_payloads[p_hash]
                proof_digest = hashlib.sha256(f"REPLAY:PAYLOAD:{p_hash}:{existing.timestamp}".encode("utf-8")).hexdigest()
                return False, ReplayProof(
                    trace_id=tid,
                    sender_id=sender_id,
                    nonce=nonce,
                    payload_hash=p_hash,
                    timestamp=now,
                    status="DUPLICATE_REJECTED",
                    reason=f"Identical payload hash '{p_hash[:16]}...' previously recorded at timestamp {existing.timestamp}.",
                    proof_digest=proof_digest,
                )

            # 4. Valid new transaction: Commit record
            record = ReplayRecord(
                sender_id=sender_id,
                nonce=nonce,
                payload_hash=p_hash,
                trace_id=tid,
                timestamp=now,
                status="ACCEPTED",
            )
            self._seen_nonces[(sender_id, nonce)] = record
            self._seen_payloads[p_hash] = record
            self._sender_highest_nonce[sender_id] = nonce

            # Append to persistent Pravah replay ledger
            if self.ledger_path:
                try:
                    with open(self.ledger_path, "a", encoding="utf-8") as f:
                        f.write(json.dumps(record.model_dump()) + "\n")
                except Exception:
                    pass

            proof_digest = hashlib.sha256(f"PROOF:VALID:{sender_id}:{nonce}:{p_hash}:{tid}".encode("utf-8")).hexdigest()
            return True, ReplayProof(
                trace_id=tid,
                sender_id=sender_id,
                nonce=nonce,
                payload_hash=p_hash,
                timestamp=now,
                status="ACCEPTED",
                reason="Payload and nonce verified unique. Registered in Pravah replay ledger.",
                proof_digest=proof_digest,
            )

    def get_ledger_records(self, limit: int = 100) -> List[ReplayRecord]:
        """Retrieve latest registered records."""
        with self._lock:
            records = list(self._seen_payloads.values())
            records.sort(key=lambda r: r.timestamp, reverse=True)
            return records[:limit]

    def reset(self) -> None:
        """Reset in-memory state (used in isolated test setups)."""
        with self._lock:
            self._sender_highest_nonce.clear()
            self._seen_nonces.clear()
            self._seen_payloads.clear()
            self._seen_jtis.clear()


# Default singleton instance
_default_replay_table: Optional[ReplayMitigationTable] = None


def get_default_replay_table() -> ReplayMitigationTable:
    """Retrieve global default replay table."""
    global _default_replay_table
    if _default_replay_table is None:
        _default_replay_table = ReplayMitigationTable()
    return _default_replay_table
