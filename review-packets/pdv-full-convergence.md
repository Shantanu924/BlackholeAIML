# PDV Full Convergence Review Packet
**Author:** Prakash Kumar
**Task:** Full PDV Convergence Integration Sprint — BlackHole Blockchain TANTRA Final Convergence
**Date:** 07 May 2026
**Status:** ALL 6 PHASES COMPLETE — LIVE TESTED AND PROVEN

---

## 1. Entry Point

All transactions enter through ONE canonical endpoint:

```
POST http://localhost:8080/api/relay/submit
```

No other entry path exists. `/api/admin/submit-transaction` now routes through the same PDV enforcement pipeline. There is no bypass path anywhere in the codebase.

---

## 2. Full Execution Flow

```
Wallet / External Client
        │
        ▼
POST /api/relay/submit  (Port 8080)
        │
        ▼
┌─────────────────────────────────────────────────────────┐
│  PHASE 3 — SCHEMA CONTRACT VALIDATION                   │
│  schema.ParseAndValidate(rawBody)                       │
│  • schema_version must be "v1"                          │
│  • unknown fields → REJECT (SCHEMA_VIOLATION)           │
│  • missing required fields → REJECT                     │
│  • amount == 0 → REJECT                                 │
└─────────────────────────────────────────────────────────┘
        │ (schema valid)
        ▼
┌─────────────────────────────────────────────────────────┐
│  PHASE 1 — PDV LAYER (deterministic correctness only)   │
│  ExecutionAgent  → execution_hash  (timestamp excluded) │
│  ValidationAgent → validation_hash (independent)        │
│  ReplayAgent     → replay_hash     (independent)        │
│                                                         │
│  EQUALITY GATE:                                         │
│  execution_hash == validation_hash == replay_hash       │
│  MISMATCH → HTTP 403 (PDV_REJECT)                       │
└─────────────────────────────────────────────────────────┘
        │ (PDV PASS)
        ▼
┌─────────────────────────────────────────────────────────┐
│  PHASE 2 — GOVERNANCE LAYER (Sarathi/DGIC)              │
│  FraudGate → POST http://localhost:9090/api/fraud/check │
│  decision == "block" → HTTP 403 (PDV_REJECT)            │
│  decision == "allow" → continue                         │
└─────────────────────────────────────────────────────────┘
        │ (fraud=allow)
        ▼
┌─────────────────────────────────────────────────────────┐
│  PHASE 1 — BLOCKCHAIN WRITE                             │
│  blockchain.ProcessTransaction(tx)                      │
│  failure → HTTP 422 (BLOCKCHAIN_REJECT)                 │
└─────────────────────────────────────────────────────────┘
        │ (blockchain write success)
        ▼
┌─────────────────────────────────────────────────────────┐
│  PHASE 5 — BUCKET WRITE (truthstore)                    │
│  truthstore.Append(trace_id + all 4 hashes + tx_hash)  │
│  chain-linked entries (prev_hash → entry_hash)          │
│  dual-write: local JSONL + remote TANTRA_BUCKET_URL     │
└─────────────────────────────────────────────────────────┘
        │ (bucket written)
        ▼
┌─────────────────────────────────────────────────────────┐
│  PHASE 4/6 — AKASHIC LINEAGE APPEND                     │
│  akashic.Append(trace_id + tx_hash + hashes + height)  │
│  chain-linked entries for reconstruction proof          │
└─────────────────────────────────────────────────────────┘
        │
        ▼
HTTP 200 — success response with all hashes + trace_id
```

---

## 3. Upstream + Downstream Integrations

| Layer | Component | Port | Direction |
|---|---|---|---|
| Entry | Wallet / External Client | — | → API |
| Schema | `schema.ParseAndValidate` | in-process | → PDV |
| PDV | `enforcement.Enforce` | in-process | → Fraud |
| Fraud/DGIC | Fraud Service | 9090 | → Blockchain |
| Blockchain | `chain.ProcessTransaction` | in-process | → Bucket |
| Bucket | `truthstore.Append` + remote URL | local + remote | → AKASHIC |
| AKASHIC | `akashic.Append` | in-process | → Response |

---

## 4. Trace Propagation Proof

