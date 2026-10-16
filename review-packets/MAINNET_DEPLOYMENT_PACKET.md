# Mainnet Deployment Packet
**Scope 6 — BlackHole Blockchain / TANTRA**
**Date:** 2025-07-17
**Status:** DEPLOYMENT-READY PATH DEFINED

---

## 1. Deployment Topology

```
┌─────────────────────────────────────────────────────────────────────┐
│                        TANTRA PRODUCTION TOPOLOGY                    │
│                                                                      │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐           │
│  │  PDV-EXEC    │    │  PDV-VAL     │    │  PDV-REPLAY  │           │
│  │  :9101       │    │  :9102       │    │  :9103       │           │
│  │  host-A      │    │  host-B      │    │  host-C      │           │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘           │
│         └──────────────────┼──────────────────┘                     │
│                            │ PDV equality gate                       │
│                            ▼                                         │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐           │
│  │  RELAY-A     │    │  RELAY-B     │    │  RELAY-C     │           │
│  │  :8080       │    │  :8081       │    │  :8082       │           │
│  │  host-D      │    │  host-E      │    │  host-F      │           │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘           │
│         └──────────────────┼──────────────────┘                     │
│                            │ KarmaChain replication                  │
│                            ▼                                         │
│  ┌──────────────┐    ┌──────────────┐                               │
│  │ NONCE-COORD  │    │  SARATHI     │                               │
│  │  :9200       │    │  :9090       │                               │
│  │  host-G      │    │  host-H      │                               │
│  └──────────────┘    └──────────────┘                               │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  WALLET SERVICE  :9000  (services/wallet/)                   │   │
│  └──────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. Environment Strategy

### Production Environment Variables (all relay nodes)

```bash
# PDV agents — point to separate hosts
PDV_EXECUTION_AGENT_URL=http://<host-A>:9101/pdv/execute
PDV_VALIDATION_AGENT_URL=http://<host-B>:9102/pdv/validate
PDV_REPLAY_AGENT_URL=http://<host-C>:9103/pdv/replay

# Governance — external Sarathi service
SARATHI_URL=http://<host-H>:9090/api/fraud/check

# Cross-node nonce coordination
NONCE_COORDINATOR_URL=http://<host-G>:9200

# KarmaChain replication (comma-separated peer relay URLs)
KARMACHAIN_NODES=http://<host-E>:8081,http://<host-F>:8082

# Production-safe defaults — DO NOT SET these to disable:
# PDV_STRICT_MODE=false        ← NEVER in production
# SIGVERIFY_STRICT_MODE=false  ← NEVER in production
# SARATHI_FAIL_CLOSED=false    ← NEVER in production

# Node identity
NODE_ID=relay-a   # unique per node
DOCKER_MODE=true
BLOCKCHAIN_DOCKER=true
```

### Dev/Test Override (explicit opt-in required)
```bash
PDV_STRICT_MODE=false
SIGVERIFY_STRICT_MODE=false
SARATHI_FAIL_CLOSED=false
```
Startup banner will emit `[WARNING]` and `DEV MODE ACTIVE` for each disabled safety.

---

## 3. Bootstrap Flow

```
Step 1: Start nonce coordinator (FIRST — all relay nodes depend on it)
  go run cmd/noncecoord/main.go -port 9200 -ledger /data/global_nonce_ledger.jsonl

Step 2: Start PDV agents (3 separate hosts/containers)
  go run cmd/pdv-agent/main.go -port 9101 -agent ExecutionAgent
  go run cmd/pdv-agent/main.go -port 9102 -agent ValidationAgent
  go run cmd/pdv-agent/main.go -port 9103 -agent ReplayAgent

Step 3: Start Sarathi/fraud service
  go run cmd/fraud-service/main.go   (port 9090)

Step 4: Start relay nodes (after all dependencies healthy)
  go run cmd/relay/main.go 3000      (relay-A, API on :8080)
  go run cmd/relay/main.go 3001      (relay-B, API on :8081)

Step 5: Start wallet service
  cd services/wallet && go run main.go -port 9000

Step 6: Verify startup safety posture
  go run cmd/startuptest/main.go
  # Expected: PRODUCTION MODE — all safety defaults active

Step 7: Run convergence proof
  go run cmd/selftest/main.go -relay http://localhost:8080 -out selftest_results.jsonl
  # Expected: 12/12 PASS

Step 8: Run validation v2
  go run cmd/validation-v2/main.go -relay http://localhost:8080 -coord http://localhost:9200 -out v2_results.jsonl
  # Expected: 8/8 PASS

