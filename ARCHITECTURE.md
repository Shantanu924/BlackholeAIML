# VAJRA Architecture & System Topology Specification

**Document Specification:** Canonical System Architecture  
**Owner:** Prakash Kumar (VAJRA Blockchain Core, Mainnet & Production Runtime Owner)  
**Constitutional Domain:** Trust Layer / Settlement Substrate  
**Status:** Production Certified  

---

## 1. System Overview

The VAJRA blockchain operates as a sovereign Proof of Stake (PoS) consensus network and cryptographic trust substrate for the BHIV ecosystem and the TANTRA execution organism. It provides non-bypassable transactional finality, deterministic state transitions, dual-evidence audit logging, and cross-chain bridge settlement.

```
                  ┌────────────────────────────────────────────────────────┐
                  │                 External Client / Web3                 │
                  └───────────────────────────┬────────────────────────────┘
                                              │ HTTPS / TLS (Port 443)
                                              ▼
                  ┌────────────────────────────────────────────────────────┐
                  │          Public Gateway / Nginx Reverse Proxy          │
                  │        (Rate Limiting, Panic Recovery, DDoS Shield)    │
                  └───────────────────────────┬────────────────────────────┘
                                              │ HTTP (Port 8080)
                                              ▼
┌───────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                   VAJRA RELAY-CHAIN CORE RUNTIME                                  │
│                                                                                                   │
│  1. Inbound Request Sanitization & MDU v1 Schema Contract Verification                            │
│  2. Secp256k1 BTCEC Cryptographic Signature Verification                                          │
│  3. Sliding-Window Nonce Ledger & Monotonic Replay Protection                                     │
│  4. 3-Agent Parallel PDV Consensus: H_exec == H_val == H_replay                                    │
│  5. Mempool Aggregation (Automatic trigger at >= 3 transactions or 6s ticker)                     │
│  6. Proof of Stake (PoS) Validator Election & Block Mining                                        │
│  7. Atomic LevelDB State Mutation & Balance Ledger Debits/Credits                                 │
│  8. Append-Only Dual Truth Commitment:                                                            │
│     - tantra_truth.jsonl     (Execution hashes & fraud decisions)                                │
│     - akashic_lineage.jsonl  (Full historical state root reconstruction)                          │
└─────────────────────────────────────────────┬─────────────────────────────────────────────────────┘
                                              │ libp2p Gossip Protocol (/blackhole/1.0.0)
                                              ▼
┌───────────────────────────────────────────────────────────────────────────────────────────────────┐
│                              5-NODE VALIDATOR TOPOLOGY & CONSENSUS MESH                           │
│                                                                                                   │
│   ┌────────────────────┐      ┌────────────────────┐      ┌────────────────────┐                  │
│   │ blackhole-node-1   │◄────►│ blackhole-node-2   │◄────►│ blackhole-node-3   │                  │
│   │ Bootnode (8080/3000│      │ Validator(8081/3001│      │ Validator(8082/3002│                  │
│   └─────────┬──────────┘      └─────────┬──────────┘      └─────────┬──────────┘                  │
│             │                           │                           │                             │
│             └──────────────────┐        │        ┌──────────────────┘                             │
│                                ▼        ▼        ▼                                                │
│                       ┌────────────────────┐  ┌────────────────────┐                              │
│                       │ blackhole-node-4   │◄►│ blackhole-node-5   │                              │
│                       │ Validator(8083/3003│  │ Validator(8084/3004│                              │
│                       └────────────────────┘  └────────────────────┘                              │
└───────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Core Subsystems & Components

### 2.1 Consensus Layer (Proof of Stake)
* **Implementation:** [`core/relay-chain/consensus/pos.go`](file:///d:/Go%20Lang/MP/block-chain-updated/core/relay-chain/consensus/pos.go)
* **Validator Selection:** Pseudorandom selection weighted proportionally by active stake balance.
* **Block Interval:** Target 5 seconds; minimum ticker interval 6 seconds.
* **Mempool Trigger:** Mining triggers automatically upon accumulating $\ge 3$ transactions or upon the 6-second heartbeat.
* **Economic Invariants:**
  - Hard supply ceiling of **1,000,000,000 `BHIV`** enforced by code-level invariant gates in [`core/relay-chain/token/token.go`](file:///d:/Go%20Lang/MP/block-chain-updated/core/relay-chain/token/token.go).
  - Dynamic reward inflation manager adjusting staking yields based on aggregate network participation.
  - Transaction fee burning mechanism creating deflationary pressure during high-throughput intervals.

### 2.2 Transaction Execution & Nonce Pipeline
Every transaction submitted to the relay chain undergoes strict deterministic gating:
$$\text{Wallet} \longrightarrow \text{API Gateway} \longrightarrow \text{MDU Schema v1} \longrightarrow \text{Secp256k1 Sig} \longrightarrow \text{Nonce Anti-Replay} \longrightarrow \text{P2P Broadcast} \longrightarrow \text{PoS Consensus} \longrightarrow \text{LevelDB Mutation} \longrightarrow \text{TruthStore}$$

1. **Schema Check:** Strict JSON validation against canonical MDU v1 format (`sender`, `recipient`, `amount`, `nonce`, `signature`).
2. **Cryptographic Authentication:** BTCEC Secp256k1 signature verification over SHA-256 canonical hash of the transaction body.
3. **Nonce Check:** Monotonic check against `noncestore` ensuring $nonce > last\_seen\_nonce$.
4. **State Commit:** Atomic LevelDB batch write mutating sender and recipient balances.
5. **Evidence Commit:** Block and transaction metadata written to `tantra_truth.jsonl`.

### 2.3 Dual Provenance & Audit Ledger Stores
* **TruthStore (`tantra_truth.jsonl`):**
  - Appends execution hash, validation hash, replay hash, transaction hash, and fraud decision.
  - Enforces equality gate: $H_{\text{exec}} == H_{\text{val}} == H_{\text{replay}}$. Any hash divergence causes hard HTTP 403 rejection.
* **AKASHIC Lineage Store (`akashic_lineage.jsonl`):**
  - Append-only journal recording state root hashes after every block mutation.
  - Enables 100% deterministic reconstruction of historical world states from genesis.

### 2.4 Node Topology & P2P Wire Protocol
* **Wire Protocol:** `go-libp2p` transport protocol `/blackhole/1.0.0` over TCP.
* **Bootnode (Node 1):** Serves `/api/p2p/info` exposing dynamic multiaddrs (`/ip4/<ip>/tcp/3000/p2p/<peer_id>`).
* **Validator Nodes (Nodes 2–5):** Auto-discover Bootnode via `BOOTNODES_HTTP=http://blackhole-node-1:8080/api/p2p/info` and maintain full-mesh peer gossip.
* **Docker Host Port Mapping:**
  - Node 1: HTTP `8080`, P2P `30303`
  - Node 2: HTTP `8081`, P2P `30304`
  - Node 3: HTTP `8082`, P2P `30305`
  - Node 4: HTTP `8083`, P2P `30306`
  - Node 5: HTTP `8084`, P2P `30307`