`trace_id` lifecycle:
1. Client may supply `trace_id` in request body
2. If absent, `injectTraceID()` in `ExecutionAgent` generates it once from `SHA256(from:to:amount:nanotime)[:16]`
3. After `Enforce()` returns, `contract.TraceID = result.TraceID` propagates it to all downstream writes
4. Every log line carries `trace=%s` with the same ID
5. Every store write (truthstore, akashic) carries the same `trace_id`
6. `GET /api/akashic/trace?trace_id=<id>` verifies end-to-end continuity

**trace_id NEVER regenerates after injection. Any break = HARD FAIL.**

### LIVE PROOF — Real trace from 07 May 2026 test session

trace_id: `3e9b3ef561e26247`

```
[SCHEMA][PASS]         type=token_transfer from=alice to=bob amount=100
[PDV][ExecutionAgent]  trace=3e9b3ef561e26247 execution_hash=894ba736...
[PDV][ValidationAgent] trace=3e9b3ef561e26247 validation_hash=894ba736...
[PDV][ReplayAgent]     trace=3e9b3ef561e26247 replay_hash=894ba736...
[PDV][PASS]            trace=3e9b3ef561e26247 all_hashes=894ba736...
[Sarathi][FraudGate]   trace=3e9b3ef561e26247 decision=allow
[BLOCKCHAIN][WRITE]    trace=3e9b3ef561e26247 tx=538b3f68... height=1
[BUCKET][WRITE]        trace=3e9b3ef561e26247 tx=538b3f68...
[AKASHIC][APPEND]      trace=3e9b3ef561e26247 tx=538b3f68... entry_hash=70197701...
[TANTRA][COMPLETE]     trace=3e9b3ef561e26247 tx=538b3f68...
```

Same `trace_id` on every single line — trace continuity proven.

---

## 5. PDV Equality Proof

All three agents call `deterministicHash(tx)` on the same immutable `deterministicZone` struct after `injectTraceID` has run once.

`deterministicZone` contains: `trace_id, type, from, to, amount, token_id, fee, nonce, signature`

**Timestamp is explicitly excluded** — carried in `TxPayload` for fraud/audit but never enters the hash.

### LIVE PROOF — Real hashes from 07 May 2026 test session

```json
{
  "trace_id": "3e9b3ef561e26247",
  "execution_hash":  "894ba736f6726e192c4f7d78caf62256ee0fe843f4f23d83d83426cdcd987a5a",
  "validation_hash": "894ba736f6726e192c4f7d78caf62256ee0fe843f4f23d83d83426cdcd987a5a",
  "replay_hash":     "894ba736f6726e192c4f7d78caf62256ee0fe843f4f23d83d83426cdcd987a5a",
  "fraud_decision": "allow",
  "block_height": 1,
  "schema_version": "v1"
}
```

All three hashes are identical — `894ba736f6726e192c4f7d78caf62256ee0fe843f4f23d83d83426cdcd987a5a`

---

## 6. Bucket Integration Proof

`truthstore.Append()` writes:
- `trace_id` — immutable trace
- `execution_hash`, `validation_hash`, `replay_hash` — PDV proof
- `tx_hash` — blockchain transaction ID
- `fraud_decision` — Sarathi decision
- `prev_hash` — chain link to previous record
- `entry_hash` — SHA-256 of this record's content

Chain integrity verified at: `GET /api/tantra/chain-integrity`

### LIVE PROOF — Real Bucket records from 07 May 2026 test session

