"""
Unit tests for PARIKSHAK Review Orchestrator.
"""

import tempfile
import os
from task_selector.review_orchestrator import ReviewOrchestrator, TaskSubmission


def test_orchestrator_verify_pinned_dependencies() -> None:
    """Verify Gate 2: Pinned dependencies check passes strictly with == and fails with >=."""
    orchestrator = ReviewOrchestrator()

    # 1. Pinned file
    with tempfile.NamedTemporaryFile("w", suffix=".txt", delete=False) as f:
        f.write("fastapi==0.115.8\npydantic==2.13.3\npytest==9.0.3\n")
        pinned_path = f.name

    try:
        res = orchestrator.verify_pinned_dependencies(pinned_path)
        assert res.passed is True
        assert res.score == 20.0
    finally:
        os.remove(pinned_path)

    # 2. Unpinned file with >=
    with tempfile.NamedTemporaryFile("w", suffix=".txt", delete=False) as f:
        f.write("fastapi>=0.115.8\npydantic==2.13.3\n")
        unpinned_path = f.name

    try:
        res2 = orchestrator.verify_pinned_dependencies(unpinned_path)
        assert res2.passed is False
        assert res2.score == 0.0
        assert "Found unpinned dependencies" in res2.details
    finally:
        os.remove(unpinned_path)


def test_orchestrator_full_evaluation_submission() -> None:
    """Verify that Prakash Yadav's submission achieves 100/100 and status PASS."""
    orchestrator = ReviewOrchestrator()

    sub = TaskSubmission(
        task_id="T-GOV-002",
        candidate_id="Prakash Yadav",
        requirements_path="requirements.txt",
    )

    report = orchestrator.evaluate_submission(sub)
    assert report.task_id == "T-GOV-002"
    assert report.candidate_id == "Prakash Yadav"
    assert report.overall_status == "PASS"
    assert report.total_score == 100.0
    assert len(report.gates) == 5
    assert all(g.passed for g in report.gates)
    assert len(report.certified_proof_hash) == 64
