# Phase 3 Distributed Replay Review Packet
**Author:** Prakash Kumar
**Task:** Constitutional Replay Infrastructure — BlackHole Blockchain TANTRA Phase 3
**Date:** 09 May 2026
**Phases Delivered:** 1 + 2 + 3 + 4 + 5 — ALL COMPLETE

---

## 1. Entry Point

All transactions enter through ONE canonical endpoint:
```
POST http://localhost:8080/api/relay/submit
```

Execution path (Phase 3 hardened):
```
Wallet Signature
→ Relay Signature Verification  (Phase 1 — sigverify)
→ Persistent Nonce Check        (Phase 2 — noncestore)
→ Distributed PDV               (3 independent agents)
→ Governance Gate               (Sarathi/DGIC port 9090)
→ Blockchain Write
→ Bucket Write
→ AKASHIC Append
→ KarmaChain Replication
```

---

## 2. Distributed Replay Flow

```
Transaction submitted
        │
        ▼
Signature verified (sigverify.Verify)
        │
        ▼
Nonce checked against persistent store (noncestore.CheckAndAccept)
        │
        ▼
Distributed PDV — 3 agents in parallel (pdv.Check)
  ├── ExecutionAgent  → hash_A
  ├── ValidationAgent → hash_B
  └── ReplayAgent     → hash_C
  hash_A == hash_B == hash_C → PASS
        │
        ▼
Governance (FraudGate → Sarathi port 9090)
        │
        ▼
Blockchain Write → Bucket Write → AKASHIC Append → KarmaChain Replication
        │
        ▼
Replay verification available:
  GET /api/replay/equality?trace_id=<id>   → node A == node B == node C
  GET /api/replay/state-root               → final_state_root equality
  GET /api/akashic/reconstruct             → lineage reconstruction proof
```

---

## 3. Multi-Node Equality Proof

**Endpoint:** `GET /api/replay/equality?trace_id=<id>`

Queries all configured `KARMACHAIN_NODES` for the same trace_id.
Compares `execution_hash`, `validation_hash`, `replay_hash` across nodes.

Expected response (single node — localhost):
```json
{
  "success": true,
  "result": {
    "equal": true,
    "trace_id": "3e9b3ef561e26247",
    "node_count": 1,
    "agreed_count": 1,
    "divergent_count": 0,
    "consensus_hash": "894ba736...",
    "state_root": "67c671b7...",
    "message": "all nodes agree — distributed replay equality confirmed"
  }
}
```

**Endpoint:** `GET /api/replay/state-root`

Compares `final_state_root` across all nodes.
Any divergence → `STATE_ROOT_DIVERGENCE` — hard fail.

---

## 4. Signature Verification Proof

**Package:** `core/relay-chain/sigverify/sigverify.go`

**Endpoint:** `POST /api/sig/verify`

Verification flow:
1. Decode `from` address as compressed btcec public key
2. Compute canonical payload hash (timestamp excluded — replay-safe)
3. Decode DER signature from `signature` field
4. Verify signature against hash using btcec/ecdsa

**Invalid signature test:**
```json
POST /api/relay/submit
{ "signature": "deadbeef", "from": "03e2459b...", ... }
→ HTTP 403, error_code: SIGNATURE_REJECT
```

**Named address (alice/bob) — signature skipped with warning:**
```json
POST /api/relay/submit
{ "from": "alice", ... }
→ HTTP 200, signature_valid: true (named address — not a pubkey)
```

---

## 5. Nonce Persistence Proof

**Package:** `core/relay-chain/noncestore/noncestore.go`
**File:** `nonce_ledger.jsonl`

**Restart recovery:**
1. Node starts → `noncestore.New()` loads all existing nonces from `nonce_ledger.jsonl`
2. All previously used nonces are in `seen` map
3. Duplicate nonce after restart → `NONCE_REPLAY` HTTP 409

**Endpoints:**
- `GET /api/nonce/lookup?address=<addr>` — latest nonce + next expected nonce
- `GET /api/nonce/records` — full nonce lineage

**Duplicate nonce test:**
```
Send nonce=5 → HTTP 200 (accepted)
Send nonce=5 again → HTTP 409, error_code: NONCE_REPLAY
Restart node → Send nonce=5 again → HTTP 409 (persisted — still rejected)
```

---

## 6. Failure Reconstruction Proof

**Phase 4 — Corruption simulation and detection:**

**Endpoint:** `POST /api/akashic/corrupt-simulate`

