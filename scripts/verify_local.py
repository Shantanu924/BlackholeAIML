"""
Local Verification Script for VAJRA & TANTRA Subsystems
Executes live runtime checks across:
1. 50-Run Deterministic State Machine
2. Live FastAPI Endpoints (Health, Auth, Replay, Audit, Error Boundaries)
3. PARIKSHAK Quality Gate Audit
"""

import os
import sys
import time
import uuid
from pathlib import Path

# Add project root to PYTHONPATH
sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from starlette.testclient import TestClient
from api.production import app
from security.state_machine import verify_deterministic_50_runs
from task_selector.review_orchestrator import ReviewOrchestrator, TaskSubmission

client = TestClient(app, raise_server_exceptions=False)

print("=" * 75)
print("1. DETERMINISM CHECK (50 CONSECUTIVE RUNS)")
print("=" * 75)
det = verify_deterministic_50_runs()
print(f"Runs Executed: {det['runs']}")
print(f"Is Deterministic: {det['is_deterministic']} ({det['matching_runs']}/50 matched)")
print(f"Canonical State Hash: {det['canonical_state_hash']}")

print("\n" + "=" * 75)
print("2. LIVE PRODUCTION API ENDPOINT TESTS")
print("=" * 75)

# A. Health
h_resp = client.get("/health")
print(f"[GET /health] Status: {h_resp.status_code}")
print(f"  Response: {h_resp.json()}")

# B. Public Key Discovery
pk_resp = client.get("/api/v1/auth/public-key")
print(f"\n[GET /api/v1/auth/public-key] Status: {pk_resp.status_code}")
print(f"  Algorithm: {pk_resp.json().get('alg')}, Format: {pk_resp.json().get('format')}")

# C. Issue RS256 Enterprise JWT
tok_resp = client.post("/api/v1/auth/token", json={
    "sub": "prakash-local-test",
    "roles": ["admin", "auditor"],
    "permissions": ["governance:evaluate", "replay:verify"],
    "tenant_id": "tantra-mainnet",
    "candidate_id": "Prakash Yadav",
    "ttl_seconds": 3600
})
tok_data = tok_resp.json()
token = tok_data["access_token"]
print(f"\n[POST /api/v1/auth/token] Status: {tok_resp.status_code}")
print(f"  Issued Token: {token[:40]}... (Total {len(token)} chars)")
print(f"  Verified Claims: sub='{tok_data['claims']['sub']}', roles={tok_data['claims']['roles']}")

# D. Replay Mitigation Defense (First Submission)
uid = uuid.uuid4().hex[:8]
nonce = int(time.time() * 1000)
payload = {"block": 1045, "action": "SETTLEMENT", "asset": "BHIV", "uid": uid}

rep1 = client.post("/api/v1/replay/verify", json={
    "sender_id": f"validator-{uid}",
    "nonce": nonce,
    "payload": payload
})
print(f"\n[POST /api/v1/replay/verify (First)] Status: {rep1.status_code}")
print(f"  Result: {rep1.json()['status']} | Reason: {rep1.json()['reason']}")
print(f"  Cryptographic Proof Digest: {rep1.json()['proof_digest']}")

# E. Replay Mitigation Defense (Duplicate Replay Attempt)
rep2 = client.post("/api/v1/replay/verify", json={
    "sender_id": f"validator-{uid}",
    "nonce": nonce,
    "payload": payload
})
print(f"\n[POST /api/v1/replay/verify (Duplicate Replay)] Status: {rep2.status_code} (Expected HTTP 409)")
print(f"  Error Intercepted: {rep2.json()['error']} | Code: {rep2.json()['code']}")
print(f"  Rejection Message: {rep2.json()['message']}")

# F. Emit Immutable Audit Event
aud_resp = client.post("/api/v1/audit/emit", json={
    "event_type": "LOCAL_VERIFICATION_CHECK",
    "actor": "Prakash Yadav",
    "action": "RUN_LOCAL_SYSTEM_VERIFICATION",
    "resource": "local/runtime/check",
    "status": "SUCCESS",
    "metadata": {"checked_by": "Prakash Yadav", "environment": "local"},
    "payload": {"test_uid": uid}
})
aud_data = aud_resp.json()
print(f"\n[POST /api/v1/audit/emit] Status: {aud_resp.status_code}")
print(f"  Event ID: {aud_data['event_id']}")
print(f"  Current SHA-256 Hash: {aud_data['current_hash']}")
print(f"  Previous Hash Link:   {aud_data['previous_hash']}")

# G. Verify Audit Hash Chain Integrity
ch_resp = client.get("/api/v1/audit/verify-chain")
print(f"\n[GET /api/v1/audit/verify-chain] Status: {ch_resp.status_code}")
print(f"  Chain Valid: {ch_resp.json()['valid']} | Message: {ch_resp.json()['message']}")

# H. Error Boundary Safety (Hostile / Malformed Schema)
err_resp = client.post("/api/v1/auth/token", json={
    "sub": "valid_user",
    "illegal_unmapped_field": "malicious_injection"
})
print(f"\n[POST /api/v1/auth/token (Hostile Input)] Status: {err_resp.status_code} (Expected HTTP 422, NOT 500)")
print(f"  Trapped by Error Boundary: {err_resp.json()['error']} | Code: {err_resp.json()['code']}")
print(f"  Safety Message: {err_resp.json()['message']}")

print("\n" + "=" * 75)
print("3. PARIKSHAK AUTOMATED QUALITY GATE AUDIT")
print("=" * 75)
rep = ReviewOrchestrator().evaluate_submission(TaskSubmission())
print(f"Task ID: {rep.task_id} | Candidate: {rep.candidate_id}")
print(f"Authority: {rep.authority}")
print(f"Overall Status: {rep.overall_status} | Total Score: {rep.total_score} / 100.0")
print(f"Certified Proof Hash: {rep.certified_proof_hash}")
print("\nIndividual Gate Verification Matrix:")
for g in rep.gates:
    status_str = "PASS" if g.passed else "FAIL"
    print(f"  * [{g.gate_id}] {g.name}")
    print(f"    Score: {g.score}/{g.max_score} | Status: {status_str}")
    print(f"    Details: {g.details}")
print("=" * 75)
