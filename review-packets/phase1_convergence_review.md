# Phase 1 Convergence Review
**Author:** Prakash Kumar
**Task:** Canonical Deterministic Runtime Integration — BlackHole Blockchain TANTRA Convergence Phase 1
**Date:** 09 May 2026
**Phases Delivered:** 1A + 1B + 1C + 1D + 1E + 1F + 1G — ALL COMPLETE

---

## Phase 1A — Canonical Runtime Mapping ✅

### Every Active Transaction Entry Path

| Path | Endpoint | File | PDV Enforced? |
|---|---|---|---|
| Primary relay entry | `POST /api/relay/submit` | `api/server.go → handleRelaySubmit` | ✅ via `runtime.Execute()` |
| Legacy admin entry | `POST /api/admin/submit-transaction` | `api/server.go → submitTransaction` | ✅ via `enforcement.Enforce()` |
| Internal runtime | `runtime.Execute()` | `runtime/runtime.go` | ✅ IS the enforcement |
| Blockchain direct | `chain.ProcessTransaction()` | `chain/blockchain.go` | ⚠️ Internal only — block mining |
| Runtime blockchain | `chain.ProcessTransactionFromRuntime()` | `chain/blockchain.go` | ✅ Only callable after PDV PASS |

### All Bypass Vectors — Identified and Closed

| Bypass Vector | Status |
|---|---|
| `submitTransaction` direct `PendingTxs` append | ✅ CLOSED |
| Free-form JSON without schema version | ✅ CLOSED — `schema.ParseAndValidate` + `DisallowUnknownFields` |
| Fraud check inside PDV layer | ✅ FIXED — `FraudGate` is separate governance layer |
| Timestamp inside deterministic hash | ✅ FIXED — `deterministicZone` excludes timestamp |
| Bucket write without PDV PASS | ✅ CLOSED — enforced by `runtime.Execute()` step order |
| AKASHIC write without Bucket write | ✅ CLOSED — enforced by `runtime.Execute()` step order |
| Truth store local-only | ✅ MITIGATED — dual-write via `TANTRA_BUCKET_URL` |
| Direct enforcement calls in handler | ✅ CLOSED — handler now calls `runtime.Execute()` only |

### Runtime Flow Diagram

```
External Input
      │
      ▼
POST /api/relay/submit  (Port 8080)
      │
      ▼
┌─────────────────────────────────────────────────────────────┐
│  SCHEMA LAYER  (schema/contract.go)                         │
│  ParseAndValidate → reject unknown fields, wrong version    │
│  Error: SCHEMA_VIOLATION → HTTP 400                         │
└─────────────────────────────────────────────────────────────┘
      │ schema valid + trace.Context created (Phase 1C)
      ▼
┌─────────────────────────────────────────────────────────────┐
│  runtime.Execute()  (runtime/runtime.go)  — Phase 1D       │
│  ├── PDV: ExecutionAgent + ValidationAgent + ReplayAgent    │
│  ├── EQUALITY GATE: exec == val == replay                   │
│  ├── GOVERNANCE: FraudGate (Sarathi/DGIC, port 9090)        │
│  ├── BLOCKCHAIN: ProcessTransactionFromRuntime              │
│  ├── BUCKET: truthstore.Append (ONLY after blockchain)      │
│  └── AKASHIC: akashic.Append (ONLY after bucket)           │
└─────────────────────────────────────────────────────────────┘
      │ trace.Context.AssertContinuity() (Phase 1C)
      ▼
HTTP 200 — ExecutionResult with all hashes + trace_id
```

---

## Phase 1B — Canonical Execution Runtime ✅

**Package:** `core/relay-chain/runtime/runtime.go`

- `Execute(ExecutionRequest)` — the ONE canonical entry point
- 7-step pipeline enforced in strict sequential order
- `BlockchainWriter` interface — decouples runtime from blockchain
- `ProcessTransactionFromRuntime` — only blockchain write path from runtime
- Every failure returns structured `ExecutionResult` — no silent failures
- `trace_id` injected once, propagated through every step

