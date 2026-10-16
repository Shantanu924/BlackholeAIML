# BlackHole Blockchain — Production Deployment Guide

**Domain:** `https://vajra.blackholeinfiverse.com`  
**VM:** `163.128.209.18`  
**Last Updated:** 2026-08-07

---

## Architecture Overview

```
Internet → nginx (port 443, SSL)
              │
    ┌─────────┼──────────┬──────────┐
    ▼         ▼          ▼          ▼
/api/      /bridge/   /defence/  /wallet/
node:8080  bridge:8084 fraud:9090 wallet:9000
```

All services run as Docker containers on a single VM behind nginx.

---

## Prerequisites

- Docker 24+
- Docker Compose v2
- Git
- SSH access to VM (`163.128.209.18`)

---

## CI/CD Deployment (Automatic)

Push to `main` branch triggers the full pipeline automatically:

```
validate → build (4 Docker images) → deploy → health check → rollback on failure
```

**Docker images built:**
- `bhiv/blackhole-node:<sha>`
- `bhiv/blackhole-bridge:<sha>`
- `bhiv/blackhole-wallet:<sha>`
- `bhiv/blackhole-fraud:<sha>`

**GitHub Secrets required:**
| Secret               | Description                    |
|----------------------|--------------------------------|
| `DOCKER_USERNAME`    | Docker Hub username            |
| `DOCKER_PASSWORD`    | Docker Hub password            |
| `VM_IP`              | `163.128.209.18`               |
| `VM_PORT`            | SSH port                       |
| `VM_USERNAME`        | SSH username                   |
| `VM_PASSWORD`        | SSH password                   |
| `ENV_FILE`           | Full contents of production .env|

---

## Manual Deployment

SSH into the VM and run:

```bash
cd ~/BLACKHOLE
git pull
docker compose -f docker-compose.production.yml pull
docker compose -f docker-compose.production.yml up -d --remove-orphans
docker compose -f docker-compose.production.yml ps
```

---

## Environment Variables

Key env vars for `blackhole-node-1`:

```env
NODE_ENV=production
HTTP_PORT=8080
NODE_ID=blackhole-node-1
SARATHI_URL=http://blackhole-fraud:9090
FRAUD_SERVICE_URL=http://blackhole-fraud:9090
WALLET_SERVICE_URL=http://blackhole-wallet:9000
BRIDGE_SERVICE_URL=http://blackhole-bridge:8084
PDV_STRICT_MODE=false
SARATHI_FAIL_CLOSED=false
SIGVERIFY_STRICT_MODE=false
```

---

## Health Checks

```bash
# Blockchain node
curl https://vajra.blackholeinfiverse.com/api/health

# All 4 participants reachable
curl https://vajra.blackholeinfiverse.com/api/vajra/contracts

# Bridge
curl https://vajra.blackholeinfiverse.com/bridge/health

# Wallet
curl https://vajra.blackholeinfiverse.com/wallet/api/status

# Fraud
curl https://vajra.blackholeinfiverse.com/defence/health
```

---

## nginx Configuration

Location blocks on port 443:

```nginx
location /api/      { proxy_pass http://blackhole-node-1:8080; }
location /bridge/   { proxy_pass http://blackhole-bridge:8084; }
location /defence/  { proxy_pass http://blackhole-fraud:9090; }
location /wallet/   { proxy_pass http://blackhole-wallet:9000; }
location /          { proxy_pass http://blackhole-node-1:8080; }
```

---

## Rollback

Automatic rollback triggers on deploy failure — reverts to last healthy SHA from `docs/RELEASE_HISTORY.md`.

Manual rollback:

```bash
cd ~/BLACKHOLE
LAST_SHA=<sha-from-release-history>
sed "s|IMG_TAG|$LAST_SHA|g" docker-compose.production.template.yml > docker-compose.production.yml
docker compose -f docker-compose.production.yml up -d --remove-orphans
```

---

## Logs

```bash
# All services
docker compose -f docker-compose.production.yml logs -f

# Specific service
docker compose -f docker-compose.production.yml logs -f blackhole-node-1

# TANTRA truth store
docker exec blackhole-node-1 cat /app/data/tantra_truth.jsonl
```

---

## Known Gaps

| Gap                              | Workaround                          |
|----------------------------------|-------------------------------------|
| PDV agents single-process        | `PDV_STRICT_MODE=false` fallback    |
| Nonce coordinator not cross-node | Per-node nonce store active         |
| Port 8444 firewall blocked       | Use `/bridge/` path on port 443     |