Step 9: Run hostile-condition proof
  go run cmd/hostile-proof/main.go -relay http://localhost:8080
  # Expected: all H1-H8 DETECTED + RECOVERED

Step 10: Verify TANTRA ecosystem status
  curl http://localhost:8080/api/tantra/status
  # Expected: all components online
```

---

## 4. Node Synchronization Procedure

When adding a new relay node to an existing cluster:

```
1. Start nonce coordinator (already running — new node connects to it)
   Set NONCE_COORDINATOR_URL=http://<coord-host>:9200

2. Start new relay node with KARMACHAIN_NODES pointing to existing nodes
   KARMACHAIN_NODES=http://<relay-a>:8080,http://<relay-b>:8081

3. Pull lineage from existing node:
   curl http://<relay-a>:8080/api/karmachain/reconstruct
   # Returns full lineage — new node replays to catch up

4. Verify state root equality:
   curl http://<new-relay>:8080/api/replay/state-root
   # Expected: equal=true, agreed_count matches node_count

5. Verify constitutional boundaries:
   curl http://<new-relay>:8080/api/constitution/declaration
   # Expected: all OWNED boundaries intact, no OPEN GAPs beyond known ones
```

---

## 5. Recovery Procedures

### PDV Agent Failure
```
Detection: POST /api/relay/submit → HTTP 403 DISTRIBUTED_PDV_REJECT
Recovery:
  1. Restart failed agent: go run cmd/pdv-agent/main.go -port <port> -agent <name>
  2. Verify health: curl http://<agent-host>:<port>/health
  3. Submit test transaction — should succeed with equality confirmed
  4. No state recovery needed — PDV agents are stateless hash computers
```

### Nonce Coordinator Failure
```
Detection: POST /api/relay/submit → HTTP 409 NONCE_COORD_UNAVAILABLE
  (fail-closed: coordinator unreachable = nonce rejected)
Recovery:
  1. Restart coordinator: go run cmd/noncecoord/main.go -port 9200 -ledger global_nonce_ledger.jsonl
  2. Coordinator loads global_nonce_ledger.jsonl on startup — all nonces restored
  3. Relay nodes resume accepting transactions automatically
  4. Verify: curl http://localhost:9200/nonce/records
```

### Relay Node Failure + Restart
```
Detection: Node process exits / container stops
Recovery:
  1. Restart: go run cmd/relay/main.go 3000
  2. nonce_ledger.jsonl loaded on startup — all nonces restored (NONCE_REPLAY on duplicate)
  3. akashic_lineage.jsonl loaded — lineage intact
  4. tantra_truth.jsonl loaded — bucket intact
  5. Verify: curl http://localhost:8080/api/convergence/proof
     Expected: converged=true
  6. If AKASHIC chain broken (corruption simulation left): restart clears in-memory state
     Persistent files survive restart — chain integrity verified on load
```

### Sarathi/Governance Failure
```
Detection: POST /api/relay/submit → HTTP 403 GOVERNANCE_REJECT
  (SARATHI_FAIL_CLOSED=true by default — unreachable = block)
Recovery:
  1. Restart Sarathi: go run cmd/fraud-service/main.go
  2. Transactions resume automatically — no state to recover
  3. If intentional downtime needed: set SARATHI_FAIL_CLOSED=false (DEV ONLY)
     Startup banner will emit WARNING
```

### AKASHIC Corruption Detected
```
Detection: GET /api/akashic/reconstruct → HTTP 409 chain broken at entry N
Recovery:
  1. Pull clean lineage from peer: curl http://<peer-relay>:8081/api/karmachain/reconstruct
  2. Replay entries via: POST /api/akashic/replicate (for each entry)
  3. Verify: GET /api/akashic/reconstruct → HTTP 200 verified=true
  4. Verify state root equality: GET /api/replay/state-root → equal=true