**Failure Stage Mapping:**

| Stage | Error Code | HTTP |
|---|---|---|
| `SCHEMA` | `SCHEMA_VIOLATION` | 400 |
| `PDV` | `PDV_REJECT` | 403 |
| `GOVERNANCE` | `GOVERNANCE_REJECT` | 403 |
| `BLOCKCHAIN` | `BLOCKCHAIN_REJECT` | 422 |
| `BUCKET` | `BUCKET_WRITE_WARN` | 200 (non-fatal) |
| `AKASHIC` | logged only | 200 (non-fatal) |

---

## Phase 1C — Trace Continuity Enforcement ✅

**Package:** `core/relay-chain/trace/trace.go`

- `trace.Context` — immutable trace_id carrier
- `Inject(traceID)` — sets once, rejects any subsequent change with `TraceBreakError`
- `AssertContinuity(traceID, stage)` — verifies ID has not drifted
- `LogStage(stage, detail)` — structured log with trace_id on every line
- `TraceBreakError` — carries `Expected`, `Got`, `Stage`

**In `handleRelaySubmit`:**
- `tc := trace.New(contract.TraceID)` at entry
- After `runtime.Execute()`: `tc.Inject(execResult.TraceID)` asserts no drift
- Drift → HTTP 500 `TRACE_BREAK` — hard fail

**Trace Continuity Verification Endpoint:**
```
GET /api/trace/verify?trace_id=<id>
```
Checks trace_id in Bucket AND AKASHIC. Returns `continuous: true` only if found in both.

---

## Phase 1D — Deterministic Contract Enforcement ✅

**`handleRelaySubmit` now calls `runtime.Execute()` exclusively.**

No direct `enforcement.Enforce()`, no direct `chain.Transaction`, no direct `truthstore.Append()`, no direct `akashic.Append()` in the handler. All execution flows through the canonical runtime.

**Schema contract:** `schema/contract.go`
- `schema_version: "v1"` required
- Unknown fields → `SCHEMA_VIOLATION`
- `DisallowUnknownFields()` on JSON decoder
- Canonical serialization with timestamp excluded from hash zone

---

## Phase 1E — Bucket + AKASHIC Convergence ✅

Write order enforced by `runtime.Execute()` step sequence — not by convention, by code:

```go
// Step 5: Blockchain write — only after PDV+Governance PASS
txHash, blockHeight, err := req.Blockchain.ProcessTransactionFromRuntime(...)

// Step 6: Bucket write — only after blockchain write success
req.TruthStore.Append(truthstore.Record{...})

// Step 7: AKASHIC append — only after Bucket write
req.AkashicStore.Append(akashic.LineageEntry{...})
```

No orphan writes possible — if blockchain write fails, neither Bucket nor AKASHIC is written.

**Chain integrity:** Both stores use `prev_hash → entry_hash` chain linking. Tampering breaks the chain and is detected by:
- `GET /api/tantra/chain-integrity` — Bucket chain
- `GET /api/akashic/reconstruct` — AKASHIC chain + PDV equality within each entry

---

## Phase 1F — Replay + Observability Lock ✅

**Replay verification endpoint:**
```
POST /api/replay/verify
Body: same schema v1 payload
```

Runs `ExecutionAgent`, `ValidationAgent`, `ReplayAgent` three times on the same payload. All three hashes must be identical. Proves determinism.

Response:
```json
{
  "deterministic": true,
  "run_1_hash": "894ba736...",
  "run_2_hash": "894ba736...",
  "run_3_hash": "894ba736...",
  "message": "replay determinism confirmed"
}
```

**All failures are observable and structured:**