```json
{
  "count": 5,
  "records": [
    {
      "trace_id": "3e9b3ef561e26247",
      "execution_hash": "894ba736f6726e192c4f7d78caf62256ee0fe843f4f23d83d83426cdcd987a5a",
      "validation_hash": "894ba736f6726e192c4f7d78caf62256ee0fe843f4f23d83d83426cdcd987a5a",
      "replay_hash": "894ba736f6726e192c4f7d78caf62256ee0fe843f4f23d83d83426cdcd987a5a",
      "tx_hash": "538b3f681bd756865808ae9a4cf606745ea179d5e2a3d25798930ca73c813c17",
      "fraud_decision": "allow",
      "timestamp": 1778155881,
      "prev_hash": "genesis",
      "entry_hash": "ae4acc5851a55391013c29a9141f7640bc230c375ab14b891e1a4a7ac68e0b32"
    },
    {
      "trace_id": "92a9614efc5e0fca",
      "execution_hash": "74acff3492c271baf7b5d60162a38fc0daa9e447018304f95e1fbdf55d29a3f8",
      "validation_hash": "74acff3492c271baf7b5d60162a38fc0daa9e447018304f95e1fbdf55d29a3f8",
      "replay_hash": "74acff3492c271baf7b5d60162a38fc0daa9e447018304f95e1fbdf55d29a3f8",
      "tx_hash": "3df837cfdedc8fa61755136720642bd68716dadd21566e85cf04a07388281282",
      "fraud_decision": "allow",
      "timestamp": 1778156388,
      "prev_hash": "ae4acc5851a55391013c29a9141f7640bc230c375ab14b891e1a4a7ac68e0b32",
      "entry_hash": "d0357b62a1f96a5b8c1106f4a7d5e26044ff589b029396611c397e289169a1dd"
    },
    {
      "trace_id": "1008a1d26ebd9407",
      "execution_hash": "dba81b705f2f742f0a6fb915725a740b2c59918a8d9a1b2d4472651b892360f6",
      "validation_hash": "dba81b705f2f742f0a6fb915725a740b2c59918a8d9a1b2d4472651b892360f6",
      "replay_hash": "dba81b705f2f742f0a6fb915725a740b2c59918a8d9a1b2d4472651b892360f6",
      "tx_hash": "3df837cfdedc8fa61755136720642bd68716dadd21566e85cf04a07388281282",
      "fraud_decision": "allow",
      "timestamp": 1778156514,
      "prev_hash": "d0357b62a1f96a5b8c1106f4a7d5e26044ff589b029396611c397e289169a1dd",
      "entry_hash": "d4fefa4ff1bea3c87e49c5bff1e50469a39a535bcd0289470bbf7658ed48f182"
    },
    {
      "trace_id": "23661a1555ef5ce3",
      "execution_hash": "7267b025d630da3bd59f7d16ed2e66706d887c8edeb75ddf7081edff61969b72",
      "validation_hash": "7267b025d630da3bd59f7d16ed2e66706d887c8edeb75ddf7081edff61969b72",
      "replay_hash": "7267b025d630da3bd59f7d16ed2e66706d887c8edeb75ddf7081edff61969b72",
      "tx_hash": "3df837cfdedc8fa61755136720642bd68716dadd21566e85cf04a07388281282",
      "fraud_decision": "allow",
      "timestamp": 1778156524,
      "prev_hash": "d4fefa4ff1bea3c87e49c5bff1e50469a39a535bcd0289470bbf7658ed48f182",
      "entry_hash": "49e3f10668b13d44986a257134ab2ba725ac3b04ed15cfb70c309bc0f3f37d45"
    },
    {
      "trace_id": "a232b1904d7253b6",
      "execution_hash": "0508b82da4e006a7332bab2172c267184f78d3047bde029ff3043b13b4966f60",
      "validation_hash": "0508b82da4e006a7332bab2172c267184f78d3047bde029ff3043b13b4966f60",
      "replay_hash": "0508b82da4e006a7332bab2172c267184f78d3047bde029ff3043b13b4966f60",
      "tx_hash": "3df837cfdedc8fa61755136720642bd68716dadd21566e85cf04a07388281282",
      "fraud_decision": "allow",
      "timestamp": 1778156531,
      "prev_hash": "49e3f10668b13d44986a257134ab2ba725ac3b04ed15cfb70c309bc0f3f37d45",
      "entry_hash": "6ae5be950e538e6038a42533d463235119430baa24ce96bbc35e6ce97958e030"
    }
  ]
}
```

Chain link verification:
- Record 1: `prev_hash = "genesis"` ✅
- Record 2: `prev_hash = ae4acc58...` = Record 1 `entry_hash` ✅
- Record 3: `prev_hash = d0357b62...` = Record 2 `entry_hash` ✅
- Record 4: `prev_hash = d4fefa4f...` = Record 3 `entry_hash` ✅
- Record 5: `prev_hash = 49e3f106...` = Record 4 `entry_hash` ✅

