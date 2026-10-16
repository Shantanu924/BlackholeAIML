# VAJRA Runtime Identity Card

**Document Version:** 1.0.0  
**Specification:** Canonical Trust Layer Identity  
**Audit Standard:** Strict Code-Verified Ground Truth (Rules 0.1 – 0.4)

---

## 1. Network Identity
* **System Name:** VAJRA Trust Layer (BlackHole Blockchain)
* **Constitutional Home:** Trust Layer / Settlement Layer
* **Network Model:** Permissioned / Private Cluster Network with Cross-Chain Bridge Interoperability Modules
* **Chain ID:** `vajra-mainnet-1` (Native token network: `vajra-mainnet`)
* **Genesis Timestamp:** `2026-08-20T00:00:00Z`
* **Consensus Mechanism:** Proof of Stake (PoS) with stake-weighted pseudorandom validator election (`core/relay-chain/consensus/pos.go`)
* **Block Interval:** 5 seconds target, 6 seconds minimal ticker interval
* **Mempool Threshold:** 3 transactions trigger automatic block mining
* **Native Currency:** `BHIV` (BlackHole Infiverse) — 18 decimals, 1,000,000,000 max supply cap
* **Primary RPC / API Interface:** HTTP REST on port `8080` (Docker mapped `8080`–`8085`)
* **P2P Wire Protocol:** `/blackhole/1.0.0` on port `3000` (Docker mapped `30303`–`30307`) via `go-libp2p`

---

## 2. Constitutional Authority Boundary
* **Role:** Cryptographic trust, deterministic execution verification, double-spend prevention, tamper-proof state commitment, and immutable evidence recording.
* **Non-Owned Authorities:**
  * TMS $\rightarrow$ Strategy & ecosystem objectives (External)
  * GC $\rightarrow$ Governance legitimacy & sanctioning (External - Sarathi/DGIC)
  * MDU $\rightarrow$ Schemas, semantic truth, & provenance (External)
  * TANTRA $\rightarrow$ Constitutional Execution Organism
  * Bucket $\rightarrow$ Canonical state repository (`tantra_truth.jsonl`)

---

## 3. Production Node Topology
* **Cluster Size:** 5 Nodes (`blockchain-node-1` through `blockchain-node-5`)
* **Node 1:** Primary Bootnode (`HTTP :8080`, `P2P :3000/30303`, `/api/p2p/info` discovery endpoint)
* **Nodes 2–5:** Validator nodes discovering peer multiaddrs dynamically via `BOOTNODES_HTTP=http://blackhole-node-1:8080/api/p2p/info`
* **P2P Identity:** Persistent Secp256k1 keypairs loaded from `data/blockchain/identity/peer_identity_<port>.json`
* **Persistent DB:** `syndtr/goleveldb` LevelDB database per node (`blockchaindb_<port>`)

---

## 4. Key Subsystem Endpoints & Bindings
* **Health Check:** `GET http://localhost:8080/api/health`
* **Blockchain Info:** `GET http://localhost:8080/api/blockchain/info`
* **CEX Polling API:** `GET http://localhost:8080/api/blocks/latest`, `GET http://localhost:8080/api/blocks/{height}`
* **CEX Broadcast API:** `POST http://localhost:8080/api/transactions/broadcast`
* **Runtime Verification:** `POST http://localhost:8080/api/sig/verify`, `POST http://localhost:8080/api/replay/equality`
* **Observability Lineage:** `GET http://localhost:8080/api/akashic/lineage`, `GET http://localhost:8080/api/tantra/records`
* **Wallet Backend Service:** `http://blackhole-wallet:9000`
* **Bridge Relayer Service:** `http://blackhole-bridge:8084`
* **Fraud Detection Service (Sarathi):** `http://blackhole-fraud:9090`
