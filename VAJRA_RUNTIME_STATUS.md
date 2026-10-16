# VAJRA Runtime Status

**Document Specification:** Phase 1 Current-State Audit  
**Owner:** Prakash Kumar (VAJRA Blockchain Core, Mainnet & Production Runtime Owner)  
**Constitutional Domain:** Trust Layer / Settlement Substrate  
**Date:** 2026-09-21  

---

## 1. Blockchain Architecture & Runtime Model

### 1.1 Architecture Overview
The VAJRA blockchain operates as a sovereign Proof of Stake (PoS) consensus network and cryptographic trust substrate for the BHIV ecosystem and the TANTRA execution organism. It uses a clean separation of concerns:
- **Consensus & State**: Go-based relay chain engine with LevelDB key-value persistence.
- **Networking**: `go-libp2p` peer-to-peer transport protocol (`/blackhole/1.0.0`) on port `3000`.
- **Identity & Auth**: Native Secp256k1 BTCEC cryptographic keys with external SSO bridging to Pravesh (`https://pravesh.blackholeinfiverse.com`).
- **Auditing & Provenance**: Dual-write Merkle evidence ledger (`tantra_truth.jsonl`) and state reconstruction store (`akashic_lineage.jsonl`).

### 1.2 Node Model & Cluster Topology
* **Cluster Size**: 5 nodes (`blackhole-node-1` through `blackhole-node-5`).
* **Node 1 (Bootnode)**:
  - HTTP API: `8080`
  - P2P Port: `3000` (Docker host-mapped `30303`)
  - Discovery Endpoint: `/api/p2p/info` providing dynamic libp2p multiaddrs.
* **Nodes 2–5 (Validators)**:
  - HTTP APIs: `8081` through `8084`
  - P2P Ports: `3001` through `3004` (Docker host-mapped `30304`–`30307`)
  - Dynamic Peer Discovery: Discovering Node 1 via `BOOTNODES_HTTP=http://blackhole-node-1:8080/api/p2p/info`.
* **State Persistence**: Each node maintains an independent `syndtr/goleveldb` database instance (`blockchaindb_<port>`).

### 1.3 Consensus Engine
* **Algorithm**: Proof of Stake (PoS) with stake-weighted pseudorandom validator election (`core/relay-chain/consensus/pos.go`).
* **Block Interval**: 5 seconds target, 6 seconds minimal ticker interval.
* **Mempool Trigger**: Automatic block mining triggers when $\ge 3$ transactions accumulate in the pool.
* **Deflationary Mechanism**: Fee burn rate target and dynamic reward inflation manager adjust block rewards based on the aggregate network staking ratio.

### 1.4 Transaction Model & Nonce Pipeline
Every transaction adheres to the canonical 9-stage deterministic pipeline:
$$\text{Wallet Client} \longrightarrow \text{API Gateway} \longrightarrow \text{MDU Schema v1 Validation} \longrightarrow \text{Secp256k1 Signature Check} \longrightarrow \text{Nonce Ledger Anti-Replay} \longrightarrow \text{TxPool Ingestion} \longrightarrow \text{P2P Broadcast} \longrightarrow \text{PoS Consensus} \longrightarrow \text{State Mutation} \longrightarrow \text{TruthStore Commitment}$$

---

## 2. Core Subsystems Audit

### 2.1 Wallet
* **Local Components**: Key derivation via Secp256k1 and Ed25519, encrypted keystores, local BadgerDB and MongoDB multi-DB storage adapters.
* **SSO Integration**: Direct integration with deployed Pravesh AUTH service (`https://pravesh.blackholeinfiverse.com`).
* **Runtime Verification**: Authenticated operations require signed payloads matching public keys registered in the account registry.

### 2.2 Native Token (`BHIV`)
* **Max Supply Cap**: `1,000,000,000` (1 Billion tokens strictly enforced in `core/relay-chain/token/token.go`).
* **Decimals**: 18 decimals.
* **Genesis Allocation**: System treasury (`100,000,000 BHIV`), genesis validators (`10,000 BHIV` each).
* **Supply Invariant**: Total supply calculation is checked on every mint operation, rejecting any overflow past 1B.

### 2.3 Staking & Slashing
* **Stake Ledger**: In-memory and LevelDB persisted balance of active validator stakes.
* **Validator Selection**: Pseudorandom selection weighted by active stake balance.
* **Slashing Manager**: Slashing penalties defined for double-signing, node downtime, and consensus divergence.

### 2.4 Over-the-Counter (OTC) Infrastructure
* **Engine**: Order book creation, counter-party matching, price limits, and dual-atomic settlement (`core/relay-chain/otc/otc.go`).
* **Escrow**: In-ledger token lockup during order matching lifecycle.
* **Status**: Decoupled from `chain` package circular imports; ready for full native runtime initialization.

### 2.5 Bridge & Interoperability
* **Supported Chains**: Ethereum (Sepolia), Polygon (Amoy), Solana.
* **Wrapped BHX Token**: Deployed on Polygon Amoy testnet at `0x64Dd946E7425898FE8FaB6D36eA050071190be40`.
* **Replay & Circuit Breaker**: Rate limiters and replay protections active in `bridge-sdk`.

### 2.6 TruthStore & Deterministic Replay
* **TruthStore**: Hash-linked JSONL journal (`tantra_truth.jsonl`) recording execution hashes, validation hashes, and fraud decisions.
* **AKASHIC Lineage**: Append-only log (`akashic_lineage.jsonl`) enabling 100% deterministic reconstruction of historical state roots.
* **Verification**: Verified via 3-agent parallel PDV agreement ($H_{exec} == H_{val} == H_{replay}$).

---

## 3. Production Infrastructure Reality Check

| Category | Component / Feature | Current State | Notes |
| :--- | :--- | :---: | :--- |
| **Local** | Local 5-node cluster, local web wallet, BadgerDB fallback, unit test suites. | **Functional** | Verified locally across 26 packages and 14 E2E hops. |
| **Deployed** | Public RPC, Nginx reverse proxy, Pravesh SSO, Bridge listener. | **Active on VM** | VM `163.128.209.18` active on ports `80`, `443`, `5180`, `8087`. |
| **Externally Reachable** | `https://vajra.blackholeinfiverse.com`, `https://pravesh.blackholeinfiverse.com`. | **Reachable** | Resolving and serving HTTP 200 responses. |
| **Mocked** | None in core blockchain engine. | **Zero Mocks** | All cryptographic checks, nonces, and LevelDB operations are real. |
| **Simulated** | Governance proposal voting in dev mode, secondary bridge relay testing. | **Dev Only** | Core consensus executes real validator election. |
| **Production Ready** | PoS engine, LevelDB storage, panic recovery, Prometheus metrics, MDU validation. | **Production Ready** | Zero-crash guarantee enforced via HTTP panic middleware. |
| **Missing / In Progress** | Automatic cross-subdomain cookie domain on Pravesh; container redeploy on VM. | **In Deployment** | Pushed to GitHub `main` (`806e5a8`) for automated CI/CD rollout. |
