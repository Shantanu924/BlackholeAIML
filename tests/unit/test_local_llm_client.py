"""
Unit tests for Resilient Local LLM Client Integration.
"""

from integrations.local_llm_client import LLMReviewSummary, LocalLLMClient


def test_local_llm_client_fallback_mode() -> None:
    """Verify that when external endpoint is offline, client returns deterministic fallback."""
    client = LocalLLMClient(base_url="http://localhost:9999", timeout_seconds=0.1, max_retries=1)

    health = client.check_health()
    assert health["status"] == "OFFLINE_FALLBACK_ACTIVE"

    summary = client.analyze_task_submission(
        task_id="T-GOV-002",
        candidate_id="Prakash Yadav",
        context={"status": "testing"},
    )

    assert isinstance(summary, LLMReviewSummary)
    assert summary.task_id == "T-GOV-002"
    assert summary.candidate_id == "Prakash Yadav"
    assert summary.compliance_verdict == "COMPLIANT"
    assert summary.confidence_score == 1.0
    assert summary.is_fallback is True
    assert len(summary.key_findings) > 0
    assert len(summary.suggested_actions) > 0