| Failure | Error Code | HTTP | Log Tag |
|---|---|---|---|
| Malformed JSON | `SCHEMA_VIOLATION` | 400 | `[SCHEMA][REJECT]` |
| Unknown field | `SCHEMA_VIOLATION` | 400 | `[SCHEMA][REJECT]` |
| Missing schema_version | `SCHEMA_VIOLATION` | 400 | `[SCHEMA][REJECT]` |
| Zero amount | `SCHEMA_VIOLATION` | 400 | `[SCHEMA][REJECT]` |
| PDV hash mismatch | `PDV_REJECT` | 403 | `[PDV][REJECT]` |
| Fraud blocked | `PDV_REJECT` | 403 | `[Sarathi][REJECT]` |
| Blockchain fail | `BLOCKCHAIN_REJECT` | 422 | `[BLOCKCHAIN][REJECT]` |
| Trace drift | `TRACE_BREAK` | 500 | `[TRACE][BREAK]` |
| Bucket tampered | — | 409 | `[truthstore][TAMPER]` |
| AKASHIC tampered | — | 409 | `[AKASHIC][TAMPER]` |

No silent failures. Every rejection carries `error_code`, `rejection_reason`, `trace_id`.

---

## Phase 1G — Convergence Proof ✅

**Full convergence proof endpoint:**
```
GET /api/convergence/proof
```

Returns a single response proving the entire system is converged:

```json
{
  "success": true,
  "converged": true,
  "timestamp": 1746721834,
  "proof": {
    "bucket_chain": {
      "intact": true,
      "record_count": 5
    },
    "akashic_reconstruction": {
      "verified": true,
      "chain_intact": true,
      "total_entries": 5,
      "final_state_root": "67c671b7...",
      "message": "reconstruction successful — lineage intact"
    },
    "blockchain": {
      "block_height": 5,
      "pending_txs": 0
    },
    "runtime": {
      "canonical_entry": "POST /api/relay/submit",
      "execution_path": "Schema → PDV → Governance → Blockchain → Bucket → AKASHIC",
      "trace_enforcement": "trace.Context — immutable after injection",
      "bypass_paths": 0,
      "schema_version": "v1"
    }
  }
}
```

---

## All Endpoints Delivered

| Endpoint | Phase | Purpose |
|---|---|---|
| `POST /api/relay/submit` | 1B/1D | Canonical transaction entry |
| `GET /api/trace/verify?trace_id=<id>` | 1C | Trace continuity across Bucket+AKASHIC |
| `POST /api/replay/verify` | 1F | Deterministic replay proof |
| `GET /api/convergence/proof` | 1G | Full system convergence proof |
| `GET /api/akashic/reconstruct` | 1E | AKASHIC lineage reconstruction |
| `GET /api/akashic/lineage` | 1E | All lineage entries |
| `GET /api/akashic/trace?trace_id=<id>` | 1C | Single entry by trace |
| `GET /api/tantra/chain-integrity` | 1E | Bucket chain tamper detection |
| `GET /api/tantra/records` | 1E | All Bucket records |
| `GET /api/tantra/verify?tx_hash=<hash>` | 1E | On-chain verification |

---

## Testing Deliverable for Vinayak Tiwari

### 5-Minute Execution Validation

**Start services:**
```
Terminal 1: cd core/relay-chain && go run cmd/fraud-service/main.go
Terminal 2: cd core/relay-chain && go run cmd/relay/main.go 3000
```

**Run in order — get fresh timestamp first:**
```powershell
$ts = [int][double]::Parse((Get-Date -UFormat %s))
```

**Test 1 — Successful execution:**
```json
POST http://localhost:8080/api/relay/submit
{
  "schema_version": "v1",
  "type": "token_transfer",
  "from": "alice",
  "to": "bob",
  "amount": 100,
  "token_id": "BHIV",
  "nonce": 1,
  "timestamp": <ts>
}
Expected: HTTP 200, execution_hash == validation_hash == replay_hash
```

**Test 2 — Schema rejection (unknown field):**
```json
POST http://localhost:8080/api/relay/submit
{ "schema_version": "v1", "type": "token_transfer", "from": "alice", "to": "bob", "amount": 100, "token_id": "BHIV", "nonce": 1, "timestamp": <ts>, "unknown_field": "x" }
Expected: HTTP 400, error_code=SCHEMA_VIOLATION
```