1. Corrupts the last AKASHIC entry's `entry_hash`
2. Immediately runs `Reconstruct()` to verify detection
3. Returns `corruption_detected: true` if chain integrity check fails

Expected response:
```json
{
  "success": true,
  "corrupted_hash": "70197701...",
  "corruption_detected": true,
  "reconstruction": {
    "verified": false,
    "chain_intact": false,
    "broken_at": 4,
    "message": "chain broken at entry 4: entry_hash mismatch"
  }
}
```

**Recovery from Bucket + AKASHIC:**
- `GET /api/karmachain/reconstruct` — pulls lineage from first reachable node
- `GET /api/akashic/reconstruct` — verifies local lineage integrity
- `GET /api/tantra/chain-integrity` — verifies Bucket chain integrity

---

## 7. Divergence Detection Cases

| Case | Detection | Endpoint |
|---|---|---|
| PDV hash mismatch across agents | `DISTRIBUTED_PDV_REJECT` | `POST /api/relay/submit` |
| State root divergence across nodes | `STATE_ROOT_DIVERGENCE` | `GET /api/replay/state-root` |
| AKASHIC entry tampered | `chain broken at entry N` | `GET /api/akashic/reconstruct` |
| Bucket entry tampered | `entry_hash mismatch` | `GET /api/tantra/chain-integrity` |
| Nonce replay after restart | `NONCE_REPLAY` | `POST /api/relay/submit` |
| Invalid signature | `SIGNATURE_REJECT` | `POST /api/relay/submit` |
| Trace drift | `TRACE_BREAK` | `POST /api/relay/submit` |

---

## 8. Constitutional Boundary Declaration

**Endpoint:** `GET /api/constitution/declaration`

**Owned by this system:**
- PDV equality (execution_hash == validation_hash == replay_hash)
- Signature verification (btcec/ecdsa before PDV)
- Nonce governance (persistent, restart-safe)
- Trace continuity (immutable trace_id)
- Bucket truth (chain-linked append-only)
- AKASHIC lineage (chain-linked reconstruction)

**NOT owned by this system:**
- Governance legitimacy → Sarathi/DGIC decides
- Semantic truth → replay proves determinism, not correctness
- Sovereign correctness → requires KSML/CET + external constitutional validation

**Bounded:**
- Replay equality ≠ governance legitimacy
- Observability ≠ execution authority
- Wallet intent ≠ execution right

---

## 9. Hidden-State Disclosure

| Hidden State | Location | Disclosure |
|---|---|---|
| PDV agents fall back to local computation | `pdv/pdv.go` | When `PDV_*_AGENT_URL` not set, all 3 agents run locally — same process |
| Sarathi on localhost | `enforcement/tantra.go` | FraudGate calls `localhost:9090` — not a truly external sovereign service |
| KarmaChain single-node | `karmachain/karmachain.go` | If `KARMACHAIN_NODES` not set, replicates to localhost only |
| Nonce store per-node | `noncestore/noncestore.go` | Cross-node nonce coordination not implemented |
| Named addresses skip sig check | `sigverify/sigverify.go` | `alice`, `bob` etc. are not pubkeys — sig verification skipped with warning |

---

## 10. Replay Authority Isolation Proof

**Endpoint:** `GET /api/constitution/verify-boundary?name=REPLAY_EQUALITY_VS_LEGITIMACY`

Response:
```json
{
  "success": true,
  "boundary": "REPLAY_EQUALITY_VS_LEGITIMACY",
  "intact": true,
  "proof": "FraudGate is called after PDV PASS — governance is separate from determinism"
}
```

**Key isolation proofs:**
- `FraudGate` is called AFTER PDV equality gate — governance cannot contaminate determinism
- `Reconstruct()` verifies hash chain integrity — not business logic
- `/api/tantra/records` is read-only — no write authority from observability
- Wallet `Execute()` routes through `/api/relay/submit` only — no direct blockchain access

---

## 11. Vinayak Testing Packet

### Start services
```
Terminal 1: cd core/relay-chain && go run cmd/fraud-service/main.go
Terminal 2: cd core/relay-chain && go run cmd/relay/main.go 3000
```

### Get fresh timestamp
```powershell
$ts = [int][double]::Parse((Get-Date -UFormat %s))
```

