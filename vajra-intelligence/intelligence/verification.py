"""
intelligence/verification.py — Hash-chain verification (Stage 5).

Re-fetches an advisory evidence record by event_id and verifies
that the full chain is intact (tamper detection).
"""
from __future__ import annotations

from datetime import datetime, timezone

from intelligence.evidence import get_by_event_id, verify_chain
from intelligence.models import VerificationResult


def verify_record(event_id: str) -> VerificationResult:
    """
    Re-fetch and verify the advisory evidence record for the given event_id.

    Steps:
        1. Look up the record by event_id.
        2. Walk the full chain and verify all SHA-256 hashes.

    Args:
        event_id: The UUID of the IntelligenceEvent to verify.

    Returns:
        VerificationResult with integrity_ok=True if the chain is intact.
    """
    entry = get_by_event_id(event_id)
    if entry is None:
        return VerificationResult(
            event_id=event_id,
            entry_found=False,
            integrity_ok=False,
            detail=f"No advisory evidence record found for event_id={event_id!r}.",
            verified_at=datetime.now(timezone.utc).isoformat(),
        )

    ok, detail = verify_chain()
    return VerificationResult(
        event_id=event_id,
        entry_found=True,
        integrity_ok=ok,
        detail=detail,
        verified_at=datetime.now(timezone.utc).isoformat(),
        evidence_record=entry,
    )
