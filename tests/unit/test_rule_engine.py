"""
Unit tests for TANTRA Multi-Gate Rule Evaluation Engine.
"""

import time
import uuid
from evaluation_engine.models import EvaluationContext
from evaluation_engine.rule_engine import RuleEngine
from security.jwt_handler import JWTHandler, RS256SecurityEngine


def test_rule_engine_full_pass_100_score(security_engine: RS256SecurityEngine) -> None:
    """Verify that a compliant submission passes all 5 rules with score 100/100."""
    handler = JWTHandler(security_engine)
    engine = RuleEngine(jwt_handler=handler)

    token = handler.issue_token(
        sub="Prakash Yadav",
        roles=["admin", "auditor"],
        permissions=["governance:evaluate", "replay:verify"],
        candidate_id="Prakash Yadav",
    )

    unique_id = uuid.uuid4().hex[:8]
    ctx = EvaluationContext(
        task_id="T-GOV-002",
        candidate_id="Prakash Yadav",
        actor="Prakash Yadav",
        token=token,
        sender_id=f"prakash-eval-{unique_id}",
        nonce=int(time.time() * 1000),
        payload={"task": "T-GOV-002", "scope": "evaluation", "uid": unique_id},
        run_50_determinism_check=True,
    )

    verdict = engine.evaluate(ctx)
    assert verdict.verdict == "PASS"
    assert verdict.total_score == 100.0
    assert verdict.passed_rules == 5
    assert verdict.failed_rules == 0
    assert len(verdict.proof_digest) == 64
    assert len(verdict.rule_results) == 5


def test_rule_engine_replay_failure_causes_fail(security_engine: RS256SecurityEngine) -> None:
    """Verify that a duplicate replay payload fails Rule 2 and sets verdict to FAIL."""
    handler = JWTHandler(security_engine)
    engine = RuleEngine(jwt_handler=handler)

    token = handler.issue_token(sub="Prakash Yadav", roles=["admin"])
    unique_id = uuid.uuid4().hex[:8]
    payload = {"unique": f"test-data-for-replay-{unique_id}"}
    sender = f"prakash-replay-sender-{unique_id}"
    nonce = int(time.time() * 1000)

    # First evaluation: Registers the payload
    ctx1 = EvaluationContext(
        task_id="T-GOV-002",
        candidate_id="Prakash Yadav",
        token=token,
        sender_id=sender,
        nonce=nonce,
        payload=payload,
        run_50_determinism_check=False,
    )
    v1 = engine.evaluate(ctx1)
    assert v1.verdict == "PASS"

    # Second evaluation: Identical payload with lower/same nonce fails Rule 2
    ctx2 = EvaluationContext(
        task_id="T-GOV-002",
        candidate_id="Prakash Yadav",
        token=token,
        sender_id=sender,
        nonce=nonce,
        payload=payload,
        run_50_determinism_check=False,
    )
    v2 = engine.evaluate(ctx2)
    assert v2.verdict == "FAIL"
    assert v2.total_score < 100.0
    assert v2.failed_rules >= 1
    replay_res = [r for r in v2.rule_results if r.rule_id == "RULE-REPLAY-02"][0]
    assert replay_res.passed is False
    assert replay_res.score == 0.0


def test_rule_engine_invalid_token_fails_rbac(security_engine: RS256SecurityEngine) -> None:
    """Verify that an invalid token fails Rule 1."""
    handler = JWTHandler(security_engine)
    engine = RuleEngine(jwt_handler=handler)

    ctx = EvaluationContext(
        task_id="T-GOV-002",
        candidate_id="Prakash Yadav",
        token="invalid.malformed.token",
        run_50_determinism_check=False,
    )
    verdict = engine.evaluate(ctx)
    assert verdict.verdict == "FAIL"
    rbac_res = [r for r in verdict.rule_results if r.rule_id == "RULE-RBAC-01"][0]
    assert rbac_res.passed is False
    assert rbac_res.score == 0.0