Chain integrity endpoint response:
```json
{
  "intact": true,
  "message": "truth store chain is intact — no tampering detected",
  "success": true
}
```

---

## 7. AKASHIC Integration Proof

`akashic.Append()` writes a `LineageEntry` containing all PDV hashes, `trace_id`, `tx_hash`, `block_height`, and chain links.

AKASHIC write occurs **only after** Bucket write — enforced by sequential code order in `handleRelaySubmit`.

### LIVE PROOF — Real AKASHIC lineage from 07 May 2026 test session

```json
{
  "count": 5,
  "entries": [
    {
      "trace_id": "3e9b3ef561e26247",
      "tx_hash": "538b3f681bd756865808ae9a4cf606745ea179d5e2a3d25798930ca73c813c17",
      "execution_hash": "894ba736f6726e192c4f7d78caf62256ee0fe843f4f23d83d83426cdcd987a5a",
      "validation_hash": "894ba736f6726e192c4f7d78caf62256ee0fe843f4f23d83d83426cdcd987a5a",
      "replay_hash": "894ba736f6726e192c4f7d78caf62256ee0fe843f4f23d83d83426cdcd987a5a",
      "fraud_decision": "allow",
      "block_height": 1,
      "prev_hash": "genesis",
      "entry_hash": "70197701fd4e08fc5d35abe9df4e247475b8201dce881e4cdbdf033962e19672"
    },
    {
      "trace_id": "92a9614efc5e0fca",
      "block_height": 2,
      "prev_hash": "70197701fd4e08fc5d35abe9df4e247475b8201dce881e4cdbdf033962e19672",
      "entry_hash": "a6a7f34a2a9f936c15816412d936dd824f6310ea449ef423f328117b6affce8c"
    },
    {
      "trace_id": "1008a1d26ebd9407",
      "block_height": 3,
      "prev_hash": "a6a7f34a2a9f936c15816412d936dd824f6310ea449ef423f328117b6affce8c",
      "entry_hash": "b872053cde47ebde96562a61103b564b6c252400cb2817e3e6a1653118cd29f2"
    },
    {
      "trace_id": "23661a1555ef5ce3",
      "block_height": 4,
      "prev_hash": "b872053cde47ebde96562a61103b564b6c252400cb2817e3e6a1653118cd29f2",
      "entry_hash": "ccba0b9b390e782699dca4eba50124bde31c2429fea2c41749254ce13cd08033"
    },
    {
      "trace_id": "a232b1904d7253b6",
      "block_height": 5,
      "prev_hash": "ccba0b9b390e782699dca4eba50124bde31c2429fea2c41749254ce13cd08033",
      "entry_hash": "044ef3aa8a9b0f5592c6d61ac3f8d7d71430cbd9719183f32d142302a97f19b9"
    }
  ]
}
```

Block heights progress 1 → 2 → 3 → 4 → 5. Chain links intact across all 5 entries.

---

## 8. Replay + Reconstruction Proof

`GET /api/akashic/reconstruct` runs `akashic.Reconstruct()` which:
1. Reads all lineage entries from `akashic_lineage.jsonl`
2. Recomputes `entry_hash` for each entry and verifies it matches stored value
3. Verifies `prev_hash` chain link to previous entry
4. Verifies `execution_hash == validation_hash == replay_hash` within each entry
5. Computes `final_state_root` = rolling SHA-256 over all entry hashes in order

### LIVE PROOF — Real reconstruction output from 07 May 2026 test session

```json
{
  "result": {
    "total_entries": 5,
    "chain_intact": true,
    "final_state_root": "67c671b7336d1c0001a0b4c69fd7c674bc2287dc3f858bcfeb7f2700c7c7b37f",
    "verified": true,
    "message": "reconstruction successful — lineage intact"
  },
  "success": true
}
```

**Benchmark met: "If one node survives, execution truth survives."**
The `akashic_lineage.jsonl` file alone reconstructs and verifies the entire execution history.

---

## 9. Failure Cases

All failures are observable, structured, and deterministic. No silent failures.

