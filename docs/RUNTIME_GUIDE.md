# VAJRA Runtime Guide

**Domain:** `https://vajra.blackholeinfiverse.com`  
**Last Updated:** 2026-08-07

---

## Constitutional Runtime Position

VAJRA occupies the **Trust Layer** of the BHIV Living Organism as a permanent Constitutional Runtime Participant.

```
BHIV Living Organism
└── Trust Layer
    ├── VAJRA Blockchain (PDV, SigVerify, Nonce, TruthStore, AKASHIC)
    ├── Wallet           (Intent, KeyMgmt, Signing, Auth)
    ├── Bridge SDK       (CrossChain, ReplayProtection, Events)
    └── Cyber Defence    (FraudDetection, VelocityCheck, Blocking)
```

---

## TANTRA Execution Pipeline

Every transaction through `POST /api/relay/submit` executes this canonical 9-step pipeline:

```
Step 1: Schema Validation      — schema_version=v1, required fields
Step 2: Trace Context Init     — trace_id injected, immutable after this point
Step 3: Nonce Governance       — per-address nonce store, replay rejected
Step 4: Signature Verification — btcec secp256k1 ECDSA over canonical payload hash
Step 5: Distributed PDV        — ExecutionAgent + ValidationAgent + ReplayAgent
Step 6: PDV Equality Gate      — execution_hash == validation_hash == replay_hash
Step 7: Fraud Gate (Sarathi)   — POST to blackhole-fraud:9090/api/fraud/check
Step 8: Blockchain Write       — token balance update, block append
Step 9: Truth Store + AKASHIC  — immutable append-only audit trail
```

Any step failure = hard reject. No partial execution.

---

## Canonical Payload Hash

The signature covers the **deterministic zone** — timestamp excluded:

```json
{
  "trace_id":  "...",
  "type":      "token_transfer",
  "from":      "<pubkey-hex>",
  "to":        "...",
  "amount":    100,
  "token_id":  "BHIV",
  "fee":       0,
  "nonce":     1,
  "signature": ""
}
```

SHA-256 of this JSON = `payload_hash`. Wallet signs this hash with btcec secp256k1 private key.

---

## PDV Equality Gate

Three agents independently compute `deterministicHash(payload)`:

- `ExecutionAgent`  → `execution_hash`
- `ValidationAgent` → `validation_hash`
- `ReplayAgent`     → `replay_hash`

All three must be identical. Any divergence = `DISTRIBUTED_PDV_REJECT`.

Current mode: single-process fallback (`PDV_STRICT_MODE=false`). All three agents run inside the same process — hash equality is guaranteed. Separate container deployment is a future infrastructure task.

---

## Truth Store (TANTRA)

Every accepted transaction is appended to `tantra_truth.jsonl` — an append-only file on the blockchain node:

```json
{
  "trace_id":        "phase4evidence01",
  "tx_hash":         "8dd91efe...",
  "execution_hash":  "15f38dba...",
  "validation_hash": "15f38dba...",
  "replay_hash":     "15f38dba...",
  "fraud_decision":  "allow",
  "timestamp":       1786101903,
  "prev_hash":       "genesis",
  "entry_hash":      "b39fe5e7..."
}
```

Verify via: `GET /api/tantra/verify?tx_hash=<id>`

---

## AKASHIC Lineage

Every transaction produces 2 chained AKASHIC entries — forming an immutable linked list:

```
genesis
  └── entry_hash_1 (aaa429ad...)
        └── entry_hash_2 (9d5ec979...)
```

Each entry's `prev_hash` = previous entry's `entry_hash`. This enables deterministic replay reconstruction.

Query via: `GET /api/akashic/lineage?tx_hash=<id>`

---

## Nonce Governance

- Each address has a monotonically increasing nonce
- Nonce is stored in a persistent BoltDB store — survives restarts
- Duplicate nonce from same address = `NONCE_REPLAY` hard reject
- Cross-node nonce coordination: not yet deployed (single-node mode)

---

## Fraud Gate (Sarathi)

After PDV equality, every transaction is checked against the fraud service:

```
POST http://blackhole-fraud:9090/api/fraud/check
{
  "from": "...", "to": "...", "amount": 100, "token_id": "BHIV"
}
```

Response `allow` → proceed. Response `block` → `FRAUD_BLOCK` hard reject.

`SARATHI_FAIL_CLOSED=false` — if fraud service is unreachable, transaction is allowed (fail-open). Set to `true` for strict production mode.

---

## Trace Continuity

`trace_id` is injected at Step 2 and must remain identical through all pipeline stages. Any drift = `TRACE_BREAK` hard reject. This ensures full end-to-end traceability of every transaction.

---

## Authority Boundaries

| Service          | Owns                                                    | Does NOT Own                                      |
|------------------|---------------------------------------------------------|---------------------------------------------------|
| Blockchain Node  | PDV_EQUALITY, SIG_VERIFY, NONCE_GOVERNANCE, TRUTH_STORE | GOVERNANCE_LEGITIMACY, SEMANTIC_TRUTH             |
| Wallet           | WALLET_INTENT, KEY_MANAGEMENT, SIGNING, AUTH            | PDV_EQUALITY, BLOCKCHAIN_WRITE, FRAUD_DETECTION   |
| Bridge SDK       | CROSS_CHAIN, REPLAY_PROTECTION, EVENTS, CIRCUIT_BREAKER | PDV_EQUALITY, FRAUD_DETECTION, NONCE_GOVERNANCE   |
| Fraud/Cyber Def  | FRAUD_DETECTION, VELOCITY_CHECK, BLOCKING               | PDV_EQUALITY, BLOCKCHAIN_WRITE, NONCE_GOVERNANCE  |
