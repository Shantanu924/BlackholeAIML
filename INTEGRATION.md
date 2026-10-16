# VAJRA Integration & Boundary Contracts Specification

**Document Specification:** Ecosystem Integration & Authority Boundaries  
**Owner:** Prakash Kumar (VAJRA Blockchain Core, Mainnet & Production Runtime Owner)  
**Constitutional Domain:** Trust Layer / Settlement Substrate  
**Status:** Production Certified  

---

## 1. Constitutional Integration Pipeline

The VAJRA Trust Layer integrates directly with the surrounding TANTRA Constitutional Ecosystem through strictly separated boundaries of authority:

$$\text{USER} \longrightarrow \text{IDENTITY / CONSENT} \longrightarrow \text{TANTRA} \longrightarrow \text{CYBERSECURITY} \longrightarrow \text{VAJRA} \longrightarrow \text{BLOCKCHAIN} \longrightarrow \text{EVIDENCE} \longrightarrow \text{REPLAY} \longrightarrow \text{INSIGHT} \longrightarrow \text{USER CONTROL}$$

### Strict Boundary Rules:
1. **Cybersecurity owns enforcement:** Address freezing and fraud detection policies originate in the Cybersecurity Cell. VAJRA executes the freeze in the ledger without self-authorizing policy changes.
2. **TANTRA owns runtime integration:** MDU schema definitions, orchestrator review gates, and Sarathi AI policy compliance.
3. **VAJRA owns the blockchain trust substrate:** Transaction validation, consensus, state mutation, nonce ordering, and LevelDB persistence.
4. **No layer silently assumes another layer's authority.**

---

## 2. TANTRA Subsystems Integration Matrix

| Subsystem | Owner | Integration Rule | Implementation & Evidence |
|---|---|---|---|
| **TMS (Management)** | TMS Owner | TMS governs high-level ecosystem strategy; VAJRA acts as the sole permanent settlement layer. | Hardwired in deployment manifests as default settlement engine. |
| **GC (Governance)** | GC Owner | Slashing and state interventions require GC cryptographic consensus. | Slashing manager requires signed governance approvals before validator expulsion. |
| **MDU (Master Data)** | MDU Owner | MDU owns the schemas; VAJRA implements strict JSON schema validation matching MDU v1. | Inbound payloads validated in `core/relay-chain/schema/` prior to execution. |
| **Bucket (Evidence)** | Bucket Owner | VAJRA commits on-chain block hashes; Bucket links off-chain evidence via IPFS/CIDs. | `TANTRA_BUCKET_URL` wired in `deploy/mainnet/main.tf` task definitions. |
| **Replay (Simulation)**| Replay Owner | VAJRA generates deterministic state transitions that can be independently replayed. | Verified via `akashic_lineage.jsonl` and 3-agent parallel PDV agreement. |
| **Pravah (Observability)**| Pravah Owner | VAJRA emits live telemetry, audit events, and metric streams. | Prometheus exporter on `/api/metrics` and CloudWatch `awslogs` forwarding. |

---

## 3. Pravesh SSO Identity Integration

* **SSO Endpoint:** `https://pravesh.blackholeinfiverse.com`
* **Protocol:** JWT Bearer authentication with remote `/api/me` token introspection.
* **Implementation:** [`services/wallet/jwt_auth.go`](file:///d:/Go%20Lang/MP/block-chain-updated/services/wallet/jwt_auth.go)
* **Auth Flow:**
  1. Client signs in via Pravesh SSO UI (`/login`).
  2. Pravesh issues signed session cookie / `blackhole_token`.
  3. Wallet service intercepts inbound requests:
     - Parses `blackhole_token` or `Authorization: Bearer <token>`.
     - Validates payload locally via RS256/HMAC, or delegates verification to `https://pravesh.blackholeinfiverse.com/api/me`.
     - Maps verified user identity to on-chain wallet address.

---

## 4. Cybersecurity Cell Integration (Enforcement Boundary)

* **Implementation:** `core/relay-chain/cybersecurity/security_manager.go`
* **API Endpoints:**
  - `POST /api/cybercrime/freeze`: Freezes compromised addresses upon authorized notice.
  - `POST /api/cybercrime/unfreeze`: Restores frozen address after clearance.
  - `GET /api/cybercrime/status?address=<addr>`: Returns current freeze status.
* **Enforcement Gate in Consensus:**
  - Inside `core/relay-chain/chain/blockchain.go`, the transaction processing loop queries `IsAddressFrozen(sender)`.
  - If frozen, the transaction is rejected immediately with HTTP 403 / Error `ADDRESS_FROZEN_BY_CYBERSECURITY`.

---

## 5. Cross-Chain Bridge Integration

* **Implementation:** [`bridge-sdk/`](file:///d:/Go%20Lang/MP/block-chain-updated/bridge-sdk) & [`bridge/`](file:///d:/Go%20Lang/MP/block-chain-updated/bridge)
* **Contract Address (Polygon Amoy Testnet):** `0x64Dd946E7425898FE8FaB6D36eA050071190be40`
* **Cross-Chain Relay Endpoints:**
  - `POST /api/bridge/deposit`: Initiates lock on source chain and emits deposit event.
  - `POST /api/bridge/claim`: Releases/mints wrapped token on destination chain upon multi-sig attestation.
  - `GET /health` (Port 8084): Diagnostic endpoint returning active relayer and contract status.