**Test 3 — Governance rejection:**
```json
POST http://localhost:8080/api/relay/submit
{ "schema_version": "v1", "type": "token_transfer", "from": "bad-actor", "to": "bob", "amount": 100, "token_id": "BHIV", "nonce": 1, "timestamp": <ts> }
Expected: HTTP 403, error_code=PDV_REJECT
```

**Test 4 — Trace continuity (use trace_id from Test 1):**
```
GET http://localhost:8080/api/trace/verify?trace_id=<trace_id>
Expected: continuous=true, found in both bucket and akashic
```

**Test 5 — Replay determinism:**
```json
POST http://localhost:8080/api/replay/verify
{ same body as Test 1 }
Expected: deterministic=true, all 3 hashes identical
```

**Test 6 — Reconstruction proof:**
```
GET http://localhost:8080/api/akashic/reconstruct
Expected: verified=true, chain_intact=true
```

**Test 7 — Full convergence proof:**
```
GET http://localhost:8080/api/convergence/proof
Expected: converged=true, all sub-proofs passing
```

### Failure Testing Checklist

| Test | Input | Expected error_code | HTTP |
|---|---|---|---|
| Missing schema_version | no schema_version | SCHEMA_VIOLATION | 400 |
| Unknown field | extra field | SCHEMA_VIOLATION | 400 |
| Zero amount | amount=0 | SCHEMA_VIOLATION | 400 |
| Blocked sender | from=bad-actor | PDV_REJECT | 403 |
| Stale timestamp | timestamp > 600s old | PDV_REJECT | 403 |
| Self transfer | from==to | PDV_REJECT | 403 |

### Replay Verification Steps

1. Send Test 1 — note `trace_id` and `execution_hash`
2. `GET /api/akashic/trace?trace_id=<id>` — verify same hashes in AKASHIC
3. `GET /api/tantra/records` — verify same hashes in Bucket
4. `POST /api/replay/verify` with same payload — verify `deterministic=true`
5. `GET /api/akashic/reconstruct` — verify `verified=true`
6. `GET /api/convergence/proof` — verify `converged=true`

### Trace Continuity Validation Steps

1. Send transaction — copy `trace_id` from response
2. Terminal 2 logs — every line `[SCHEMA][PASS]` → `[RUNTIME][COMPLETE]` carries same `trace_id`
3. `GET /api/trace/verify?trace_id=<id>` — `continuous=true`
4. `GET /api/akashic/trace?trace_id=<id>` — entry found
5. `GET /api/tantra/records` — record found with matching `trace_id`

---

## Key Files

| File | Phase | Role |
|---|---|---|
| `core/relay-chain/runtime/runtime.go` | 1B/1D | Canonical execution runtime |
| `core/relay-chain/trace/trace.go` | 1C | Immutable trace context |
| `core/relay-chain/schema/contract.go` | 1D | Versioned schema, canonical serialization |
| `core/relay-chain/enforcement/tantra.go` | 1B | PDV layer + FraudGate (separated) |
| `core/relay-chain/akashic/akashic.go` | 1E | AKASHIC lineage store + reconstruction |
| `core/relay-chain/truthstore/truthstore.go` | 1E | Bucket truth store + chain integrity |
| `core/relay-chain/chain/blockchain.go` | 1B | `ProcessTransactionFromRuntime` |
| `core/relay-chain/api/server.go` | 1C/1D/1F/1G | All endpoints |

---

## Current Convergence Gaps

| Gap | Status | Notes |
|---|---|---|
| KSML/CET upstream contract mapping | Pending | Awaiting Raj Prajapati |
| Multi-node PDV | Future | All agents share process locality |
| Branching state model | Future | Linear progression only |
| Peer testing sheet | Pending | Vinayak's sheet to be shared |
