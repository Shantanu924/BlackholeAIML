# Wallet TANTRA Convergence Review Packet
**Author:** Prakash Kumar
**Task:** Phase 2 — Canonical Wallet Runtime Integration (BlackHole Blockchain – TANTRA)
**Date:** 09 May 2026
**Phases Delivered:** 2A + 2B + 2C + 2D + 2E + 2F + 2G — ALL COMPLETE

---

## 1. Entry Point

The wallet's canonical entry point into TANTRA is:

```
services/wallet/tantra/tantra.go → Runtime.Execute(IntentRequest)
```

This calls `POST http://localhost:8080/api/relay/submit` — the ONLY submission path.

No direct blockchain calls. No P2P transaction injection. No admin endpoint calls.

---

## 2. Canonical Wallet Execution Flow

```
Wallet User Action
      │
      ▼
tantra.Runtime.Execute(IntentRequest)
      │
      ▼
┌─────────────────────────────────────────────────────────────┐
│  BuildIntent()  — Phase 2A                                  │
│  • Validate constitutional boundaries                       │
│  • Generate immutable trace_id (SHA-256[:16])               │
│  • Get deterministic nonce from NonceRegistry               │
│  • Build canonical schema v1 IntentContract                 │
│  • Compute CanonicalHash (timestamp excluded)               │
└─────────────────────────────────────────────────────────────┘
      │
      ▼
┌─────────────────────────────────────────────────────────────┐
│  Submit()  — Phase 2B                                       │
│  • Nonce replay check (Phase 2C)                            │
│  • POST /api/relay/submit with schema v1 contract           │
│  • Receives SubmissionResult with trace_id + all hashes     │
└─────────────────────────────────────────────────────────────┘
      │
      ▼
┌─────────────────────────────────────────────────────────────┐
│  TANTRA Runtime (relay-chain)                               │
│  Schema → PDV → Governance → Blockchain → Bucket → AKASHIC  │
└─────────────────────────────────────────────────────────────┘
      │
      ▼
SubmissionResult: trace_id + tx_id + execution_hash + block_height
```

---

## 3. Deterministic Signing Flow

```go
// 1. Build canonical payload (timestamp excluded — replay-safe)
payload := tantra.CanonicalPayload(traceID, txType, from, to, tokenID,
                                    signature, amount, fee, nonce)

// 2. Compute canonical hash
hash := tantra.CanonicalHash(payload)  // SHA-256 of canonical JSON

// 3. Sign the canonical hash with wallet private key (btcec/ecdsa)
signature := ecdsa.Sign(privKey, hash)

// 4. Include signature in IntentContract.Signature field
```

The canonical payload excludes timestamp — identical inputs always produce identical hashes across environments. This is the replay-safe guarantee.

---

## 4. Replay Flow

```
Same IntentRequest (same from/to/amount/token)
      │
      ▼
BuildIntent() → different nonce (NonceRegistry increments)
      │
      ▼
Different IntentContract → different canonical hash → different trace_id
      │
      ▼
PDV produces different hashes → different AKASHIC entry
```

Replay attack attempt (same nonce injected externally):
```
CheckReplay(address, nonce) → true → [WALLET][NONCE_REPLAY] logged
```

Replay verification endpoint:
```
POST /api/replay/verify  ← same payload → deterministic=true (3 identical hashes)
```

---

## 5. Failure Cases

| Failure | Error Code | Source |
|---|---|---|
| Empty from address | `WALLET_VIOLATION` | `BuildIntent()` |
| Empty to address | `WALLET_VIOLATION` | `BuildIntent()` |
| Zero amount | `WALLET_VIOLATION` | `BuildIntent()` |
| Empty token_id | `WALLET_VIOLATION` | `BuildIntent()` |
| Missing schema_version | `SCHEMA_VIOLATION` | TANTRA relay |
| Unknown field in payload | `SCHEMA_VIOLATION` | TANTRA relay |
| PDV hash mismatch | `PDV_REJECT` | TANTRA relay |
| Fraud blocked | `PDV_REJECT` | TANTRA relay |
| Blockchain write fail | `BLOCKCHAIN_REJECT` | TANTRA relay |
| Network error | `WALLET_NETWORK_ERROR` | `Submit()` |
| Nonce replay detected | logged | `Submit()` |

---

## 6. Constitutional Boundaries

### Wallet MAY:
- `BuildIntent()` — create a canonical schema v1 transaction intent
- `Submit()` — POST to `/api/relay/submit` — the ONLY submission path
- `Execute()` — BuildIntent + Submit in one call
- `LookupTrace()` — query trace continuity
- `LookupTransaction()` — query transaction status
- `LookupNonce()` — query current nonce for an address
- `VerifyReplay()` — prove determinism of a payload
- `ConvergenceProof()` — verify full system convergence

### Wallet MAY NOT:
- Call `/api/admin/add-tokens` — REMOVED from `blockchain_client.go`
- Call `/api/admin/submit-transaction` — REMOVED
- Send raw `chain.Transaction` via P2P — REMOVED (`sendTransactionToNetwork` deleted)
- Call `sendTransactionViaHTTP` — REMOVED (was calling admin endpoints)
- Define whether a transaction is legitimate — that is Governance's role
- Bypass PDV equality enforcement
- Write to Bucket or AKASHIC directly
- Accumulate hidden execution authority