| Failure | Error Code | HTTP | Log Tag |
|---|---|---|---|
| Malformed JSON | `SCHEMA_VIOLATION` | 400 | `[SCHEMA][REJECT]` |
| Unknown fields in payload | `SCHEMA_VIOLATION` | 400 | `[SCHEMA][REJECT]` |
| Missing required field | `SCHEMA_VIOLATION` | 400 | `[SCHEMA][REJECT]` |
| Wrong schema_version | `SCHEMA_VIOLATION` | 400 | `[SCHEMA][REJECT]` |
| Amount = 0 | `SCHEMA_VIOLATION` | 400 | `[SCHEMA][REJECT]` |
| PDV hash mismatch | `PDV_REJECT` | 403 | `[PDV][REJECT]` |
| Fraud gate block | `PDV_REJECT` | 403 | `[Sarathi][REJECT]` |
| Blockchain write fail | `BLOCKCHAIN_REJECT` | 422 | `[BLOCKCHAIN][REJECT]` |
| Truth store tampered | — | 409 | `[truthstore][TAMPER]` |
| AKASHIC chain broken | — | 409 | `[AKASHIC][TAMPER]` |

---

## 10. Real Outputs — Live Test Session 07 May 2026

### Test 1 — Successful Transaction (HTTP 200)

Request:
```json
POST /api/relay/submit
{
  "schema_version": "v1",
  "type": "token_transfer",
  "from": "alice",
  "to": "bob",
  "amount": 100,
  "token_id": "BHIV",
  "nonce": 1,
  "timestamp": 1778155881
}
```

Response:
```json
{
  "block_height": 1,
  "execution_hash": "894ba736f6726e192c4f7d78caf62256ee0fe843f4f23d83d83426cdcd987a5a",
  "fraud_decision": "allow",
  "replay_hash": "894ba736f6726e192c4f7d78caf62256ee0fe843f4f23d83d83426cdcd987a5a",
  "schema_version": "v1",
  "status": "pending",
  "submitted_at": 1778155881,
  "success": true,
  "trace_id": "3e9b3ef561e26247",
  "transaction_id": "538b3f681bd756865808ae9a4cf606745ea179d5e2a3d25798930ca73c813c17",
  "validation_hash": "894ba736f6726e192c4f7d78caf62256ee0fe843f4f23d83d83426cdcd987a5a"
}
```

All three hashes identical. ✅

---

### Test 2 — Fraud Rejection (HTTP 403)

Request: `from: "bad-actor"`

Response:
```json
{
  "error_code": "PDV_REJECT",
  "rejection_reason": "Sarathi fraud gate blocked this transaction",
  "success": false,
  "trace_id": "9c04fa571a9be274"
}
```

✅

---

### Test 3 — Schema Violation — Zero Amount (HTTP 400)

Response:
```json
{
  "error_code": "SCHEMA_VIOLATION",
  "rejection_reason": "schema violation: field=amount reason=must be greater than 0",
  "success": false,
  "trace_id": ""
}
```

✅

---

### Test 4 — Schema Violation — Unknown Field (HTTP 400)

Response:
```json
{
  "error_code": "SCHEMA_VIOLATION",
  "rejection_reason": "schema violation: field=payload reason=malformed JSON or unknown fields: json: unknown field \"unknown_field\"",
  "success": false,
  "trace_id": ""
}
```

✅

---

### Test 5 — AKASHIC Trace Continuity

Request: `GET /api/akashic/trace?trace_id=3e9b3ef561e26247`

Response:
```json
{
  "entry": {
    "trace_id": "3e9b3ef561e26247",
    "tx_hash": "538b3f681bd756865808ae9a4cf606745ea179d5e2a3d25798930ca73c813c17",
    "execution_hash": "894ba736f6726e192c4f7d78caf62256ee0fe843f4f23d83d83426cdcd987a5a",
    "validation_hash": "894ba736f6726e192c4f7d78caf62256ee0fe843f4f23d83d83426cdcd987a5a",
    "replay_hash": "894ba736f6726e192c4f7d78caf62256ee0fe843f4f23d83d83426cdcd987a5a",
    "fraud_decision": "allow",
    "block_height": 1,
    "prev_hash": "genesis",
    "entry_hash": "70197701fd4e08fc5d35abe9df4e247475b8201dce881e4cdbdf033962e19672"
  },
  "success": true
}
```

