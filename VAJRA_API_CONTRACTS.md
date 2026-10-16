# VAJRA API Contracts — Phase 2

**Constitutional Runtime Participants — Declared API Boundaries**

Each participant declares:
- What it **owns** (authority) — it is responsible for these decisions
- What it **does not own** (non_authority) — it defers to other participants
- What endpoints it **exposes** — verifiable via `GET /identity`

---

## Participant 1: VAJRA Blockchain Node (port 8080)

**Trust Layer:** VAJRA Blockchain
**Identity Endpoint:** `GET /identity`

### Authority (OWNED)
| Capability | Endpoint |
|---|---|
| PDV_EQUALITY | `POST /api/relay/submit` — ExecutionAgent == ValidationAgent == ReplayAgent |
| SIGNATURE_VERIFICATION | `POST /api/sig/verify` |
| NONCE_GOVERNANCE | `GET /api/nonce/lookup`, `GET /api/nonce/records` |
| TRACE_CONTINUITY | `GET /api/trace/verify` |
| BUCKET_TRUTH | `GET /api/tantra/records`, `GET /api/tantra/verify`, `GET /api/tantra/chain-integrity` |
| AKASHIC_LINEAGE | `GET /api/akashic/lineage`, `GET /api/akashic/trace`, `GET /api/akashic/reconstruct` |

### Non-Authority (NOT OWNED)
| Capability | Owner |
|---|---|
| GOVERNANCE_LEGITIMACY | Sarathi — external governance service |
| SEMANTIC_TRUTH | Human/legal layer — outside blockchain scope |
| SOVEREIGN_CORRECTNESS | Constitutional layer — outside runtime scope |

### Adjacent Participants
- `fraud-service:9090` — fraud decisions
- `bridge:8084` — cross-chain relay
- `wallet:9000` — transaction intent

### Declared Endpoints
```
POST /api/relay/submit            — canonical transaction entry
GET  /api/health                  — liveness check
GET  /api/tantra/status           — TANTRA ecosystem connectivity
GET  /api/constitution/declaration — constitutional boundary declaration
GET  /api/convergence/proof       — full live convergence proof
GET  /api/vajra/contracts         — unified contract validation (probes all 4 /identity)
GET  /identity                    — VAJRA runtime identity card
```

---

## Participant 2: VAJRA Fraud Detection Service (port 9090)

**Trust Layer:** Cyber Defence
**Identity Endpoint:** `GET /identity`
**Production Path:** `https://vajra.blackholeinfiverse.com/defence/`

### Authority (OWNED)
| Capability | Endpoint |
|---|---|
| FRAUD_DETECTION | `POST /api/fraud/check` |
| VELOCITY_CHECK | `POST /api/fraud/check` — rule 6: velocity |
| ADDRESS_BLOCKING | `POST /api/fraud/block` |
| GOVERNANCE_DECISION | `POST /api/fraud/check` → returns allow/block |

### Non-Authority (NOT OWNED)
| Capability | Owner |
|---|---|
| PDV_EQUALITY | Blockchain Node — hash equality is blockchain's concern |
| BLOCKCHAIN_WRITE | Blockchain Node — fraud service never writes to chain |
| NONCE_GOVERNANCE | Blockchain Node — nonce is blockchain's concern |

### Adjacent Participants
- `blockchain-node-1:8080` — receives fraud check requests

### Declared Endpoints
```
POST /api/fraud/check   — evaluate transaction for fraud
GET  /api/fraud/status  — service health + rule stats
POST /api/fraud/block   — manually block an address
GET  /health            — liveness check
GET  /identity          — VAJRA runtime identity card
```

---

## Participant 3: VAJRA Wallet Service (port 9000)

**Trust Layer:** Wallet
**Identity Endpoint:** `GET /identity`
**Production Path:** `https://vajra.blackholeinfiverse.com/wallet/`

### Authority (OWNED)
| Capability | Endpoint |
|---|---|
| WALLET_INTENT | `POST /api/wallets/transfer` — user's intent to transact |
| KEY_MANAGEMENT | `POST /api/wallets/create`, `POST /api/wallets/import`, `POST /api/wallets/export` |
| TRANSACTION_SIGNING | `POST /api/wallets/transfer` — signs before submitting to blockchain |
| USER_AUTHENTICATION | `POST /api/login`, `POST /api/register` |

