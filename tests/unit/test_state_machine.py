"""
Unit tests for Deterministic State Machine and 50 Consecutive Runs Invariance.
"""

import pytest
from security.state_machine import (
    DeterministicStateMachine,
    State,
    StateTransitionError,
    verify_deterministic_50_runs,
)


def test_valid_state_transitions() -> None:
    """Verify standard happy-path lifecycle transitions."""
    sm = DeterministicStateMachine(task_id="T-GOV-002", candidate_id="Prakash Yadav")
    assert sm.current_state == State.DRAFT

    h1 = sm.transition_to(State.SUBMITTED, actor="Prakash Yadav", reason="Candidate submission")
    assert sm.current_state == State.SUBMITTED
    assert len(h1) == 64

    h2 = sm.transition_to(State.VALIDATING, actor="NIYANTRAN", reason="Running schema check")
    assert sm.current_state == State.VALIDATING

    h3 = sm.transition_to(State.RULE_EVALUATION, actor="RuleEngine", reason="PARIKSHAK rules")
    assert sm.current_state == State.RULE_EVALUATION

    h4 = sm.transition_to(State.AUDIT_EMITTED, actor="PravahEmitter", reason="Ledger record created")
    assert sm.current_state == State.AUDIT_EMITTED

    h5 = sm.transition_to(State.FINALIZED, actor="PARIKSHAK", reason="100/100 score approved")
    assert sm.current_state == State.FINALIZED

    state_hash = sm.compute_state_hash()
    assert len(state_hash) == 64
    assert len(sm.get_history()) == 5


def test_illegal_state_transition_prevented() -> None:
    """Verify that jumping across illegal states is blocked by guard."""
    sm = DeterministicStateMachine()

    with pytest.raises(StateTransitionError, match="Illegal transition: Cannot transition from DRAFT to FINALIZED"):
        sm.transition_to(State.FINALIZED, actor="malicious_actor")

    with pytest.raises(StateTransitionError, match="Illegal transition: Cannot transition from DRAFT to VALIDATING"):
        sm.transition_to(State.VALIDATING, actor="malicious_actor")


def test_rejection_branch_transitions() -> None:
    """Verify that transitions can branch into REJECTED."""
    sm = DeterministicStateMachine()
    sm.transition_to(State.SUBMITTED, actor="user")
    sm.transition_to(State.REJECTED, actor="guard", reason="Signature verification failed")
    assert sm.current_state == State.REJECTED

    # Terminal state cannot transition further
    with pytest.raises(StateTransitionError, match="Illegal transition: Cannot transition from REJECTED"):
        sm.transition_to(State.SUBMITTED, actor="user")


def test_terminal_finalized_state_cannot_transition() -> None:
    """Verify that FINALIZED terminal state rejects any further transitions."""
    sm = DeterministicStateMachine()
    sm.transition_to(State.SUBMITTED, actor="user")
    sm.transition_to(State.VALIDATING, actor="user")
    sm.transition_to(State.RULE_EVALUATION, actor="user")
    sm.transition_to(State.AUDIT_EMITTED, actor="user")
    sm.transition_to(State.FINALIZED, actor="user")

    with pytest.raises(StateTransitionError):
        sm.transition_to(State.DRAFT, actor="user")


def test_deterministic_50_runs_identical_hashes() -> None:
    """
    Fulfills PARIKSHAK quality gate:
    Verify that 50 consecutive runs produce 100% identical SHA-256 state hashes.
    """
    result = verify_deterministic_50_runs()
    assert result["runs"] == 50
    assert result["is_deterministic"] is True
    assert result["matching_runs"] == 50
    assert len(result["canonical_state_hash"]) == 64
