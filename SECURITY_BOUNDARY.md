# VAJRA Security Boundary & Cryptographic Enforcement Specification

**Document Specification:** Security Architecture & Cryptographic Threat Boundaries  
**Owner:** Prakash Kumar (VAJRA Blockchain Core, Mainnet & Production Runtime Owner)  
**Constitutional Domain:** Trust Layer / Settlement Substrate  
**Status:** Production Certified  

---

## 1. Security Architecture & Threat Boundaries

VAJRA enforces multi-tier defense-in-depth across six distinct cryptographic boundaries:

```
┌──────────────────────────────────────────────────────────────────────────┐
│ Boundary 1: Network Ingress & TLS Termination (Nginx Reverse Proxy)      │
│ - Rate limiting (100 req/min per IP), DDoS filtering, TLS 1.3 only       │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     │
┌────────────────────────────────────▼─────────────────────────────────────┐
│ Boundary 2: Global Panic Recovery & Error Boundary Safety                │
│ - Zero-crash guarantee (traps unhandled panics and malformed JSON)       │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     │
┌────────────────────────────────────▼─────────────────────────────────────┐
│ Boundary 3: Enterprise RBAC & Asymmetric RS256 Token Gating              │
│ - RSA 2048/4096-bit public key validation, strict audience/time checks   │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     │
┌────────────────────────────────────▼─────────────────────────────────────┐
│ Boundary 4: Replay Mitigation Table & Monotonic Nonce Ledger             │
│ - Deduplication window, duplicate hash rejection, monotonic nonces       │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     │
┌────────────────────────────────────▼─────────────────────────────────────┐
│ Boundary 5: Cybersecurity Cell Address Freeze Gate                       │
│ - Real-time address freeze enforcement, blocking blacklisted funds       │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     │
┌────────────────────────────────────▼─────────────────────────────────────┐
│ Boundary 6: Consensus 3-Agent Equality Gate (H_exec == H_val == H_replay)│
│ - Immutable TruthStore append and continuous SHA-256 hash chaining       │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Boundary Detailed Specifications

### 2.1 Boundary 1: Network Ingress & Reverse Proxy
* **Enforced in:** [`deploy/mainnet/nginx.conf`](file:///d:/Go%20Lang/MP/block-chain-updated/deploy/mainnet/nginx.conf)
* **Controls:** 
  - Restricts maximum request body size to 2 MB.
  - Rejects insecure HTTP requests with 301 redirects to HTTPS.
  - Implements burst rate-limiting preventing API flooding attacks.

### 2.2 Boundary 2: Global Panic Recovery & Error Boundaries
* **Enforced in:** [`core/relay-chain/api/server.go`](file:///d:/Go%20Lang/MP/block-chain-updated/core/relay-chain/api/server.go) and [`security/middleware.py`](file:///d:/Go%20Lang/MP/block-chain-updated/security/middleware.py)
* **Controls:**
  - Wraps all HTTP handlers in deferred panic recovery blocks.
  - Traps malformed JSON payloads and type errors, safely returning structured 4xx envelopes (`MALFORMED_INPUT`).
  - Guarantees that no malformed transaction or hostile payload can crash node processes.

### 2.3 Boundary 3: Enterprise RBAC & RS256 Token Verification
* **Enforced in:** [`security/jwt_handler.py`](file:///d:/Go%20Lang/MP/block-chain-updated/security/jwt_handler.py)
* **Controls:**
  - Validates asymmetric RSA signatures using PKCS1v15 + SHA256.
  - Validates `exp` (token expiration), `nbf` (not before), and `aud` (`vajra.production.runtime`).
  - Gating rules:
    - `admin`: Full system control and emergency maintenance overrides.
    - `auditor`: Read-only access to TruthStore, audit chains, and governance reports.
    - `validator`: Permitted to participate in consensus block signing and relaying.
    - `operator`: Standard API submission and node status access.

### 2.4 Boundary 4: Replay Mitigation Table & Monotonic Nonces
* **Enforced in:** [`security/replay_mitigation.py`](file:///d:/Go%20Lang/MP/block-chain-updated/security/replay_mitigation.py)
* **Controls:**
  - Monotonic check: Rejects any nonce where $nonce \le last\_seen\_nonce$.
  - Canonical payload hash dedup: Rejects identical payload SHA-256 fingerprints seen within the sliding TTL window (300s).
  - JTI check: Prevents JWT reuse across multiple requests.
  - Cryptographic proof: Generates signed `ReplayProof` objects on every rejection.

### 2.5 Boundary 5: Cybersecurity Freeze Enforcement
* **Enforced in:** [`core/relay-chain/chain/blockchain.go`](file:///d:/Go%20Lang/MP/block-chain-updated/core/relay-chain/chain/blockchain.go)
* **Controls:**
  - Validates whether `IsAddressFrozen(tx.Sender)` returns true.
  - Rejection is non-bypassable: Frozen transactions never enter the mempool or proposed blocks.
  - Freeze modifications require signed authorization from the Cybersecurity Cell.

### 2.6 Boundary 6: Consensus 3-Agent Equality Gate
* **Enforced in:** [`core/relay-chain/enforcement/tantra.go`](file:///d:/Go%20Lang/MP/block-chain-updated/core/relay-chain/enforcement/tantra.go)
* **Controls:**
  - Three distinct agents independently evaluate incoming transactions: `ExecutionAgent`, `ValidationAgent`, and `ReplayAgent`.
  - Equality invariant: $H_{\text{exec}} == H_{\text{val}} == H_{\text{replay}}$.
  - If any hash deviates, the transaction is immediately aborted with HTTP 403.