### 2.5 Interoperability & Bridge Subsystem
* **Bridge SDK:** [`bridge-sdk/`](file:///d:/Go%20Lang/MP/block-chain-updated/bridge-sdk)
* **Supported Chains:** Polygon (Amoy Testnet), Ethereum (Sepolia), Solana.
* **Deployed Contract:** Polygon Amoy wrapped `BHX` token contract at `0x64Dd946E7425898FE8FaB6D36eA050071190be40`.
* **Security Controls:** Monotonic deposit nonces, cryptographic signature validation by relayers, and sliding-window rate limiters.

### 2.6 Over-The-Counter (OTC) Infrastructure
* **Engine:** [`core/relay-chain/otc/otc.go`](file:///d:/Go%20Lang/MP/block-chain-updated/core/relay-chain/otc/otc.go)
* **Workflow:** Order placement $\rightarrow$ Counterparty discovery $\rightarrow$ In-ledger escrow lockup $\rightarrow$ Dual-atomic swap settlement $\rightarrow$ Audit emission.

---

## 3. Production Deployment Architecture

* **Primary Host:** Dedicated Ubuntu VM `163.128.209.18`
* **Containerization:** Multi-stage Docker deployment managed via `docker-compose.cluster.yml`
* **Reverse Proxy:** Nginx with SSL/TLS termination, HTTP/2 support, and automated certificate renewals:
  - `https://vajra.blackholeinfiverse.com` $\longrightarrow$ Blockchain RPC (Port 8080)
  - `https://pravesh.blackholeinfiverse.com` $\longrightarrow$ Pravesh SSO Service (Port 8087)
* **Observability:**
  - Health probe: `GET /api/health`
  - Prometheus metrics: `GET /api/metrics`
  - Unified diagnostics: `GET /api/monitoring/unified`