trace_id from Test 1 found in AKASHIC — trace continuity proven end-to-end. ✅

---

### Test 6 — Reconstruction Proof

Request: `GET /api/akashic/reconstruct`

Response:
```json
{
  "result": {
    "total_entries": 5,
    "chain_intact": true,
    "final_state_root": "67c671b7336d1c0001a0b4c69fd7c674bc2287dc3f858bcfeb7f2700c7c7b37f",
    "verified": true,
    "message": "reconstruction successful — lineage intact"
  },
  "success": true
}
```

✅

---

### Test 7 — Chain Integrity

Request: `GET /api/tantra/chain-integrity`

Response:
```json
{
  "intact": true,
  "message": "truth store chain is intact — no tampering detected",
  "success": true
}
```

✅

---

## 11. What Changed vs Previous Phase

| Area | Previous Phase | This Phase |
|---|---|---|
| Schema | Free-form JSON, no version | Versioned `v1` contract, unknown fields rejected |
| PDV layer | ValidationAgent called fraud service | ValidationAgent is pure hash only — fraud separated |
| Fraud layer | Mixed inside PDV | Separate `FraudGate` called only after PDV PASS |
| Timestamp in hash | Included — broke replay | Excluded via `deterministicZone` struct |
| Truth store | Local JSONL only | Chain-linked + dual-write to remote bucket |
| AKASHIC | Did not exist | Full lineage store with chain links + reconstruction |
| Observability | Generic error strings | Structured `error_code` + `rejection_reason` on every failure |
| Trace continuity | trace_id could be lost | trace_id propagated explicitly through every layer |
| Bypass path | `/api/admin/submit-transaction` bypassed PDV | Now routes through full PDV pipeline |
| P2P binding | Used link-local address — failed on Windows | Fixed to always bind on 127.0.0.1 |

---

## 12. Current Convergence Gaps

| Gap | Status | Notes |
|---|---|---|
| Multi-node PDV | Not yet | All agents share process locality — true distributed PDV requires separate node processes |
| KSML / CET contracts | Not yet | Awaiting Tanvi's upstream canonical execution structure |
| Branching state model | Not yet | Linear progression only — no parent_state_hash or alternate lineage |
| L2 execution chain | Not yet | Downstream of PDV — future phase |
| Peer testing validation | Pending | Sheet to be shared upon completion |

---

## Key Files

| File | Role |
|---|---|
| `core/relay-chain/schema/contract.go` | Phase 3 — versioned schema, canonical serialization, unknown field rejection |
| `core/relay-chain/enforcement/tantra.go` | PDV layer (pure determinism) + FraudGate (governance, separate) |
| `core/relay-chain/akashic/akashic.go` | Phase 4/6 — AKASHIC lineage store, chain-linked, reconstruction proof |
| `core/relay-chain/truthstore/truthstore.go` | Phase 5 — Bucket store, chain-linked, dual-write |
| `core/relay-chain/api/server.go` | Full convergence flow in `handleRelaySubmit` |
| `core/relay-chain/fraud/fraud.go` | Sarathi/DGIC fraud service binary (port 9090) |
| `core/relay-chain/chain/p2p.go` | Fixed to bind on 127.0.0.1 |
| `akashic_lineage.jsonl` | Runtime AKASHIC lineage (created on first accepted tx) |
| `tantra_truth.jsonl` | Runtime Bucket truth store |

---

## How to Run

**Terminal 1 — Fraud Service (port 9090):**
```
cd core/relay-chain
go run cmd/fraud-service/main.go
```

**Terminal 2 — Blockchain Node (port 8080):**
```
cd core/relay-chain
go run cmd/relay/main.go 3000
```

**Verification endpoints:**
```
GET  http://localhost:8080/api/akashic/reconstruct
GET  http://localhost:8080/api/akashic/lineage
GET  http://localhost:8080/api/akashic/trace?trace_id=<id>
GET  http://localhost:8080/api/tantra/chain-integrity
GET  http://localhost:8080/api/tantra/records
GET  http://localhost:8080/api/tantra/verify?tx_hash=<hash>
POST http://localhost:8080/api/relay/submit
```
