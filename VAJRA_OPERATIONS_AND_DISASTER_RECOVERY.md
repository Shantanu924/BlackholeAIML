# VAJRA OPERATIONS & DISASTER RECOVERY RUNBOOK

**Document Version:** 1.0.0  
**Status:** Production Operational Runbook  
**Audit Standard:** Ground-Truth Operations (Rules 0.1 – 0.4)

---

## 1. Cluster Operational Management

### 1.1 Service Management Commands
```bash
# Start full 5-node production cluster
docker compose -f docker-compose.production.yml up -d

# Check cluster status and health
docker compose -f docker-compose.production.yml ps

# Inspect real-time logs
docker compose -f docker-compose.production.yml logs -f --tail=100 blockchain-node-1
```

### 1.2 Health Check Endpoints
| Component | Endpoint | Expected Response |
|---|---|---|
| **Node 1 Health** | `GET http://localhost:8080/api/health` | `{"status":"healthy","uptime":...}` |
| **P2P Discovery** | `GET http://localhost:8080/api/p2p/info` | `{"peer_id":"...","multiaddrs":["..."]}` |
| **Bridge Health** | `GET http://localhost:8084/health` | `{"status":"ok","relayers":...}` |
| **Wallet Health** | `GET http://localhost:9000/api/status` | `{"status":"running","db":"connected"}` |
| **Fraud Service** | `GET http://localhost:9091/health` | `{"status":"operational"}` |

---

## 2. CI/CD & Automated Rollback Architecture

```
 GitHub main push
        │
        ▼ (Validate Compose)
 Docker Buildx Multi-Image Compilation
        │
        ▼ (Push to Docker Hub with Short Git SHA)
 SSH Deployment to Remote Production VM
        │
        ▼ (Deploy Stack with substituted IMG_TAG)
 18-Attempt Health-Check Loop (3 Minutes)
        │
        ├─────────────────────────────┬─────────────────────────────┐
        ▼ (All Services Healthy)      ▼ (Health Timeout / Failure)  │
 Update RELEASE_HISTORY.md      Trigger Rollback Job                │
 (Status: SUCCESS)              Extract Last Healthy Git SHA         │
                                Redeploy Last Healthy Container      │
                                Log Status: ROLLBACK_SUCCESS         │
```

---

## 3. Disaster Recovery & Node Replacement

### 3.1 Replacing a Failed Validator Node
1. **Stop Failed Container:** `docker stop blackhole-node-X && docker rm blackhole-node-X`
2. **Clear Corrupted State (if applicable):** Remove corrupted volume directory or restore from snapshot.
3. **Restart Node:** Node dynamically retrieves the live anchor multiaddr from `http://blackhole-node-1:8080/api/p2p/info` and initiates `MessageTypeSyncReq` to catch up from height $0$ to the current chain tip.

### 3.2 State Backup & Restoration Procedures
* **LevelDB State Backup:**
  ```bash
  # Backup blockchain database
  tar -czf backup_blockchain_$(date +%Y%m%d_%H%M%S).tar.gz /data/blockchain
  ```
* **Audit Ledger Backup:**
  ```bash
  # Backup append-only truth and lineage ledgers
  cp nonce_ledger.jsonl tantra_truth.jsonl akashic_lineage.jsonl /var/backups/vajra/
  ```
* **State Restoration:**
  1. Stop node containers.
  2. Extract backup tarball into `/data/blockchain` or `blockchaindb_<port>`.
  3. Start node; `checkExistingBlockchain()` automatically detects existing LevelDB state and resumes indexing without re-creating genesis.
