# VAJRA Blockchain Transaction Lifecycle Flow

**Author / Contributor:** Ishan Shirode  
**Role:** VAJRA Go + Blockchain Engineering Apprenticeship and Production Support  
**Lead / Technical Direction:** Prakash Kumar  
**Target Codebase:** `BHIV-Engineering-Exchange/bhiv-VAJRA_Blockchain`  

---

## 1. End-to-End Flow Pipeline

The end-to-end lifecycle of a VAJRA transaction follows a strict 11-stage pipeline from wallet origin to RPC observation:

```
Wallet
  ↓
Transaction
  ↓
Validation
  ↓
Signature
  ↓
Nonce
  ↓
Runtime
  ↓
Consensus / Node
  ↓
State
  ↓
Evidence
  ↓
Replay
  ↓
API / RPC
```

---

## 2. Stage-by-Stage Architectural Analysis

### Stage 1: Wallet
* **Existing Implementation:** Keypair generator, balance tracker, web and CLI wallet wrappers.
* **Repository Location:** [`core/relay-chain/crypto/crypto.go`](file:///g:/VAJRA/core/relay-chain/crypto/crypto.go), [`ui/`](file:///g:/VAJRA/ui), [`start_wallet.bat`](file:///g:/VAJRA/start_wallet.bat)
* **API / Function:** `crypto.GenerateKeyPair()`, `GET /api/v1/wallet/balance?address={addr}`
* **Owner:** Core Dev / Ishan (support)
* **Test:** Key generation unit tests, UI balance retrieval tests.
* **Evidence:** Generated secp256k1/ECDSA key pairs and valid hex addresses.
* **Known Gap:** Native hardware wallet integrations (Ledger/Trezor) planned for later phase.

---

### Stage 2: Transaction
* **Existing Implementation:** Typed transaction structs covering transfers, token operations, staking, and contract calls.
* **Repository Location:** [`core/relay-chain/chain/transaction.go`](file:///g:/VAJRA/core/relay-chain/chain/transaction.go)
* **API / Function:** `NewTransaction(txType, from, to, amount, publicKey)`, `CalculateHash()`
* **Owner:** Prakash Kumar / Core Ledger
* **Test:** Struct serialization tests, hash collision checks.
* **Evidence:** SHA-256 transaction hash digests matching deterministic inputs.
* **Known Gap:** Dynamic EIP-1559 style fee estimation (currently static fee model).

---

### Stage 3: Validation
* **Existing Implementation:** Strict schema validation ensuring type safety, non-empty fields, positive amounts, and authorized tokens.
* **Repository Location:** [`core/relay-chain/schema/schema.go`](file:///g:/VAJRA/core/relay-chain/schema/schema.go), [`core/relay-chain/validation/`](file:///g:/VAJRA/core/relay-chain/validation)
* **API / Function:** `schema.ValidateTxContract(contract)`
* **Owner:** Core Runtime / Prakash Kumar
* **Test:** Schema boundary tests, negative amount rejections, malformed payload tests.
* **Evidence:** `StageSchema` rejection records with precise error codes.
* **Known Gap:** Enhanced schema validation error descriptions for client UI.

---

### Stage 4: Signature
* **Existing Implementation:** Asymmetric cryptographic signature validation using ECDSA / btcec.
* **Repository Location:** [`core/relay-chain/sigverify/`](file:///g:/VAJRA/core/relay-chain/sigverify), [`core/relay-chain/crypto/crypto.go`](file:///g:/VAJRA/core/relay-chain/crypto/crypto.go)
* **API / Function:** `tx.Sign(privateKey)`, `tx.Verify()`, `sigverify.VerifySignature(payload, sig, pubKey)`
* **Owner:** Core Security
* **Test:** `cmd/sig-proof`, invalid key signature rejection tests.
* **Evidence:** Valid boolean signature verification digests in runtime logs.
* **Known Gap:** Ed25519 multi-curve support across all transaction types.

---

### Stage 5: Nonce
* **Existing Implementation:** Global distributed and per-account sequential nonce tracking to prevent double-spending.
* **Repository Location:** [`core/relay-chain/noncecoord/`](file:///g:/VAJRA/core/relay-chain/noncecoord), [`core/relay-chain/noncestore/`](file:///g:/VAJRA/core/relay-chain/noncestore)
* **API / Function:** `noncecoord.ValidateAndConsume(account, nonce)`, `bc.AccountNonces[from]`
* **Owner:** Nonce Subsystem Lead
* **Test:** `cmd/nonce-proof`, `cmd/nonce-coord-proof`, concurrency replay tests.
* **Evidence:** `nonce_ledger.jsonl` append log records.
* **Known Gap:** In-memory sliding window cleanup tuning under extreme sustained load.

---

### Stage 6: Runtime
* **Existing Implementation:** 9-Stage deterministic execution engine orchestrating PDV, governance, blockchain write, and lineage.
* **Repository Location:** [`core/relay-chain/runtime/runtime.go`](file:///g:/VAJRA/core/relay-chain/runtime/runtime.go)
* **API / Function:** `runtime.Execute(req ExecutionRequest) ExecutionResult`
* **Owner:** Prakash Kumar
* **Test:** [`runtime/runtime_test.go`](file:///g:/VAJRA/core/relay-chain/runtime/runtime_test.go), `cmd/full-runtime-activation-e2e`
* **Evidence:** Detailed `ExecutionResult` with `TraceID`, `ExecutionHash`, `ValidationHash`, and `BlockHeight`.
* **Known Gap:** Configurable per-stage timeout context propagation.

---

### Stage 7: Consensus / Node
* **Existing Implementation:** Proof-of-Stake validator selection with stake snapshot weighting, mining loop, and P2P gossip.
* **Repository Location:** [`core/relay-chain/consensus/`](file:///g:/VAJRA/core/relay-chain/consensus), [`core/relay-chain/chain/p2p.go`](file:///g:/VAJRA/core/relay-chain/chain/p2p.go), [`core/relay-chain/cmd/relay/main.go`](file:///g:/VAJRA/core/relay-chain/cmd/relay/main.go)
* **API / Function:** `validator.SelectValidator()`, `bc.AddBlock(newBlock)`, `bc.P2PNode.BroadcastBlock(block)`
* **Owner:** Consensus & Network Lead
* **Test:** P2P peer identity tests, multi-node sync simulation.
* **Evidence:** Merkle roots, block hashes, and round consensus logs.
* **Known Gap:** Dynamic validator churn slashing automation under network partitions.

---

### Stage 8: State
* **Existing Implementation:** Account balance state, stake ledger, pending transactions pool, and smart contract storage.
* **Repository Location:** [`core/relay-chain/chain/blockchain.go`](file:///g:/VAJRA/core/relay-chain/chain/blockchain.go), [`core/relay-chain/chain/stakeledger.go`](file:///g:/VAJRA/core/relay-chain/chain/stakeledger.go)
* **API / Function:** `bc.Balances[addr]`, `bc.StakeLedger.UpdateStake(addr, amount)`
* **Owner:** Core Ledger
* **Test:** Balance debit/credit invariance tests, reward inflation tests.
* **Evidence:** Blockchain state snapshots saved to disk (`blockchain_state.json`).
* **Known Gap:** State trie (Merkle Patricia Trie) migration for light client proofs.

---

### Stage 9: Evidence & Provenance
* **Existing Implementation:** Immutable Akashic lineage logging and truth record persistence.
* **Repository Location:** [`core/relay-chain/akashic/`](file:///g:/VAJRA/core/relay-chain/akashic), [`core/relay-chain/truthstore/`](file:///g:/VAJRA/core/relay-chain/truthstore)
* **API / Function:** `akashic.Store.Append(entry)`, `truthstore.Store.Append(record)`
* **Owner:** Provenance Subsystem Lead
* **Test:** SHA-256 hash chaining verification, tamper detection tests.
* **Evidence:** [`akashic_lineage.jsonl`](file:///g:/VAJRA/core/relay-chain/akashic_lineage.jsonl), [`tantra_truth.jsonl`](file:///g:/VAJRA/core/relay-chain/tantra_truth.jsonl)
* **Known Gap:** Real-time WebSocket streaming of lineage events to external auditors.

---

### Stage 10: Replay
* **Existing Implementation:** Replay mitigation table, cryptographic proof engine, and lineage replay verifier.
* **Repository Location:** [`core/relay-chain/replayverifier/`](file:///g:/VAJRA/core/relay-chain/replayverifier), [`security/replay_mitigation.py`](file:///g:/VAJRA/security/replay_mitigation.py)
* **API / Function:** `replayverifier.VerifyReplayProof(proof)`
* **Owner:** Security & Governance Lead
* **Test:** Replay attack simulation tests, 50-run deterministic replay verification.
* **Evidence:** Cryptographic replay verification receipts (`resilience_proof.json`).
* **Known Gap:** Automated multi-tenant replay table partition scaling.

---

### Stage 11: API / RPC
* **Existing Implementation:** RESTful API server, health endpoints, WebSocket channels, and JSON-RPC router.
* **Repository Location:** [`core/relay-chain/api/server.go`](file:///g:/VAJRA/core/relay-chain/api/server.go), [`core/relay-chain/explorer/`](file:///g:/VAJRA/core/relay-chain/explorer)
* **API / Function:** `GET /api/v1/status`, `POST /api/v1/transaction`, `GET /api/v1/blocks`
* **Owner:** Core API / Ishan (Apprenticeship & Support)
* **Test:** HTTP route status codes, payload serialization tests, latency benchmarking.
* **Evidence:** Live HTTP response JSON payloads and Prometheus metrics.
* **Known Gap:** Full Ethereum JSON-RPC (`eth_*`) compatibility wrapper.