```

---

## 6. Observability Dashboard / Proof Visibility

### Live Endpoints (relay node)

| Endpoint | Purpose |
|---|---|
| `GET /api/tantra/status` | Full TANTRA ecosystem connectivity |
| `GET /api/constitution/declaration` | Runtime authority boundaries |
| `GET /api/convergence/proof` | Full convergence proof (bucket + AKASHIC + runtime) |
| `GET /api/replay/state-root` | State root equality across nodes |
| `GET /api/akashic/reconstruct` | AKASHIC lineage reconstruction |
| `GET /api/tantra/chain-integrity` | Bucket chain tamper detection |
| `GET /api/nonce/records` | Full nonce lineage |
| `GET /api/tantra/records` | Full truth store records |
| `GET /api/constitution/verify-boundary?name=<n>` | Boundary integrity |
| `GET /api/health` | Node health |

### Proof Programs

| Program | What It Proves | Run Mode |
|---|---|---|
| `cmd/selftest/main.go` | 12 tamper-evident tests, suite_hash | Live relay |
| `cmd/validation-v2/main.go` | 8 ecosystem tests, suite_hash | Standalone + live |
| `cmd/hostile-proof/main.go` | H1-H8 hostile conditions | Standalone + live |
| `cmd/pdv-proof/main.go` | 3-process distributed equality | Standalone |
| `cmd/sig-proof/main.go` | 8 signature proofs | Standalone |
| `cmd/nonce-coord-proof/main.go` | 7 cross-node nonce proofs | Live coordinator |
| `cmd/governance-proof/main.go` | 5 governance proofs | Standalone |
| `cmd/recovery-proof/main.go` | 6 recovery proofs | Live relay |

---

## 7. Failure-Handling Playbook

| Failure | HTTP Code | Error Code | Immediate Action |
|---|---|---|---|
| PDV agent unreachable (strict) | 403 | `DISTRIBUTED_PDV_REJECT` | Restart agent, verify health |
| PDV hash mismatch | 403 | `DISTRIBUTED_PDV_REJECT` | Investigate agent state divergence |
| Duplicate nonce | 409 | `NONCE_REPLAY` | Client must use next_nonce from `/api/nonce/lookup` |
| Nonce coordinator down | 409 | `NONCE_COORD_UNAVAILABLE` | Restart coordinator |
| Governance unreachable | 403 | `GOVERNANCE_REJECT` | Restart Sarathi |
| Invalid signature | 403 | `SIGNATURE_REJECT` | Client must sign with valid btcec key |
| Schema violation | 400 | `SCHEMA_VIOLATION` | Client must include all required fields |
| AKASHIC corruption | 409 | chain broken at N | Pull lineage from peer, replay |
| State root divergence | 409 | `STATE_ROOT_DIVERGENCE` | Identify divergent node, resync |
| Trace drift | 500 | `TRACE_BREAK` | Internal error — restart relay |

---

## 8. Docker Deployment

### Single-node (dev)
```bash
cd core/relay-chain
docker-compose -f docker-compose.pdv.yml up --build
```

### Full TANTRA ecosystem (production-like)
```bash
cd core/relay-chain
docker-compose -f docker-compose.tantra.yml up --build
```

### Verify after startup
```bash
# Wait for all services healthy, then:
curl http://localhost:8080/api/tantra/status
curl http://localhost:8080/api/constitution/declaration
curl http://localhost:8080/api/convergence/proof
```

---

## 9. Known Unresolved Gaps (Explicit)

| # | Gap | Severity | Mitigation |
|---|---|---|---|
| 1 | PDV agents on same machine (non-Docker) | LOW | Use docker-compose.tantra.yml for true cross-host |
| 2 | Sarathi not externally sovereign | MEDIUM | Set SARATHI_URL to external service |
| 3 | KSML/CET upstream alignment | LOW | Awaiting Raj Prajapati — /api/ksml/submit ready |
| 4 | Wallet NonceRegistry in-memory | LOW | Relay-side nonce is persistent; wallet-side is session |
| 5 | KarmaChain replication unidirectional | LOW | Set KARMACHAIN_NODES on all nodes for bidirectional |

---

## 10. Mainnet Readiness Delta

| Requirement | Status |
|---|---|
| Distributed PDV (3 separate processes) | COMPLETE — docker-compose.tantra.yml |
| Cross-node nonce governance | COMPLETE — noncecoord service |
| Production-safe defaults | COMPLETE — startupcheck inversion |
| Hostile-condition testing | COMPLETE — H1-H8 all DETECTED+RECOVERED |
| TANTRA ecosystem connectivity | COMPLETE — docker-compose.tantra.yml + /api/tantra/status |
| Mainnet deployment packet | COMPLETE — this document |
| Independent validation v2 | COMPLETE — cmd/validation-v2 |
| Build debt (DEX min() conflict) | COMPLETE — renamed to minUint64() |
| Constitutional declaration | COMPLETE — /api/constitution/declaration |
| Replay-safe nonce continuity | COMPLETE — noncestore + noncecoord |
