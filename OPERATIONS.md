# VAJRA Production Operations & SRE Guide

**Document Specification:** Site Reliability Engineering & Operations Manual  
**Owner:** Prakash Kumar (VAJRA Blockchain Core, Mainnet & Production Runtime Owner)  
**Constitutional Domain:** Trust Layer / Settlement Substrate  
**Status:** Production Certified  

---

## 1. Metrics & Observability Architecture

VAJRA exposes comprehensive Prometheus metrics and structured JSON telemetry across all nodes:

### 1.1 Prometheus Endpoints
* **Node 1 (Bootnode):** `http://localhost:8080/api/metrics`
* **Nodes 2–5 (Validators):** `http://localhost:8081/api/metrics` through `8084/api/metrics`
* **Wallet Service:** `http://localhost:9000/api/status`
* **Bridge SDK:** `http://localhost:8084/health`

### 1.2 Core Metrics Catalog

| Metric Name | Type | Description | Threshold / Alert Limit |
|---|---|---|---|
| `vajra_block_height` | Gauge | Current committed block height | Alert if no increment for > 30s |
| `vajra_peer_count` | Gauge | Number of active connected libp2p peers | Alert if peer_count < 2 |
| `vajra_mempool_size` | Gauge | Number of pending transactions awaiting mining | Warning if > 500 |
| `vajra_tps_sliding_window` | Gauge | Real-time transactions per second | Monitored for throughput spikes |
| `vajra_node_memory_bytes` | Gauge | Process RSS memory allocation | Alert if memory usage > 85% of host |
| `vajra_consensus_round_seconds` | Histogram | Latency of PoS validator election & block creation | Alert if round > 12s |

---

## 2. Prometheus Scrape Configuration

Sample `prometheus.yml` configuration:
```yaml
global:
  scrape_interval: 5s
  evaluation_interval: 5s

scrape_configs:
  - job_name: 'vajra-nodes'
    static_configs:
      - targets:
          - 'blackhole-node-1:8080'
          - 'blackhole-node-2:8081'
          - 'blackhole-node-3:8082'
          - 'blackhole-node-4:8083'
          - 'blackhole-node-5:8084'
        labels:
          cluster: 'vajra-mainnet-primary'

  - job_name: 'vajra-wallet'
    static_configs:
      - targets: ['blackhole-wallet:9000']
```

---

## 3. Automated Backup & LevelDB Snapshot Retention

### 3.1 Backup Schedule & Script Execution
Automated daily snapshots are taken via cron using `scripts/snapshot_recovery.sh`:
```bash
# Crontab schedule: Daily at 02:00 UTC
0 2 * * * /bin/bash /opt/vajra/scripts/snapshot_recovery.sh --backup >> /var/log/vajra_backup.log 2>&1
```

### 3.2 Backup Directory Layout
```
/opt/vajra/backups/
├── snapshot_20260921_020000.tar.gz
├── snapshot_20260922_020000.tar.gz
└── latest -> snapshot_20260922_020000.tar.gz
```

### 3.3 Retention Policy
* **Daily Snapshots:** Retained for 7 days.
* **Weekly Snapshots:** Retained for 4 weeks.
* **Monthly Archives:** Uploaded to encrypted S3 cold storage (`s3://bhiv-vajra-backups/cold/`).

---

## 4. Incident Response & Triage Protocols

### Incident 1: Block Production Stoppage (Stagnant Block Height)
1. Query health on all nodes:
   ```bash
   for p in 8080 8081 8082 8083 8084; do curl -s "http://localhost:$p/api/health"; echo ""; done
   ```
2. Check validator election logs:
   ```bash
   docker logs --tail 50 blackhole-node-1 | grep -i "consensus\|pos"
   ```
3. If partition detected, trigger a soft restart of the follower nodes:
   ```bash
   docker restart blackhole-node-2 blackhole-node-3 blackhole-node-4 blackhole-node-5
   ```

### Incident 2: LevelDB Storage Corruption or Disk Full
1. Halt the affected node:
   ```bash
   docker stop blackhole-node-3
   ```
2. Remove corrupted directory:
   ```bash
   rm -rf blockchaindb_8082/
   ```
3. Re-hydrate from Node 1 via state sync:
   ```bash
   docker start blackhole-node-3
   ```
   *(Node 3 will automatically download block history and re-execute blocks sequentially).*