### Test 1 — Invalid signature rejection (Phase 1)
```powershell
$body = '{"schema_version":"v1","type":"token_transfer","from":"03e2459b73c0c6522530f6b26e834d992dfc55d170bee35d0bcdc047fe0d61c25b","to":"bob","amount":100,"token_id":"BHIV","nonce":1,"timestamp":' + $ts + ',"signature":"deadbeef"}'
try { Invoke-RestMethod -Uri "http://localhost:8080/api/relay/submit" -Method POST -ContentType "application/json" -Body $body } catch { $_.ErrorDetails.Message }
```
Expected: `error_code: SIGNATURE_REJECT`

### Test 2 — Successful transaction (named address)
```powershell
$body = '{"schema_version":"v1","type":"token_transfer","from":"alice","to":"bob","amount":100,"token_id":"BHIV","nonce":1,"timestamp":' + $ts + '}'
Invoke-RestMethod -Uri "http://localhost:8080/api/relay/submit" -Method POST -ContentType "application/json" -Body $body | ConvertTo-Json
```
Expected: `success: true, signature_valid: true`

### Test 3 — Duplicate nonce replay (Phase 2)
```powershell
# Send same nonce twice
$body = '{"schema_version":"v1","type":"token_transfer","from":"alice","to":"bob","amount":100,"token_id":"BHIV","nonce":1,"timestamp":' + $ts + '}'
try { Invoke-RestMethod -Uri "http://localhost:8080/api/relay/submit" -Method POST -ContentType "application/json" -Body $body } catch { $_.ErrorDetails.Message }
```
Expected: `error_code: NONCE_REPLAY`

### Test 4 — Nonce lookup (Phase 2)
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/api/nonce/lookup?address=alice" | ConvertTo-Json
```
Expected: `latest_nonce: 1, next_nonce: 2`

### Test 5 — Distributed replay equality (Phase 3)
```powershell
# Use trace_id from Test 2
Invoke-RestMethod -Uri "http://localhost:8080/api/replay/equality?trace_id=<trace_id>" | ConvertTo-Json
```
Expected: `equal: true, agreed_count: 1`

### Test 6 — State root equality (Phase 3)
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/api/replay/state-root" | ConvertTo-Json
```
Expected: `equal: true`

### Test 7 — Corruption simulation and detection (Phase 4)
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/api/akashic/corrupt-simulate" -Method POST | ConvertTo-Json
```
Expected: `corruption_detected: true, reconstruction.verified: false`

### Test 8 — Constitutional declaration (Phase 5)
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/api/constitution/declaration" | ConvertTo-Json -Depth 5
```
Expected: full boundary declaration with owned/not-owned/bounded categories

### Test 9 — Boundary verification (Phase 5)
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/api/constitution/verify-boundary?name=REPLAY_EQUALITY_VS_LEGITIMACY" | ConvertTo-Json
```
Expected: `intact: true`

### Test 10 — Restart replay continuity (Phase 2)
```
1. Send transaction with nonce=10
2. Stop blockchain node (Ctrl+C)
3. Restart: go run cmd/relay/main.go 3000
4. Send same transaction with nonce=10 again
Expected: NONCE_REPLAY — nonce persisted across restart
```

---

## 12. Current Convergence Gaps

| Gap | Status | Notes |
|---|---|---|
| PDV agents on separate processes | Partial | Falls back to local — set PDV_*_AGENT_URL env vars for true distribution |
| Sarathi as truly external service | Partial | Runs on localhost:9090 — not sovereign external governance |
| Cross-node nonce coordination | Not built | Each node has its own nonce store |
| Named address signature enforcement | Partial | alice/bob skip sig check — production requires all addresses to be pubkeys |
| KSML/CET upstream alignment | Pending | Awaiting Raj Prajapati |

---

## All New Endpoints (Phase 1-5)

| Endpoint | Phase | Purpose |
|---|---|---|
| `POST /api/sig/verify` | 1 | Standalone signature verification |
| `GET /api/nonce/lookup?address=<addr>` | 2 | Nonce lineage lookup |
| `GET /api/nonce/records` | 2 | Full nonce ledger |
| `GET /api/replay/equality?trace_id=<id>` | 3 | Distributed hash equality |
| `GET /api/replay/state-root` | 3 | State root equality across nodes |
| `POST /api/akashic/corrupt-simulate` | 4 | Corruption simulation + detection |
| `GET /api/constitution/declaration` | 5 | Runtime authority boundaries |
| `GET /api/constitution/verify-boundary?name=<n>` | 5 | Boundary integrity check |
