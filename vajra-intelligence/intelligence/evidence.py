"""
intelligence/evidence.py — Advisory Evidence Store (Stage 4).

SQLite-backed hash-chained evidence log maintained by the intelligence layer.

⚠️  This is NOT the VAJRA blockchain. VAJRA remains the single canonical
    on-chain evidence store. This module provides an independent, tamper-evident
    advisory audit trail for the intelligence layer's own records.

Design:
- Each record includes SHA-256(content + previous_hash) → forms a hash chain.
- Genesis record has previous_hash = '0' * 64.
- Verification walks the full chain and checks all hashes are consistent.
"""
from __future__ import annotations

import hashlib
import json
import sqlite3
import threading
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional

from config import settings
from intelligence.models import Decision, EvidenceRecord, IntelligenceEvent

# Thread-local SQLite connection (safe for multi-threaded FastAPI)
_local = threading.local()
_DB_LOCK = threading.Lock()


def _get_conn() -> sqlite3.Connection:
    """Return a thread-local SQLite connection, creating it if needed."""
    db_path = Path(settings.EVIDENCE_DB_PATH)
    db_path.parent.mkdir(parents=True, exist_ok=True)
    if not hasattr(_local, "conn") or _local.conn is None:
        _local.conn = sqlite3.connect(str(db_path), check_same_thread=False)
        _local.conn.row_factory = sqlite3.Row
    return _local.conn


def init_db() -> None:
    """Initialise the evidence table. Safe to call multiple times."""
    conn = _get_conn()
    conn.execute("""
        CREATE TABLE IF NOT EXISTS intelligence_evidence (
            block_id        INTEGER PRIMARY KEY AUTOINCREMENT,
            record_id       TEXT    NOT NULL UNIQUE,
            event_id        TEXT    NOT NULL UNIQUE,
            user_id         TEXT    NOT NULL,
            action          TEXT    NOT NULL,
            resource        TEXT    NOT NULL,
            timestamp       TEXT    NOT NULL,
            decision_outcome TEXT   NOT NULL,
            risk_level      TEXT    NOT NULL,
            tx_ref          TEXT    NOT NULL,
            current_hash    TEXT    NOT NULL,
            previous_hash   TEXT    NOT NULL,
            created_at      TEXT    NOT NULL
        )
    """)
    conn.commit()


def _compute_hash(
    record_id: str,
    event_id: str,
    user_id: str,
    action: str,
    resource: str,
    timestamp: str,
    decision_outcome: str,
    risk_level: str,
    previous_hash: str,
) -> str:
    """Compute SHA-256 over the record's canonical fields + previous_hash."""
    payload = json.dumps(
        {
            "record_id": record_id,
            "event_id": event_id,
            "user_id": user_id,
            "action": action,
            "resource": resource,
            "timestamp": timestamp,
            "decision_outcome": decision_outcome,
            "risk_level": risk_level,
            "previous_hash": previous_hash,
        },
        sort_keys=True,
    )
    return hashlib.sha256(payload.encode()).hexdigest()


def _get_last_hash(conn: sqlite3.Connection) -> str:
    """Return the hash of the last record, or genesis sentinel if empty."""
    row = conn.execute(
        "SELECT current_hash FROM intelligence_evidence ORDER BY block_id DESC LIMIT 1"
    ).fetchone()
    return row["current_hash"] if row else "0" * 64


