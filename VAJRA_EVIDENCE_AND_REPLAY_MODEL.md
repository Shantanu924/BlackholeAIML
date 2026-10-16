# VAJRA EVIDENCE & REPLAY VERIFICATION MODEL

**Document Version:** 1.0.0  
**Status:** Code-Verified Execution & Replay Pipeline  
**Audit Standard:** Ground-Truth Proof (Rules 0.1 – 0.4)

---

## 1. Fundamental Conceptual Distinctions

```
 Execution Evidence
        ≠
 Cryptographic Attestation
        ≠
 Bucket State
        ≠
 Deterministic Replay
        ≠
 Governance Legitimacy
```

| Layer / Concept | What It Proves | What It Does NOT Prove |
|---|---|---|
| **Execution Evidence** | An execution attempt occurred with specific input parameters at a given timestamp. | Does not prove the transaction was cryptographically valid or committed to state. |
| **Cryptographic Attestation** | The payload was signed by the holder of a private key matching the public key. | Does not prove the transaction was economically legitimate or compliant with governance policy. |
| **Deterministic Replay** | Two independent nodes executed the identical code and arrived at the exact same state root ($H_A \equiv H_B$). | Does not prove the transaction was authorized by governance or should have been permitted. |
| **Bucket State** | The canonical, append-only historical log of committed truth records. | Does not grant execution authority or override live consensus. |
| **Governance Legitimacy** | The transaction complies with human/AI ecosystem policies, sanctions, and economic rules. | Enforced by Sarathi / GC — external to mathematical determinism. |

---

## 2. 9-Stage Execution & Evidence Pipeline (`runtime/runtime.go`)

```
   1. Schema Validation (MDU contract check)
             │
             ▼
   2. Signature Verification (btcec/secp256k1 validation)
             │
             ▼
   3. Distributed PDV (Parallel execution across 3 independent agents)
             │
             ▼
   4. PDV Equality Gate (Enforces ExecutionHash == ValidationHash == ReplayHash)
             │
             ▼
   5. Governance Gate (Sarathi pre-flight fraud check via blackhole-fraud:9090)
             │
             ▼
   6. Blockchain Write (Mempool inclusion & LevelDB block commitment)
             │
             ▼
   7. Bucket Write (Appends immutable truth record to tantra_truth.jsonl)
             │
             ▼
   8. AKASHIC Lineage Append (Appends trace lineage to akashic_lineage.jsonl)
             │
             ▼
   9. KarmaChain Replication (Broadcasts lineage to peer cluster nodes)
```

---

## 3. Replay Verification Endpoints

1. **`POST /api/replay/equality`:** Verifies that a transaction replay produces the exact same execution hash and state diff across independent validator nodes.
2. **`POST /api/replay/verify`:** Reconstructs historical state from `akashic_lineage.jsonl` and validates Merkle root integrity.
3. **`GET /api/convergence/proof`:** Returns cryptographically signed convergence proofs confirming full runtime determinism.
4. **`GET /api/tantra/chain-integrity`:** Validates SHA-256 hash chaining across the entire Bucket TruthStore (`tantra_truth.jsonl`).
