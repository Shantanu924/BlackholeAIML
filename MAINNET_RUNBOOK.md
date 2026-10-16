# VAJRA Mainnet Operations Runbook

**Document Specification:** Production Deployment & Node Operations Runbook  
**Owner:** Prakash Kumar (VAJRA Blockchain Core, Mainnet & Production Runtime Owner)  
**Constitutional Domain:** Trust Layer / Settlement Substrate  
**Status:** Production Certified  

---

## 1. Prerequisites & Environment Setup

* **Operating System:** Ubuntu 22.04 LTS / 24.04 LTS
* **Hardware Requirements:** Minimum 4 vCPU, 8 GB RAM, 100 GB SSD storage
* **Software Dependencies:** Docker Engine $\ge 24.0$, Docker Compose v2, Go 1.22+, Python 3.12+
* **Required Open Firewall Ports:**
  - `80`, `443` (HTTP/HTTPS Nginx Ingress)
  - `8080`–`8084` (Node REST APIs)
  - `30303`–`30307` (libp2p P2P wire gossip)
  - `9000` (Wallet backend API)
  - `8087` (Pravesh SSO Service)

---

## 2. Bootstrapping Sequence

### Step 1: Genesis Configuration
Verify that `scripts/genesis_config.json` contains the canonical allocations:
- Treasury allocation: `100,000,000 BHIV`
- Genesis validator stakes: `10,000 BHIV` each
- Total supply ceiling invariant: `1,000,000,000 BHIV`

### Step 2: Launch 5-Node Blockchain Cluster
Deploy the 5-node cluster using Docker Compose:
```bash
docker compose -f docker-compose.cluster.yml up -d
```
Verify all 5 nodes report running status:
```bash
docker compose -f docker-compose.cluster.yml ps
```

### Step 3: Verify Bootnode Peer Discovery
Query Node 1 to verify dynamic libp2p multiaddrs:
```bash
curl -s http://localhost:8080/api/p2p/info | jq .
```
Expected output:
```json
{
  "peer_id": "12D3KooW...",
  "listen_addresses": [
    "/ip4/0.0.0.0/tcp/3000"
  ]
}
```

### Step 4: Verify Node Interconnectivity
Check connected peer count on validator nodes:
```bash
curl -s http://localhost:8081/api/blockchain/info | jq .peer_count
```
Ensure each node reports connected peers $\ge 1$.

### Step 5: Start Wallet & Pravesh Auth Service
```bash
docker compose -f docker-compose.wallet.yml up -d
curl -s http://localhost:9000/api/status | jq .
```

### Step 6: Start Cross-Chain Bridge Daemon
```bash
docker compose -f docker-compose.bridge.yml up -d
curl -s http://localhost:8084/health | jq .
```

---

## 3. Routine Node Maintenance & Operations

### 3.1 Checking Node Health
```bash
curl -i http://localhost:8080/api/health
curl -s http://localhost:8080/api/monitoring/unified | jq .
```

### 3.2 Checking Block Production & Mempool
```bash
curl -s http://localhost:8080/api/blocks/latest | jq .
```

### 3.3 Prometheus Metrics Scraping
```bash
curl -s http://localhost:8080/api/metrics | head -n 25
```

---

## 4. Key Management & Rotation

### 4.1 Validator Node Signing Key Rotation
1. Generate a new Secp256k1 key pair:
   ```bash
   go run core/relay-chain/cmd/keygen/main.go > new_validator.key
   ```
2. Submit a stake migration transaction re-delegating validator balance to the new public key.
3. Replace the node's mounted key in `config/node_key.json` and restart the validator container:
   ```bash
   docker restart blackhole-node-2
   ```

### 4.2 Governance RS256 Key Pair Rotation
1. Generate new 2048-bit RSA key pair:
   ```bash
   python -c "from security.jwt_handler import RS256SecurityEngine; eng = RS256SecurityEngine(); print(eng.get_private_key_pem()); print(eng.get_public_key_pem())"
   ```
2. Save private key to `secrets/jwt_rs256_private.pem` and public key to `secrets/jwt_rs256_public.pem`.
3. Set environment variables `JWT_PRIVATE_KEY_PATH` and `JWT_PUBLIC_KEY_PATH` before restarting the governance API container.

---

## 5. Rollback & Disaster Recovery Procedures

### 5.1 Automated Snapshot Restore
If state corruption or unrecoverable consensus partition occurs:
```bash
bash scripts/snapshot_recovery.sh --restore latest
```
This restores the validated LevelDB snapshot and restarts all 5 nodes sequentially.

### 5.2 Emergency Rollback Procedure
If a bad release or consensus divergence requires rolling back to a previous binary:
```bash
bash scripts/rollback.sh
```
This script reverts git to the last certified tag, rebuilds the Docker images, and verifies health via `/api/health`.
