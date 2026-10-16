# BRIDGE SECURITY & SETTLEMENT ARCHITECTURE

**Document Version:** 1.0.0  
**Status:** Code-Verified Bridge Architecture  
**Audit Standard:** Strict Technical Proof (Rules 0.1 – 0.4)

---

## 1. Cross-Chain Settlement Lifecycle

The VAJRA Bridge system (`bridge-sdk` and `core/relay-chain/bridge`) operates a bidirectional **Lock-and-Mint / Burn-and-Release** settlement model across connected networks.

```
   Source Network (EVM / Solana)                   VAJRA Relay Chain
   ┌───────────────────────────┐                   ┌───────────────────────────┐
   │ User deposits Asset       │                   │ Relayer calls             │
   │ (Lock / Burn Event)       │                   │ POST /api/relay/submit    │
   └─────────────┬─────────────┘                   └─────────────▲─────────────┘
                 │                                               │
                 ▼                                               │
   ┌─────────────────────────────────────────────────────────────┴─────────────┐
   │                       Bridge SDK Daemon (:8084)                           │
   │  1. Event Detection: RPC polling & WebSocket listener                     │
   │  2. Finality Confirmation: 12 EVM blocks / 32 Solana slots                │
   │  3. Replay Protection: Checks & records TxHash in bbolt database          │
   │  4. Cryptographic Attestation: Generates threshold signature payload     │
   │  5. Circuit Breaker: Validates error thresholds are not tripped           │
   │  6. Relay Dispatch: Forwards payload to Relay Chain runtime               │
   └───────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Security Controls & Defenses

### 2.1 Replay Attack Mitigation
* **Dual-Layer Deduplication:**
  1. **External Layer:** The `bridge-sdk` stores every processed source transaction hash inside a persistent `bbolt` database (`bridge_v4.db`). Duplicate transactions are rejected immediately before signature generation.
  2. **Internal Layer:** The Relay Chain verifies incoming transactions against `nonce_ledger.jsonl`. Submissions containing previously consumed nonces fail with `NONCE_REPLAY`.

### 2.2 Circuit Breaker Implementation (`bridge-sdk/bridge_sdk.go:27`)
* **State Machine:** Maintained per chain (`IsOpen: bool`).
* **Trip Logic:** If consecutive RPC network failures or signature mismatches exceed `Threshold` (default: 5), the circuit breaker trips (`IsOpen = true`), suspending all automated bridging for that asset route.
* **Administrative Reset:** Breaker status can be inspected via `/health` and manually enabled via authenticated administrative controls.

### 2.3 Confirmation Depth & Finality
* **Polygon / Ethereum Testnets:** 12 confirmed blocks required before transaction processing.
* **Solana Devnet:** 32 confirmed slots.
* **VAJRA Relay Chain:** 1 block confirmation (instant local state execution upon block inclusion).

---

## 3. Cryptographic Settlement & Audit Trail

Every bridged asset event generates an immutable record in the TANTRA **Bucket TruthStore** (`tantra_truth.jsonl`) containing:
* `trace_id`: Unique end-to-end correlation identifier.
* `source_chain`: Network identifier (e.g. `polygon-amoy`, `ethereum-sepolia`).
* `source_tx_hash`: External blockchain transaction hash.
* `target_address`: Credited VAJRA account.
* `amount`: Transferred token units.
* `prev_hash` & `entry_hash`: Cryptographic SHA-256 chain linking ensuring historical immutability.
