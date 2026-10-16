# Canonical API Request & Response Samples

This document provides exact curl commands, headers, and responses validating the live VAJRA runtime and TANTRA governance API.

---

## 1. System Health Check
```bash
curl -X GET http://localhost:8000/api/v1/health
```
**Response (HTTP 200 OK):**
```json
{
  "status": "HEALTHY",
  "service": "TANTRA Governance & VAJRA Trust Layer",
  "task_id": "T-GOV-002",
  "authority": "PARIKSHAK Production Runtime Engine",
  "version": "2.0.0",
  "timestamp": 1790072400.123456
}
```

---

## 2. Issue Enterprise RS256 Token
```bash
curl -X POST http://localhost:8000/api/v1/auth/token \
  -H "Content-Type: application/json" \
  -d '{
    "sub": "prakash-admin",
    "roles": ["admin", "auditor"],
    "permissions": ["governance:evaluate", "replay:verify"],
    "tenant_id": "tantra-mainnet",
    "candidate_id": "Prakash Yadav",
    "ttl_seconds": 3600
  }'
```
**Response (HTTP 200 OK):**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InRhbnRyYS1wcmltYXJ5LXYxIn0...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "claims": {
    "sub": "prakash-admin",
    "roles": ["admin", "auditor"],
    "permissions": ["governance:evaluate", "replay:verify"],
    "tenant_id": "tantra-mainnet",
    "candidate_id": "Prakash Yadav",
    "jti": "5a4f7832-7f28-4bb9-9ff3-c35105eb9c0a",
    "iss": "tantra.governance.niyantran",
    "aud": "vajra.production.runtime",
    "iat": 1790072400,
    "exp": 1790076000,
    "nbf": null
  }
}
```

---

## 3. Pravah Replay Verification
```bash
curl -X POST http://localhost:8000/api/v1/replay/verify \
  -H "Content-Type: application/json" \
  -d '{
    "sender_id": "0xValidatorAlpha",
    "nonce": 1001,
    "payload": {"tx": "token_transfer", "amount": 250, "token": "BHIV"}
  }'
```
**Response (HTTP 200 OK):**
```json
{
  "valid": true,
  "status": "ACCEPTED",
  "reason": "Payload and nonce verified unique. Registered in Pravah replay ledger.",
  "proof_digest": "3c98fe37b198c25345dd9c...",
  "trace_id": "trace-8f921a4bc3"
}
```

**Duplicate Replay Attempt Response (HTTP 409 Conflict):**
```json
{
  "error": true,
  "code": "REPLAY_DETECTED",
  "message": "Replay rejected: Nonce 1001 is invalid for sender '0xValidatorAlpha'. Current height: 1001.",
  "trace_id": "trace-8f921a4bc3",
  "timestamp": 1790072405.654321
}
```

---

## 4. Emit Pravah Audit Event
```bash
curl -X POST http://localhost:8000/api/v1/audit/emit \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "BLOCK_COMMITTED",
    "actor": "blackhole-node-1",
    "action": "COMMIT_BLOCK",
    "resource": "blockchain/block/1045",
    "status": "SUCCESS",
    "metadata": {"height": 1045, "tx_count": 8},
    "payload": {"block_hash": "0x98abc123..."}
  }'
```
**Response (HTTP 200 OK):**
```json
{
  "event_id": "evt-7a91bf2e4d",
  "timestamp": 1790072410.123456,
  "event_type": "BLOCK_COMMITTED",
  "actor": "blackhole-node-1",
  "action": "COMMIT_BLOCK",
  "resource": "blockchain/block/1045",
  "status": "SUCCESS",
  "current_hash": "e027297c24543a0d1f1c68464257a62606c675192860ee05d2d57f429283a724",
  "previous_hash": "7fb2f0496046804c9ce6992377bdc541ef0d960f4e49bff0501ebb4c457fe330"
}
```

---

## 5. Verify Cryptographic Audit Hash Chain
```bash
curl -X GET http://localhost:8000/api/v1/audit/verify-chain
```
**Response (HTTP 200 OK):**
```json
{
  "valid": true,
  "message": "Chain integrity verified: 45 consecutive blocks cryptographically valid."
}
```

---

## 6. Deterministic State Transition
```bash
curl -X POST http://localhost:8000/api/v1/governance/transition \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "target_state": "SUBMITTED",
    "actor": "prakash-admin",
    "reason": "Candidate submission commit"
  }'
```
**Response (HTTP 200 OK):**
```json
{
  "success": true,
  "current_state": "SUBMITTED",
  "step_hash": "8f902acb341...",
  "state_hash": "afc509b35cd72c00e5b0c59ad1d8e2cfc0254598670ddc0c1990b38b63da3712"
}
```