### Non-Authority (NOT OWNED)
| Capability | Owner |
|---|---|
| PDV_EQUALITY | Blockchain Node — wallet submits, blockchain validates |
| GOVERNANCE_LEGITIMACY | Sarathi — wallet does not govern |
| BLOCKCHAIN_WRITE | Blockchain Node — wallet only submits intent |
| FRAUD_DETECTION | Fraud Service — wallet does not make fraud decisions |

### Adjacent Participants
- `blockchain-node-1:8080` — submits transactions via `POST /api/relay/submit`
- `bridge:8084` — cross-chain operations

### Declared Endpoints
```
POST /api/wallets/transfer   — submit transfer intent
GET  /api/wallets            — list user wallets
POST /api/wallets/create     — create new wallet
GET  /api/status             — service health
GET  /identity               — VAJRA runtime identity card
```

---

## Participant 4: VAJRA Bridge SDK (port 8084)

**Trust Layer:** Bridge
**Identity Endpoint:** `GET /identity`
**Production Path:** `https://vajra.blackholeinfiverse.com/bridge/`

### Authority (OWNED)
| Capability | Endpoint |
|---|---|
| CROSS_CHAIN_TRANSFER | `POST /relay/eth`, `POST /relay/sol` |
| REPLAY_PROTECTION | Internal BoltDB — deduplicates by eventHash |
| EVENT_LISTENING | Ethereum + Solana listeners |
| CIRCUIT_BREAKER | Internal — per-chain circuit breakers |

### Non-Authority (NOT OWNED)
| Capability | Owner |
|---|---|
| PDV_EQUALITY | Blockchain Node — bridge does not run PDV |
| FRAUD_DETECTION | Fraud Service — bridge defers fraud decisions |
| NONCE_GOVERNANCE | Blockchain Node — bridge does not govern nonces |
| BLOCKCHAIN_CONSENSUS | Blockchain Node — bridge relays, does not validate |

### Adjacent Participants
- `blockchain-node-1:8080` — submits bridge transactions
- `wallet:9000` — wallet-initiated bridge operations

### Declared Endpoints
```
GET  /health         — liveness check
GET  /stats          — bridge statistics
GET  /transactions   — transaction list
GET  /bridge/status  — bridge operational status
POST /relay/eth      — relay signed Ethereum bridge message
POST /relay/sol      — relay signed Solana bridge message
GET  /identity       — VAJRA runtime identity card
```

---

## Contract Validation

To verify all contracts are live:

```bash
# Blockchain node identity
curl https://vajra.blackholeinfiverse.com/identity

# Fraud service identity
curl https://vajra.blackholeinfiverse.com/defence/identity

# Wallet service identity
curl https://vajra.blackholeinfiverse.com/wallet/identity

# Bridge SDK identity
curl https://vajra.blackholeinfiverse.com/bridge/identity

# Machine-readable unified contract validation (probes all 4)
curl https://vajra.blackholeinfiverse.com/api/vajra/contracts
```

---

## Boundary Rules

1. **No participant writes to another participant's store** — each owns its own data
2. **Fraud service never writes to blockchain** — it only returns allow/block decisions
3. **Wallet never makes fraud decisions** — it submits intent, blockchain enforces
4. **Bridge never runs PDV** — it relays events, blockchain validates
5. **All transactions enter blockchain via `POST /api/relay/submit`** — no bypass paths

---

## nginx Routing (Production)

| Path | Service | Internal |
|---|---|---|
| `https://vajra.blackholeinfiverse.com/` | Blockchain Node | `blackhole-node-1:8080` |
| `https://vajra.blackholeinfiverse.com/api/` | Blockchain Node | `blackhole-node-1:8080` |
| `https://vajra.blackholeinfiverse.com/wallet/` | Wallet Service | `blackhole-wallet:9000` |
| `https://vajra.blackholeinfiverse.com/bridge/` | Bridge SDK | `blackhole-bridge:8084` |
| `https://vajra.blackholeinfiverse.com/defence/` | Fraud Service | `blackhole-fraud:9090` |

---

*VAJRA Phase 2 — API Contracts*
*Owner: Prakash Yadav*
*Last Updated: 05 Aug 2026*
