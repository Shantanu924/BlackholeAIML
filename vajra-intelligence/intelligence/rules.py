"""
intelligence/rules.py — Deterministic rule-based risk scoring engine.

Ported and extended from aiml/prototype/src/aiml_assessment/rules.py.
All rules are transparent, logged in the 'features' dict, and fully reproducible.

Score → Risk mapping:
  0–1   → LOW      (CLEAR,   ALLOW,  confidence ~0.90)
  2–3   → MEDIUM   (FLAGGED, REVIEW, confidence ~0.75)
  4–5   → HIGH     (BLOCKED, DENY,   confidence ~0.85)
  6+    → CRITICAL (BLOCKED, DENY,   confidence ~0.95)

Rules:
  1. Unknown / unrecognised user_id         → +3
  2. VAJRA authorization denied             → +3
  3. High-value action (DELETE/EXPORT/
     TRANSFER/STAKE/BRIDGE/VOTE/UNSTAKE)    → +2
  4. Sensitive resource targeted            → +2
  5. Off-hours access (UTC 00–06)           → +1
  6. Frequency abuse (>N events / minute)   → +2
  7. Cross-chain bridge activity            → +1  [VAJRA extension]
  8. Validator / staking operation          → +1  [VAJRA extension]
"""
from __future__ import annotations

from datetime import datetime, timezone
from typing import TYPE_CHECKING

from config import (
    HIGH_VALUE_ACTIONS,
    SENSITIVE_RESOURCES,
    settings,
)

if TYPE_CHECKING:
    from intelligence.models import IntelligenceEvent

# In-memory per-user sliding-window frequency tracker
# {user_id: [datetime, ...]}  — pruned to a 60-second window
_frequency_tracker: dict[str, list[datetime]] = {}
_WINDOW_SECONDS = 60


# ── Helper ───────────────────────────────────────────────────────────────────

def _check_frequency(user_id: str, now: datetime) -> int:
    """Return +2 if user has exceeded FREQ_ABUSE_THRESHOLD events in the last minute."""
    events = _frequency_tracker.setdefault(user_id, [])
    events[:] = [t for t in events if (now - t).total_seconds() <= _WINDOW_SECONDS]
    events.append(now)
    return 2 if len(events) > settings.FREQ_ABUSE_THRESHOLD else 0


def _parse_timestamp(ts: str) -> datetime:
    """Parse ISO 8601 timestamp; fall back to now() if malformed."""
    try:
        return datetime.fromisoformat(ts.replace("Z", "+00:00"))
    except (ValueError, AttributeError):
        return datetime.now(timezone.utc)


# ── Core scoring ─────────────────────────────────────────────────────────────

def score_event(event: "IntelligenceEvent") -> tuple[int, dict]:
    """
    Score an IntelligenceEvent using all rules.

    Returns:
        (total_score, features_dict)

    The features dict is the full feature trace — every signal is logged
    so the result is auditable, explainable, and replayable.
    """
    features: dict = {}
    score = 0
    ts = _parse_timestamp(event.timestamp)

    # ── Rule 1: Unknown user ──────────────────────────────────────────────────
    # Intelligence layer has no user-store of its own; relies on VAJRA auth result.
    # A user that VAJRA says is unknown (not in auth_reason) scores 0 here —
    # Rule 2 (unauthorized) covers that case.
    # We keep this slot for future integration with VAJRA identity adapter.
    features["unknown_user"] = False  # NOT IMPLEMENTED — will flip via identity adapter
    # score += 0 (placeholder)

    # ── Rule 2: Unauthorized attempt ─────────────────────────────────────────
    unauthorized_score = 3 if not event.authorized else 0
    features["unauthorized_attempt"] = not event.authorized
    score += unauthorized_score

    # ── Rule 3: High-value action ─────────────────────────────────────────────
    action_upper = event.action.upper()
    is_high_value = action_upper in HIGH_VALUE_ACTIONS
    action_score = 2 if is_high_value else 0
    features["high_value_action"] = is_high_value
    features["action"] = action_upper
    score += action_score

    # ── Rule 4: Sensitive resource ────────────────────────────────────────────
    is_sensitive = event.resource in SENSITIVE_RESOURCES
    sensitive_score = 2 if is_sensitive else 0
    features["sensitive_resource"] = is_sensitive
    features["resource"] = event.resource
    score += sensitive_score

    # ── Rule 5: Off-hours access ──────────────────────────────────────────────
    hour_utc = ts.hour
    off_hours = settings.OFF_HOURS_START <= hour_utc < settings.OFF_HOURS_END
    off_hours_score = 1 if off_hours else 0
    features["off_hours_access"] = off_hours
    features["hour_utc"] = hour_utc
    score += off_hours_score

    # ── Rule 6: Frequency abuse ───────────────────────────────────────────────
    freq_score = _check_frequency(event.user_id, ts)
    features["frequency_abuse"] = freq_score > 0
    score += freq_score

    # ── Rule 7: Cross-chain bridge activity [VAJRA extension] ─────────────────
    is_bridge = (
        event.source_system.value == "BRIDGE"
        or action_upper == "BRIDGE"
        or "bridge" in event.resource.lower()
    )
    bridge_score = 1 if is_bridge else 0
    features["bridge_activity"] = is_bridge
    score += bridge_score

    # ── Rule 8: Validator / staking operation [VAJRA extension] ───────────────
    is_validator_op = action_upper in {"STAKE", "UNSTAKE", "VOTE"} or "validator" in event.resource.lower()
    validator_score = 1 if is_validator_op else 0
    features["validator_operation"] = is_validator_op
    score += validator_score

    features["total_score"] = score
    return score, features


# ── Mapping ───────────────────────────────────────────────────────────────────

def map_score_to_result(score: int) -> tuple[str, str, str, float]:
    """Map total score → (risk, status, recommended_action, confidence)."""
    if score >= settings.RISK_SCORE_CRITICAL:
        return "CRITICAL", "BLOCKED", "DENY", 0.95
    elif score >= settings.RISK_SCORE_HIGH:
        return "HIGH", "BLOCKED", "DENY", 0.85
    elif score >= settings.RISK_SCORE_MEDIUM:
        return "MEDIUM", "FLAGGED", "REVIEW", 0.75
    else:
        return "LOW", "CLEAR", "ALLOW", 0.90


def build_reason(features: dict, risk: str) -> str:
    """Build a human-readable risk explanation from the feature trace."""
    parts: list[str] = []

    if features.get("unauthorized_attempt"):
        parts.append("VAJRA authorization was denied")
    if features.get("high_value_action"):
        parts.append(f"action '{features.get('action')}' is high-value")
    if features.get("sensitive_resource"):
        parts.append(f"resource '{features.get('resource')}' is sensitive")
    if features.get("off_hours_access"):
        parts.append(f"off-hours access (UTC hour {features.get('hour_utc')})")
    if features.get("frequency_abuse"):
        parts.append("frequency abuse — too many events in 60 s")
    if features.get("bridge_activity"):
        parts.append("cross-chain bridge activity detected")
    if features.get("validator_operation"):
        parts.append("validator/staking operation detected")

    if not parts:
        parts.append("no elevated risk signals detected")

    return f"Risk={risk}. Signals: {'; '.join(parts)}."
