"""
Deterministic State Machine & Concurrency Control Engine
Enforces strict state transitions, invariant guards, and guaranteed deterministic
SHA-256 state hashing across 50 consecutive runs.
"""

import hashlib
import json
import threading
from enum import Enum
from typing import Any, Dict, List, Optional, Set, Tuple


class State(str, Enum):
    DRAFT = "DRAFT"
    SUBMITTED = "SUBMITTED"
    VALIDATING = "VALIDATING"
    RULE_EVALUATION = "RULE_EVALUATION"
    AUDIT_EMITTED = "AUDIT_EMITTED"
    FINALIZED = "FINALIZED"
    REJECTED = "REJECTED"


class StateTransitionError(Exception):
    """Raised when an illegal or guard-violating transition is attempted."""
    pass


ALLOWED_TRANSITIONS: Dict[State, Set[State]] = {
    State.DRAFT: {State.SUBMITTED},
    State.SUBMITTED: {State.VALIDATING, State.REJECTED},
    State.VALIDATING: {State.RULE_EVALUATION, State.REJECTED},
    State.RULE_EVALUATION: {State.AUDIT_EMITTED, State.REJECTED},
    State.AUDIT_EMITTED: {State.FINALIZED, State.REJECTED},
    State.FINALIZED: set(),  # Terminal state
    State.REJECTED: set(),   # Terminal state
}


class DeterministicStateMachine:
    """
    Concurrency-controlled state machine ensuring deterministic transitions
    and verifiable SHA-256 state hashing.
    """

    def __init__(
        self,
        task_id: str = "T-GOV-002",
        candidate_id: str = "Prakash Yadav",
        version: str = "2.0.0",
    ) -> None:
        self.task_id = task_id
        self.candidate_id = candidate_id
        self.version = version
        self.current_state = State.DRAFT
        self._lock = threading.RLock()
        self._step_counter = 0
        self._history: List[Dict[str, Any]] = []

    def transition_to(
        self,
        target_state: State,
        actor: str,
        reason: str = "",
        metadata: Optional[Dict[str, Any]] = None,
    ) -> str:
        """
        Execute an atomic state transition.
        Returns the SHA-256 hash of the transition step.
        """
        with self._lock:
            allowed = ALLOWED_TRANSITIONS.get(self.current_state, set())
            if target_state not in allowed:
                raise StateTransitionError(
                    f"Illegal transition: Cannot transition from {self.current_state.value} to {target_state.value}."
                )

            from_state = self.current_state.value
            to_state = target_state.value
            self._step_counter += 1
            step = self._step_counter
            meta = metadata or {}

            # Canonical step representation (deterministic, excluded non-deterministic time)
            canonical_step = {
                "step": step,
                "from_state": from_state,
                "to_state": to_state,
                "actor": actor,
                "reason": reason,
                "metadata": {k: str(v) for k, v in sorted(meta.items())},
            }
            raw = json.dumps(canonical_step, sort_keys=True, separators=(",", ":")).encode("utf-8")
            step_hash = hashlib.sha256(raw).hexdigest()

            self._history.append({
                "step": step,
                "from_state": from_state,
                "to_state": to_state,
                "actor": actor,
                "reason": reason,
                "step_hash": step_hash,
            })

            self.current_state = target_state
            return step_hash

    def compute_state_hash(self) -> str:
        """
        Compute an immutable, deterministic SHA-256 digest of the entire state machine.
        Guaranteed to produce identical digests across runs for identical transitions.
        """
        with self._lock:
            state_dict = {
                "task_id": self.task_id,
                "candidate_id": self.candidate_id,
                "version": self.version,
                "current_state": self.current_state.value,
                "step_count": self._step_counter,
                "history": [
                    {
                        "step": h["step"],
                        "from_state": h["from_state"],
                        "to_state": h["to_state"],
                        "actor": h["actor"],
                        "step_hash": h["step_hash"],
                    }
                    for h in self._history
                ],
            }
            canonical_bytes = json.dumps(state_dict, sort_keys=True, separators=(",", ":")).encode("utf-8")
            return hashlib.sha256(canonical_bytes).hexdigest()

    def get_history(self) -> List[Dict[str, Any]]:
        """Retrieve copy of transition history."""
        with self._lock:
            return list(self._history)


def verify_deterministic_50_runs() -> Dict[str, Any]:
    """
    Executes 50 independent runs of the state machine transition pipeline
    and verifies that 100% of runs yield the exact identical SHA-256 state hash.
    Fulfills PARIKSHAK deterministic execution gate.
    """
    state_hashes: List[str] = []

    for run_idx in range(1, 51):
        sm = DeterministicStateMachine(task_id="T-GOV-002", candidate_id="Prakash Yadav", version="2.0.0")
        sm.transition_to(State.SUBMITTED, actor="Prakash Yadav", reason="Initial candidate commit submission")
        sm.transition_to(State.VALIDATING, actor="NIYANTRAN", reason="Schema and signature check")
        sm.transition_to(State.RULE_EVALUATION, actor="RuleEngine", reason="PARIKSHAK multi-gate evaluation")
        sm.transition_to(State.AUDIT_EMITTED, actor="PravahEmitter", reason="Hash-chained audit entry registered")
        sm.transition_to(State.FINALIZED, actor="PARIKSHAK", reason="100/100 score approved")

        h = sm.compute_state_hash()
        state_hashes.append(h)

    first_hash = state_hashes[0]
    is_deterministic = all(h == first_hash for h in state_hashes)

    return {
        "runs": len(state_hashes),
        "is_deterministic": is_deterministic,
        "canonical_state_hash": first_hash,
        "matching_runs": sum(1 for h in state_hashes if h == first_hash),
    }
