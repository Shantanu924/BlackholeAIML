# TANTRA ↔ VAJRA CONSTITUTIONAL CONTRACTS & INTEGRATION MATRIX

**Document Version:** 1.0.0  
**Status:** Canonical Trust Layer Integration Specification  
**Audit Standard:** Code-Verified Boundaries (Rules 0.1 – 0.4)

---

## 1. Constitutional Authority Boundary Declaration

```
  TMS (Strategy)      GC (Governance / Legitimacy)      MDU (Schemas / Semantics)
       │                          │                                  │
       └──────────────────────────┼──────────────────────────────────┘
                                  │
                                  ▼
                     TANTRA Constitutional Runtime
                                  │
                                  ▼
                  VAJRA Trust Layer & Determinism
                      • Schema Validation
                      • btcec Signature Verification
                      • 3-Agent Parallel PDV
                      • PDV Equality Gate
                      • Governance (Sarathi) Gate
                      • Blockchain Ledger Write
                      • TruthStore (Bucket) Write
                      • AKASHIC Lineage Append
                      • KarmaChain Replication
                                  │
                   ┌──────────────┴──────────────┐
                   ▼                             ▼
        Bucket (TruthStore)             AKASHIC Lineage
        (tantra_truth.jsonl)         (akashic_lineage.jsonl)
                   │                             │
                   └──────────────┬──────────────┘
                                  ▼
                       Replay & InsightFlow
```

### 1.1 Non-Negotiable Boundary Rules (`constitution/constitution.go`)
1. **Replay Equality $\ne$ Governance Legitimacy:** Hash agreement across PDV agents proves mathematical correctness, not policy legitimacy. Legitimacy is decided by Sarathi/GC.
2. **Observability $\ne$ Execution Authority:** Reading telemetry via InsightFlow does not grant write permissions.
3. **Replay Reconstruction $\ne$ Semantic Truth:** Lineage reconstruction verifies Merkle chain integrity, not business domain validity.
4. **Deterministic Execution $\ne$ Sovereign Correctness:** Sovereignty requires alignment with external governance and constitutional contracts.

---

## 2. Canonical API & Event Contracts

### 2.1 Execution Request Contract (`core/relay-chain/runtime/runtime.go:44`)
Every transaction processed by the Trust Layer must conform to `ExecutionRequest`:
```go
type ExecutionRequest struct {
    Contract     *schema.TxContract
    Blockchain   BlockchainWriter
    TruthStore   *truthstore.Store
    AkashicStore *akashic.Store
}
```

### 2.2 Canonical Execution Result Schema
```json
{
  "trace_id": "trace_1787184000_abc123",
  "tx_hash": "0x4f8a9c...",
  "execution_hash": "sha256:e3b0c44...",
  "validation_hash": "sha256:e3b0c44...",
  "replay_hash": "sha256:e3b0c44...",
  "signature_valid": true,
  "payload_hash": "sha256:7a8b9c...",
  "fraud_decision": "ALLOW",
  "block_height": 1042,
  "schema_version": "1.0.0",
  "allowed": true,
  "submitted_at": 1787184000
}
```

### 2.3 Failure Stages & Error Codes
| Failure Stage | Error Code | Trigger Condition |
|---|---|---|
| `STAGE_SCHEMA` | `SCHEMA_NIL` / `SCHEMA_INVALID` | Malformed or unregistered transaction schema. |
| `STAGE_SIGNATURE` | `SIGNATURE_INVALID` | ECDSA signature verification failure or missing public key. |
| `STAGE_PDV` | `PDV_MISMATCH` | ExecutionHash, ValidationHash, or ReplayHash divergence across agents. |
| `STAGE_GOVERNANCE` | `SARATHI_REJECT` | Fraud score exceeds safety threshold or address is frozen. |
| `STAGE_BLOCKCHAIN` | `NONCE_REPLAY` / `INSUFFICIENT_FUNDS` | Duplicate sequence nonce or balance deficit during ledger commit. |

---

## 3. Storage & Lineage Contracts

1. **Bucket State (`tantra_truth.jsonl`):** Append-only hash-linked record storing canonical truth logs.
2. **AKASHIC Lineage (`akashic_lineage.jsonl`):** Deterministic execution trace logging for historical state reconstruction.
3. **KarmaChain Replication (`/api/karmachain/replicate`):** Asynchronous broadcast of state lineage records to peer cluster nodes.
