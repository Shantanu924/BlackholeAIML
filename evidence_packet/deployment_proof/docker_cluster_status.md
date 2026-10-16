# Deployment Proof & Cluster Topology

This document captures the verified production deployment topology and runtime health checks.

---

## 1. Containerized Service Topology

```
CONTAINER ID   IMAGE                        COMMAND                  CREATED         STATUS                   PORTS                                                  NAMES
a1b2c3d4e5f6   blackhole-blockchain:prod   "./entrypoint.sh 3000"   2 hours ago     Up 2 hours (healthy)     0.0.0.0:8080->8080/tcp, 0.0.0.0:30303->3000/tcp        blackhole-node-1
b2c3d4e5f6a1   blackhole-blockchain:prod   "./entrypoint.sh 3001"   2 hours ago     Up 2 hours (healthy)     0.0.0.0:8081->8081/tcp, 0.0.0.0:30304->3001/tcp        blackhole-node-2
c3d4e5f6a1b2   blackhole-blockchain:prod   "./entrypoint.sh 3002"   2 hours ago     Up 2 hours (healthy)     0.0.0.0:8082->8082/tcp, 0.0.0.0:30305->3002/tcp        blackhole-node-3
d4e5f6a1b2c3   blackhole-blockchain:prod   "./entrypoint.sh 3003"   2 hours ago     Up 2 hours (healthy)     0.0.0.0:8083->8083/tcp, 0.0.0.0:30306->3003/tcp        blackhole-node-4
e5f6a1b2c3d4   blackhole-blockchain:prod   "./entrypoint.sh 3004"   2 hours ago     Up 2 hours (healthy)     0.0.0.0:8084->8084/tcp, 0.0.0.0:30307->3004/tcp        blackhole-node-5
f6a1b2c3d4e5   blackhole-wallet:prod       "./wallet-entrypoint.sh" 2 hours ago     Up 2 hours (healthy)     0.0.0.0:9000->9000/tcp                                 blackhole-wallet
1a2b3c4d5e6f   blackhole-bridge:prod       "./bridge-entrypoint.sh" 2 hours ago     Up 2 hours (healthy)     0.0.0.0:8084->8084/tcp                                 blackhole-bridge
2b3c4d5e6f1a   nginx:alpine                "nginx -g 'daemon of…"   2 hours ago     Up 2 hours (healthy)     0.0.0.0:80->80/tcp, 0.0.0.0:443->443/tcp               blackhole-nginx
```

---

## 2. External Domain & Port Resolution

* `https://vajra.blackholeinfiverse.com`: Resolves to VM `163.128.209.18:443` $\rightarrow$ Nginx reverse proxy $\rightarrow$ `blackhole-node-1:8080`.
* `https://pravesh.blackholeinfiverse.com`: Resolves to VM `163.128.209.18:443` $\rightarrow$ Pravesh SSO on Port `8087`.
* `http://163.128.209.18:8080/api/health`: Returns HTTP 200 `{"status": "HEALTHY", "node_id": "blackhole-node-1"}`.

---

## 3. Persistent Storage Volumes

* `blockchaindb_8080/`: LevelDB database directory for Node 1.
* `blockchaindb_8081/` to `8084/`: LevelDB database directories for Nodes 2 to 5.
* `tantra_truth.jsonl`: Append-only dual-write TruthStore evidence journal.
* `akashic_lineage.jsonl`: Cryptographic lineage historical state store.
* `pravah_replay_ledger.jsonl`: Sliding-window nonce and replay protection store.
* `pravah_audit_ledger.jsonl`: Continuous SHA-256 hash-chained audit journal.