### Removed Bypass Functions:
| Function | Was Doing | Status |
|---|---|---|
| `sendTransactionToNetwork` | P2P raw tx injection | ✅ REMOVED |
| `sendTransactionViaHTTP` | Called `/api/admin/add-tokens` | ✅ REMOVED |
| `TransferTokens` (old) | Built raw `chain.Transaction` | ✅ REPLACED with TANTRA routing |
| `StakeTokens` (old) | Built raw `chain.Transaction` | ✅ REPLACED with TANTRA routing |

---

## 7. Observability Endpoints

All wallet-originated transactions are fully observable:

| Endpoint | Purpose |
|---|---|
| `GET /api/trace/verify?trace_id=<id>` | Trace continuity across Bucket+AKASHIC |
| `GET /api/akashic/trace?trace_id=<id>` | AKASHIC lineage entry |
| `GET /api/tantra/verify?tx_hash=<hash>` | On-chain verification |
| `GET /api/tantra/records` | All Bucket records |
| `POST /api/replay/verify` | Determinism proof |
| `GET /api/convergence/proof` | Full system convergence |

Wallet-side observability via `tantra.Runtime`:
```go
rt := tantra.NewRuntime("http://localhost:8080")
rt.LookupTrace(traceID)        // trace continuity
rt.LookupTransaction(txHash)   // on-chain status
rt.LookupNonce(address)        // current nonce
rt.VerifyReplay(intent)        // determinism proof
rt.ConvergenceProof()          // full convergence
```

---

## 8. Convergence Proof

Full live proof flow (Phase 2G):

```go
rt := tantra.NewRuntime("http://localhost:8080")

// Step 1: Create and submit intent
result, _ := rt.Execute(tantra.IntentRequest{
    From:    "alice",
    To:      "bob",
    Amount:  100,
    TokenID: "BHIV",
    Type:    "token_transfer",
})
// result.TraceID = "3e9b3ef561e26247"
// result.ExecutionHash == result.ValidationHash == result.ReplayHash

// Step 2: Verify trace continuity
trace, _ := rt.LookupTrace(result.TraceID)
// trace["continuous"] = true (found in Bucket AND AKASHIC)

// Step 3: Verify replay determinism
intent, _ := rt.BuildIntent(...)
replay, _ := rt.VerifyReplay(intent)
// replay["deterministic"] = true

// Step 4: Full convergence proof
proof, _ := rt.ConvergenceProof()
// proof["converged"] = true
```

---

## Key Files

| File | Phase | Role |
|---|---|---|
| `services/wallet/tantra/tantra.go` | 2A-2G | Canonical wallet TANTRA runtime |
| `services/wallet/wallet/blockchain_client.go` | 2B | Bypass functions removed, TANTRA routing |
| `services/wallet/transaction/transaction.go` | 2A | Signing utilities |

---

## Testing Packet for Vinayak Tiwari

### Replay Tests
```go
rt := tantra.NewRuntime("http://localhost:8080")
// Send same intent twice — second gets different nonce → different hash
r1, _ := rt.Execute(IntentRequest{From:"alice", To:"bob", Amount:100, TokenID:"BHIV"})
r2, _ := rt.Execute(IntentRequest{From:"alice", To:"bob", Amount:100, TokenID:"BHIV"})
// r1.ExecutionHash != r2.ExecutionHash (different nonce)
```

### Duplicate Nonce Tests
```go
nr := tantra.NewNonceRegistry()
n1 := nr.NextNonce("alice")  // returns 1
n2 := nr.NextNonce("alice")  // returns 2
replay := nr.CheckReplay("alice", n1)  // returns true — already used
```

### Malformed Schema Tests
```go
// Missing schema_version → SCHEMA_VIOLATION
// Unknown field → SCHEMA_VIOLATION
// Zero amount → WALLET_VIOLATION before reaching relay
```

### Bypass Attempt Tests
```go
// All of these are now compile errors — functions removed:
// client.sendTransactionToNetwork(tx)  → undefined
// client.sendTransactionViaHTTP(tx)    → undefined
```

### Trace Continuity Tests
```go
result, _ := rt.Execute(...)
trace, _ := rt.LookupTrace(result.TraceID)
// trace["continuous"] must be true
// trace["layers"]["bucket"]["found"] must be true
// trace["layers"]["akashic"]["found"] must be true
```

### Reconstruction Tests
```go
proof, _ := rt.ConvergenceProof()
// proof["converged"] must be true
// proof["proof"]["akashic_reconstruction"]["verified"] must be true
// proof["proof"]["bucket_chain"]["intact"] must be true
```

---

## Current Gaps

| Gap | Notes |
|---|---|
| Signature verification in PDV | Wallet signs with btcec/ecdsa; relay does not yet verify signature — future phase |
| KSML/CET upstream mapping | Awaiting Raj Prajapati |
| Multi-wallet nonce persistence | NonceRegistry is in-memory; restart resets nonces |