def append(event: IntelligenceEvent, decision: Decision, risk_level: str) -> EvidenceRecord:
    """
    Append a new advisory evidence record to the chain.

    Args:
        event:      The IntelligenceEvent being recorded.
        decision:   The Decision Gate outcome.
        risk_level: Risk level string from the AssessmentResult.

    Returns:
        The newly created EvidenceRecord.
    """
    with _DB_LOCK:
        conn = _get_conn()
        init_db()

        record_id = str(uuid.uuid4())
        tx_ref = f"VI-{uuid.uuid4().hex[:12].upper()}"
        previous_hash = _get_last_hash(conn)

        current_hash = _compute_hash(
            record_id=record_id,
            event_id=event.event_id,
            user_id=event.user_id,
            action=event.action,
            resource=event.resource,
            timestamp=event.timestamp,
            decision_outcome=decision.outcome.value,
            risk_level=risk_level,
            previous_hash=previous_hash,
        )

        created_at = datetime.now(timezone.utc).isoformat()

        conn.execute(
            """
            INSERT INTO intelligence_evidence
                (record_id, event_id, user_id, action, resource, timestamp,
                 decision_outcome, risk_level, tx_ref, current_hash, previous_hash, created_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            """,
            (
                record_id,
                event.event_id,
                event.user_id,
                event.action,
                event.resource,
                event.timestamp,
                decision.outcome.value,
                risk_level,
                tx_ref,
                current_hash,
                previous_hash,
                created_at,
            ),
        )
        conn.commit()

        block_id: int = conn.execute(
            "SELECT block_id FROM intelligence_evidence WHERE record_id = ?", (record_id,)
        ).fetchone()["block_id"]

    return EvidenceRecord(
        record_id=record_id,
        event_id=event.event_id,
        user_id=event.user_id,
        action=event.action,
        resource=event.resource,
        timestamp=event.timestamp,
        decision_outcome=decision.outcome.value,
        risk_level=risk_level,
        block_id=block_id,
        tx_ref=tx_ref,
        current_hash=current_hash,
        previous_hash=previous_hash,
    )


def get_by_event_id(event_id: str) -> Optional[EvidenceRecord]:
    """Fetch an evidence record by event_id. Returns None if not found."""
    conn = _get_conn()
    init_db()
    row = conn.execute(
        "SELECT * FROM intelligence_evidence WHERE event_id = ?", (event_id,)
    ).fetchone()
    if row is None:
        return None
    return EvidenceRecord(
        record_id=row["record_id"],
        event_id=row["event_id"],
        user_id=row["user_id"],
        action=row["action"],
        resource=row["resource"],
        timestamp=row["timestamp"],
        decision_outcome=row["decision_outcome"],
        risk_level=row["risk_level"],
        block_id=row["block_id"],
        tx_ref=row["tx_ref"],
        current_hash=row["current_hash"],
        previous_hash=row["previous_hash"],
    )


def get_full_chain() -> list[EvidenceRecord]:
    """Return all evidence records ordered by block_id (oldest first)."""
    conn = _get_conn()
    init_db()
    rows = conn.execute(
        "SELECT * FROM intelligence_evidence ORDER BY block_id ASC"
    ).fetchall()
    return [
        EvidenceRecord(
            record_id=r["record_id"],
            event_id=r["event_id"],
            user_id=r["user_id"],
            action=r["action"],
            resource=r["resource"],
            timestamp=r["timestamp"],
            decision_outcome=r["decision_outcome"],
            risk_level=r["risk_level"],
            block_id=r["block_id"],
            tx_ref=r["tx_ref"],
            current_hash=r["current_hash"],
            previous_hash=r["previous_hash"],
        )
        for r in rows
    ]


def verify_chain() -> tuple[bool, str]:
    """
    Walk the full evidence chain and verify all hashes are consistent.

    Returns:
        (True, "Chain intact.") if all hashes verify.
        (False, <error message>) if any hash mismatch is found.
    """
    records = get_full_chain()
    if not records:
        return True, "Chain is empty — no records to verify."

    expected_previous = "0" * 64
    for rec in records:
        if rec.previous_hash != expected_previous:
            return False, (
                f"Chain broken at block_id={rec.block_id}: "
                f"previous_hash mismatch (expected={expected_previous[:12]}…, "
                f"got={rec.previous_hash[:12]}…)."
            )

        recomputed = _compute_hash(
            record_id=rec.record_id,
            event_id=rec.event_id,
            user_id=rec.user_id,
            action=rec.action,
            resource=rec.resource,
            timestamp=rec.timestamp,
            decision_outcome=rec.decision_outcome,
            risk_level=rec.risk_level,
            previous_hash=rec.previous_hash,
        )
        if recomputed != rec.current_hash:
            return False, (
                f"Tamper detected at block_id={rec.block_id}: "
                f"stored_hash={rec.current_hash[:12]}…, "
                f"recomputed={recomputed[:12]}…."
            )

        expected_previous = rec.current_hash

    return True, f"Chain intact — {len(records)} block(s) verified."
