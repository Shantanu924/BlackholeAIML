"""
Integration tests for Deterministic State Transitions and Concurrency Safety.
Validates PARIKSHAK Gate 3: 50 consecutive runs must produce identical SHA-256 state hashes.
"""

import concurrent.futures
from security.state_machine import (
    DeterministicStateMachine,
    State,
    verify_deterministic_50_runs,
)


def test_50_consecutive_runs_deterministic_state_hash() -> None:
    """
    PARIKSHAK GATE 3 VALIDATION:
    50 consecutive executions of the full lifecycle must produce
    the identical SHA-256 state hash.
    """
    verif = verify_deterministic_50_runs()
    assert verif["runs"] == 50
    assert verif["is_deterministic"] is True
    assert verif["matching_runs"] == 50

    # Ensure hash is standard 64-character lowercase hexadecimal
    canonical_hash = verif["canonical_state_hash"]
    assert len(canonical_hash) == 64
    assert int(canonical_hash, 16) > 0


def test_parallel_deterministic_concurrency() -> None:
    """
    Execute 50 state machines concurrently across multiple worker threads.
    Verify all 50 parallel runs produce the exact same canonical state hash.
    """
    def run_pipeline(idx: int) -> str:
        sm = DeterministicStateMachine(task_id="T-GOV-002", candidate_id="Prakash Yadav", version="2.0.0")
        sm.transition_to(State.SUBMITTED, actor="Prakash Yadav", reason="Candidate commit")
        sm.transition_to(State.VALIDATING, actor="NIYANTRAN", reason="Validation phase")
        sm.transition_to(State.RULE_EVALUATION, actor="RuleEngine", reason="Rule check")
        sm.transition_to(State.AUDIT_EMITTED, actor="PravahEmitter", reason="Audit commit")
        sm.transition_to(State.FINALIZED, actor="PARIKSHAK", reason="Finalized")
        return sm.compute_state_hash()

    results = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=10) as executor:
        futures = [executor.submit(run_pipeline, i) for i in range(50)]
        for fut in concurrent.futures.as_completed(futures):
            results.append(fut.result())

    assert len(results) == 50
    first_hash = results[0]
    assert all(h == first_hash for h in results), "Parallel execution produced non-deterministic hashes!"
