# Changelog — VAJRA Blockchain & Governance Convergence

All notable changes to the VAJRA sovereign blockchain, trust substrate, and TANTRA governance subsystems are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [2.3.0] - 2026-09-26 — Ishan Shirode Go Apprenticeship & Production Support

### Added
- **Go Learning Reference Guide (`GO_LEARNING.md`):** Complete formalization of Go syntax, structs, concurrency, interfaces, cryptographic primitives, and error handling applied directly to the VAJRA runtime.
- **Architectural Subsystem Map (`VAJRA_CODEBASE_MAP.md`):** Comprehensive mapping of all 28+ directories, dependencies, ownership, APIs, tests, and known gaps across the codebase.
- **11-Stage Transaction Lifecycle Flow (`BLOCKCHAIN_FLOW.md`):** Detailed analysis of `Wallet → Transaction → Validation → Signature → Nonce → Runtime → Consensus → State → Evidence → Replay → API/RPC`.
- **Production Diagnostics Engine (`core/relay-chain/diagnostics/`):** Added non-intrusive subsystem health and cryptographic integrity inspector for running nodes.
- **Diagnostics API Routes:** Registered `/api/diagnostics` and `/api/v1/diagnostics` in `APIServer`.
- **Standalone Diagnostic CLI (`core/relay-chain/cmd/vajra-inspect/main.go`):** Operator utility for health probes, state inspection, pipeline self-testing, and Akashic lineage verification.
- **Lifecycle Integration Test Suite (`core/relay-chain/runtime/integration_lifecycle_test.go`):** Automated demonstration covering valid transactions, schema violations, bad signatures, and nonce replay rejections.

---

## [2.2.0] - 2026-09-22 — Phase 2: Enterprise Governance & RBAC (T-GOV-002)

### Added
- **Asymmetric RS256 Token Engine:** Added `security/jwt_handler.py` supporting RSA 2048/4096-bit public/private key pairs and RFC 7519 compliance.
- **Enterprise RBAC Boundaries:** Enforced granular role-gating (`admin`, `auditor`, `validator`, `operator`) and permissions checking with admin role override.
- **Pravah Replay Mitigation Table:** Added `security/replay_mitigation.py` providing monotonic nonce checking, sliding-window deduplication, and cryptographic `ReplayProof` issuance.
- **Pravah Replay Ledger Store:** Persisted validated nonces to `pravah_replay_ledger.jsonl`.
- **Immutable Audit Emitter:** Added `security/audit_emitter.py` with continuous SHA-256 hash chaining ($\text{CurrentHash} = \text{SHA256}(\text{PrevHash} \parallel \dots)$) and `verify_audit_chain()` utility.
- **Deterministic State Machine:** Added `security/state_machine.py` enforcing strict transitions and guaranteed identical SHA-256 state hashes across 50 consecutive runs.
- **Global Error Boundary Middleware:** Added `ErrorBoundaryMiddleware` in `security/middleware.py` intercepting panics and malformed inputs, guaranteeing zero unhandled 500 crashes.
- **Multi-Gate Rule Engine:** Added `evaluation_engine/rule_engine.py` evaluating RBAC, Replay, Crypto, Determinism, and Robustness rules.
- **PARIKSHAK Review Orchestrator:** Added `task_selector/review_orchestrator.py` automating quality gate verification.
- **Resilient Local LLM Integration:** Added `integrations/local_llm_client.py` with deterministic offline fallback mode.
- **Production FastAPI Application:** Added `api/production.py` and `api/schemas.py` enforcing `extra='forbid'` on all schemas.
- **Automated Pytest Suite:** Added 45 tests across `tests/unit/` and `tests/integration/` achieving 100% pass rate.

### Changed
- Strictly pinned all dependencies in `requirements.txt` with `==` to prevent version drift.

---

## [2.1.0] - 2026-09-21 — Pravesh SSO Authentication Integration

### Added
- **Pravesh SSO Hybrid Authentication:** Integrated `services/wallet/jwt_auth.go` checking `blackhole_token` and delegating remote token validation to `https://pravesh.blackholeinfiverse.com/api/me`.
- **Pravesh Test Suite:** Added `services/wallet/pravesh_auth_test.go` with 100% pass rate.
- **Cross-Subdomain Configuration:** Updated `docker-compose.production.template.yml` and `docker-compose.yml` with `PRAVESH_AUTH_URL`.

---

## [2.0.0] - 2026-09-18 — Full PDV Convergence & 11-Hop E2E Verification

### Added
- **11-Hop E2E Verification Runner:** Added `core/relay-chain/cmd/pdv-convergence-e2e/main.go` verifying 11 architectural hops.
- **3-Agent Parallel PDV Agreement Gate:** Enforced $H_{\text{exec}} == H_{\text{val}} == H_{\text{replay}}$ with hard HTTP 403 rejection on mismatch.
- **Global HTTP Panic Recovery Middleware:** Enforced deferred panic interception protecting nodes from crashing on malformed payloads.
- **Dual Evidence Stores:** Committed execution hashes to `tantra_truth.jsonl` and historical state roots to `akashic_lineage.jsonl`.
- **Unit Test Expansion:** Expanded test matrix across 26 Go packages achieving 78/78 passing tests.

---

## [1.9.0] - 2026-09-10 — Proof of Stake Consensus & Tokenomics Locking

### Added
- **Proof of Stake (PoS) Engine:** Added `core/relay-chain/consensus/pos.go` with stake-weighted pseudorandom validator election.
- **Dynamic Reward Inflation Manager:** Implemented inflation manager adjusting staking yields based on network participation.
- **1 Billion BHIV Token Hard Cap:** Code-level invariant gates in `core/relay-chain/token/token.go` strictly preventing minting beyond 1,000,000,000 tokens.

---

## [1.8.0] - 2026-09-01 — 5-Node Topology & libp2p Networking

### Added
- **libp2p Transport Protocol:** Deployed `/blackhole/1.0.0` over TCP on ports `3000`–`3004`.
- **5-Node Cluster Topology:** Defined in `docker-compose.cluster.yml` (`blackhole-node-1` through `5`).
- **Dynamic Bootnode Discovery:** Added `/api/p2p/info` endpoint for dynamic multiaddr discovery.
