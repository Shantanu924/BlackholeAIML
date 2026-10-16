# VAJRA Mainnet Architecture Specification

**Document Version:** 1.0.0  
**Status:** Production-Ready Permissioned Architecture  
**Audit Standard:** Ground-Truth Verified (Rules 0.1 – 0.4)

---

## 1. Network Overview & Classification

| Parameter | Technical Specification | Evidence Location |
|---|---|---|
| **Network Name** | VAJRA Mainnet (`vajra-mainnet-1`) | `scripts/genesis_config.json` |
| **Network Classification** | **Permissioned / Private PoS Cluster Network with Testnet Bridge Modules** | `docker-compose.production.template.yml` |
| **Chain ID** | `vajra-mainnet-1` | `scripts/genesis_config.json` |
| **Genesis Timestamp** | `2026-08-20T00:00:00Z` (`1787184000`) | `core/relay-chain/chain/blockchain.go:464` |
| **Genesis Block Hash** | Deterministic SHA-256 Merkle Hash | `core/relay-chain/chain/blockchain.go:466` |
| **Consensus Algorithm** | Proof of Stake (PoS) with Stake-Weighted Selection | `core/relay-chain/consensus/pos.go` |
| **Validator Count** | 5 Active Nodes in Cluster | `docker-compose.production.template.yml` |
| **Block Time / Interval** | 5 seconds nominal; 6 seconds mining ticker | `core/relay-chain/cmd/relay/main.go:270` |
| **Block Finality** | Longest-Chain Rule with Immediate LevelDB Commit | `core/relay-chain/consensus/pos.go:100-120` |
| **P2P Networking** | `go-libp2p` v0.41.1 on protocol `/blackhole/1.0.0` | `core/relay-chain/chain/p2p.go:77` |
| **RPC / API Engine** | Go `net/http` REST API on port `8080` | `core/relay-chain/api/server.go` |
| **State Storage Engine** | LevelDB (`github.com/syndtr/goleveldb`) | `core/relay-chain/chain/blockchain.go:178` |

---

## 2. Validator Topology & Identity Architecture

```
                    ┌─────────────────────────┐
                    │  Nginx Reverse Proxy    │
                    │      (Port 443/80)      │
                    └───────────┬─────────────┘
                                │
                 ┌──────────────┴──────────────┐
                 │                             │
                 ▼                             ▼
   ┌───────────────────────────┐ ┌───────────────────────────┐
   │    blockchain-node-1      │ │     blockchain-node-2     │
   │   • Role: Primary Bootnode│ │   • Role: Validator 2     │
   │   • HTTP API: :8080       │ │   • HTTP API: :8081       │
   │   • P2P Wire: :3000/:30303│ │   • P2P Wire: :3000/:30304│
   │   • LevelDB: /app/data    │ │   • LevelDB: /app/data    │
   └─────────────┬─────────────┘ └─────────────┬─────────────┘
                 │                             │
                 └──────────────┬──────────────┘
                                │ libp2p Gossip (/blackhole/1.0.0)
                 ┌──────────────┴──────────────┐
                 │                             │
                 ▼                             ▼
   ┌───────────────────────────┐ ┌───────────────────────────┐
   │    blockchain-node-3      │ │     blockchain-node-4     │
   │   • Role: Validator 3     │ │   • Role: Validator 4     │
   │   • HTTP API: :8082       │ │   • HTTP API: :8083       │
   │   • P2P Wire: :3000/:30305│ │   • P2P Wire: :3000/:30306│
   └───────────────────────────┘ └───────────────────────────┘
                                │
                                ▼
                 ┌───────────────────────────┐
                 │     blockchain-node-5     │
                 │   • Role: Validator 5     │
                 │   • HTTP API: :8085       │
                 │   • P2P Wire: :3000/:30307│
                 └───────────────────────────┘
```

### 2.1 Bootnode Discovery Protocol
1. **Node 1 Initialization:** Node 1 boots as the anchor peer on P2P port 3000 (host port 30303). It reads/generates its persistent Secp256k1 key from `data/blockchain/identity/peer_identity_3000.json` and exposes its multiaddr via `GET /api/p2p/info`.
2. **Nodes 2–5 Dynamic Discovery:** Secondary nodes start with `BOOTNODES_HTTP=http://blackhole-node-1:8080/api/p2p/info`. They poll Node 1's REST endpoint to retrieve the live multiaddr, connect via `bc.P2PNode.Connect()`, and begin libp2p stream communication.
3. **P2P Wire Serialization:** Messages use binary-encoded frames with `ProtocolVersion = 1` for transaction broadcasting (`MessageTypeTx`), block broadcasting (`MessageTypeBlock`), and synchronization range requests (`MessageTypeSyncReq`/`MessageTypeSyncResp`).

---

## 3. Consensus & State Transition Model

### 3.1 Proof of Stake (PoS) Election
* Validator stakes are tracked inside `StakeLedger`.
* When pending transactions are ready (`len(PendingTxs) > 0` or threshold $\ge 3$), `miningLoop` invokes `SelectValidator()`:
  $$\text{Selection} = \text{rand}() \pmod{\sum \text{Stake}_i}$$
* The chosen validator creates a block including the nominal 10 BHIV reward transaction from `system` to `selectedValidator`.

### 3.2 Block Verification Pipeline
When a block is received from the P2P stream or mining loop, `ValidateBlock()` verifies:
1. **Block Interval Check:** Ensures elapsed time since last block exceeds tolerance ($5\text{s} - 100\text{ms}$).
2. **Structural Validation:** Merkle root matches transaction SHA-256 hashes; previous hash matches current tip.
3. **Fork Resolution (Longest Chain Rule):** If a competing valid chain branch has greater length, the node executes `Reorganize()` to adopt the canonical chain.
4. **State Persistence:** Committed blocks and account balances are written to LevelDB (`db.Put("block:<hash>", data)`).

---

## 4. Disaster Recovery & Backup Architecture

* **Automated Rollback (CI/CD):** `.github/workflows/cicd.yml` validates health checks across 18 attempts (3 minutes). If any critical service is unhealthy, the pipeline automatically pulls the last healthy Git SHA from `RELEASE_HISTORY.md` and triggers zero-downtime rollback.
* **State Snapshots:** LevelDB directories (`blockchaindb_*`) and JSONL audit files (`nonce_ledger.jsonl`, `tantra_truth.jsonl`, `akashic_lineage.jsonl`) can be backed up via `tar` snapshots.
* **Failover Script:** `scripts/run_failover_test.sh` validates node replacement and dynamic re-synchronization across peer restarts.
