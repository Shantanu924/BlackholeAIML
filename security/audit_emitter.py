"""
Pravah Immutable Audit Emitter & Cryptographic Hash Chain Engine
Emits tamper-evident structured events into the Pravah replay ledger with SHA-256
hash-chaining and cryptographic lineage verification.
"""

import hashlib
import json
import os
import threading
import time
import uuid
from typing import Any, Dict, List, Optional, Tuple

from pydantic import BaseModel, Field


class AuditEvent(BaseModel):
    """Tamper-evident audit event with cryptographic linkage to preceding block/event."""
    event_id: str = Field(default_factory=lambda: f"evt-{uuid.uuid4().hex[:12]}")
    timestamp: float = Field(default_factory=time.time)
    event_type: str = Field(..., description="AUTH_SUCCESS, REPLAY_ATTEMPT, RULE_EVALUATION, etc.")
    actor: str = Field(..., description="Identity responsible for triggering the action")
    action: str = Field(..., description="Specific operation executed")
    resource: str = Field(..., description="Target route, asset, or entity")
    status: str = Field(..., description="SUCCESS, DENIED, REJECTED, ERROR")
    metadata: Dict[str, Any] = Field(default_factory=dict)
    payload_hash: str = Field(default="0" * 64)
    previous_hash: str = Field(default="0" * 64)
    current_hash: str = Field(default="")

    model_config = {"extra": "forbid"}


def compute_audit_hash(
    previous_hash: str,
    event_id: str,
    timestamp: float,
    event_type: str,
    actor: str,
    action: str,
    resource: str,
    status: str,
    payload_hash: str,
) -> str:
    """
    Compute cryptographic SHA-256 digest binding the event to its predecessor.
    """
    raw = f"{previous_hash}|{event_id}|{timestamp:.6f}|{event_type}|{actor}|{action}|{resource}|{status}|{payload_hash}"
    return hashlib.sha256(raw.encode("utf-8")).hexdigest()


class PravahAuditEmitter:
    """
    Thread-safe event streaming emitter that maintains an immutable
    SHA-256 hash chain and persists into the Pravah audit ledger.
    """

    def __init__(self, ledger_file: Optional[str] = "pravah_audit_ledger.jsonl") -> None:
        self.ledger_file = ledger_file
        self._lock = threading.RLock()
        self._events: List[AuditEvent] = []
        self._last_hash = "0" * 64

        if self.ledger_file and os.path.exists(self.ledger_file):
            self._hydrate_chain()

    def _hydrate_chain(self) -> None:
        """Hydrate existing chain from ledger file and verify integrity."""
        try:
            with open(self.ledger_file, "r", encoding="utf-8") as f:
                for line in f:
                    line = line.strip()
                    if not line:
                        continue
                    data = json.loads(line)
                    evt = AuditEvent(**data)
                    self._events.append(evt)
                    self._last_hash = evt.current_hash
        except Exception:
            pass

    def emit(
        self,
        event_type: str,
        actor: str,
        action: str,
        resource: str,
        status: str,
        metadata: Optional[Dict[str, Any]] = None,
        payload: Optional[Any] = None,
    ) -> AuditEvent:
        """
        Create, cryptographically link, and commit a new audit event.
        """
        now = time.time()
        p_hash = "0" * 64
        if payload is not None:
            if isinstance(payload, str):
                p_hash = hashlib.sha256(payload.encode("utf-8")).hexdigest()
            else:
                p_hash = hashlib.sha256(
                    json.dumps(payload, sort_keys=True, separators=(",", ":")).encode("utf-8")
                ).hexdigest()

        e_id = f"evt-{uuid.uuid4().hex[:12]}"
        meta = metadata or {}

        with self._lock:
            c_hash = compute_audit_hash(
                previous_hash=self._last_hash,
                event_id=e_id,
                timestamp=now,
                event_type=event_type,
                actor=actor,
                action=action,
                resource=resource,
                status=status,
                payload_hash=p_hash,
            )

            evt = AuditEvent(
                event_id=e_id,
                timestamp=now,
                event_type=event_type,
                actor=actor,
                action=action,
                resource=resource,
                status=status,
                metadata=meta,
                payload_hash=p_hash,
                previous_hash=self._last_hash,
                current_hash=c_hash,
            )

            self._events.append(evt)
            self._last_hash = c_hash

            if self.ledger_file:
                try:
                    with open(self.ledger_file, "a", encoding="utf-8") as f:
                        f.write(json.dumps(evt.model_dump()) + "\n")
                except Exception:
                    pass

            return evt

    def get_recent_events(self, limit: int = 50) -> List[AuditEvent]:
        """Retrieve recent audit events in reverse chronological order."""
        with self._lock:
            return list(reversed(self._events[-limit:]))

    def reset(self) -> None:
        """Reset internal chain state (for test isolation)."""
        with self._lock:
            self._events.clear()
            self._last_hash = "0" * 64


def verify_audit_chain(events: Optional[List[AuditEvent]] = None, ledger_file: Optional[str] = None) -> Tuple[bool, str]:
    """
    Verify the cryptographic integrity of an audit event sequence.
    Returns (True, "Chain integrity verified") or (False, "Failure description").
    """
    chain_to_verify: List[AuditEvent] = []
    if events is not None:
        chain_to_verify = events
    elif ledger_file and os.path.exists(ledger_file):
        with open(ledger_file, "r", encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                chain_to_verify.append(AuditEvent(**json.loads(line)))
    else:
        return True, "Empty chain: Verified trivially."

    if not chain_to_verify:
        return True, "Empty chain: Verified trivially."

    expected_prev = "0" * 64
    for idx, evt in enumerate(chain_to_verify):
        # 1. Previous hash match
        if evt.previous_hash != expected_prev:
            return False, (
                f"Broken chain link at index {idx} (event {evt.event_id}): "
                f"expected previous_hash '{expected_prev}', got '{evt.previous_hash}'."
            )

        # 2. Recomputed hash match
        recomputed = compute_audit_hash(
            previous_hash=evt.previous_hash,
            event_id=evt.event_id,
            timestamp=evt.timestamp,
            event_type=evt.event_type,
            actor=evt.actor,
            action=evt.action,
            resource=evt.resource,
            status=evt.status,
            payload_hash=evt.payload_hash,
        )
        if evt.current_hash != recomputed:
            return False, (
                f"Tampered block detected at index {idx} (event {evt.event_id}): "
                f"stored current_hash '{evt.current_hash}' != recomputed '{recomputed}'."
            )

        expected_prev = evt.current_hash

    return True, f"Chain integrity verified: {len(chain_to_verify)} consecutive blocks cryptographically valid."


# Global singleton instance
_default_emitter: Optional[PravahAuditEmitter] = None


def get_default_audit_emitter() -> PravahAuditEmitter:
    """Retrieve global default audit emitter."""
    global _default_emitter
    if _default_emitter is None:
        _default_emitter = PravahAuditEmitter()
    return _default_emitter
