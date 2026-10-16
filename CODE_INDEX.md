# CODE INDEX — TANTRA Governance Subsystem (T-GOV-002)

**Author:** Prakash Yadav  
**Task ID:** `T-GOV-002`  
**Product Governance:** TANTRA  
**Authority:** PARIKSHAK Production Runtime Engine & NIYANTRAN Task Orchestration  

---

## 1. Directory Tree

```
block-chain-updated/
├── requirements.txt                         # Pinned dependencies (== exact versions)
├── security/
│   ├── __init__.py                          # Security package exports
│   ├── jwt_handler.py                       # RS256 token engine & enterprise RBAC
│   ├── replay_mitigation.py                 # Pravah replay table & cryptographic proof engine
│   ├── audit_emitter.py                     # Immutable audit emitter & SHA-256 hash chaining
│   ├── state_machine.py                     # Deterministic state machine & 50-run invariance
│   └── middleware.py                        # RS256 RBAC dependency & ErrorBoundaryMiddleware
├── evaluation_engine/
│   ├── __init__.py                          # Evaluation engine package exports
│   ├── models.py                            # Strict Pydantic models (extra='forbid')
│   └── rule_engine.py                       # 5-gate rule evaluation engine
├── integrations/
│   ├── __init__.py                          # Integrations package exports
│   └── local_llm_client.py                  # Resilient local LLM client & deterministic fallback
├── task_selector/
│   ├── __init__.py                          # Task selector package exports
│   └── review_orchestrator.py               # PARIKSHAK review orchestrator & quality gates
├── api/
│   ├── __init__.py                          # API package exports
│   ├── schemas.py                           # Strict Pydantic API schemas (extra='forbid')
│   └── production.py                        # Production FastAPI application & route endpoints
├── tests/
│   ├── __init__.py                          # Test suite package exports
│   ├── conftest.py                          # Session fixtures (key pairs, client, ledgers)
│   ├── unit/
│   │   ├── __init__.py                      # Unit test package exports
│   │   ├── test_security_middleware.py      # RS256 JWT, RBAC, signature & expiration tests
│   │   ├── test_replay_mitigation.py        # Nonce tracking, replay proofs, TTL & concurrency
│   │   ├── test_audit_emitter.py            # SHA-256 chaining & tamper detection tests
│   │   ├── test_state_machine.py            # State transitions & 50-run identical state hash
│   │   ├── test_rule_engine.py              # 5-gate rule evaluation & scoring tests
│   │   ├── test_local_llm_client.py         # Offline fallback & structured summary parsing
│   │   └── test_review_orchestrator.py      # Dependency pinning & candidate gate evaluation
│   └── integration/
│       ├── __init__.py                      # Integration test package exports
│       ├── test_production_api.py           # Production FastAPI routes & authorization
│       ├── test_error_boundaries.py         # Zero unhandled 500 error boundary tests
│       ├── test_pravah_streaming.py         # End-to-end event-driven streaming to Pravah
│       └── test_deterministic_concurrency.py# 50 parallel/consecutive runs state hash invariance
├── REVIEW_PACKET.md                         # Formal review packet & PARIKSHAK certification
├── CODE_PACKET.md                           # Changed files & git diff summary
└── CODE_INDEX.md                            # Complete architectural symbol index
```

---

## 2. Module & Symbol Reference

### `security` Subsystem

| File | Symbol | Type | Description |
|---|---|---|---|
| [`security/jwt_handler.py`](file:///d:/Go%20Lang/MP/block-chain-updated/security/jwt_handler.py) | `RS256SecurityEngine` | Class | Asymmetric RSA key pair manager; signs and verifies RS256 JWTs using PKCS1v15 + SHA256. |
| [`security/jwt_handler.py`](file:///d:/Go%20Lang/MP/block-chain-updated/security/jwt_handler.py) | `TokenPayload` | Pydantic Model | Strict JWT payload model (`sub`, `roles`, `permissions`, `tenant_id`, `jti`, `exp`, `iat`). |
| [`security/jwt_handler.py`](file:///d:/Go%20Lang/MP/block-chain-updated/security/jwt_handler.py) | `JWTHandler` | Class | High-level token issuance and RBAC check operations. |
| [`security/replay_mitigation.py`](file:///d:/Go%20Lang/MP/block-chain-updated/security/replay_mitigation.py) | `ReplayMitigationTable` | Class | Thread-safe sliding-window table tracking nonces, payload SHA-256 hashes, and JTIs. |
| [`security/replay_mitigation.py`](file:///d:/Go%20Lang/MP/block-chain-updated/security/replay_mitigation.py) | `ReplayProof` | Pydantic Model | Verifiable cryptographic proof of replay check status. |
| [`security/audit_emitter.py`](file:///d:/Go%20Lang/MP/block-chain-updated/security/audit_emitter.py) | `PravahAuditEmitter` | Class | Asynchronous streaming audit emitter; maintains continuous SHA-256 hash chaining. |
| [`security/audit_emitter.py`](file:///d:/Go%20Lang/MP/block-chain-updated/security/audit_emitter.py) | `verify_audit_chain` | Function | Validates cryptographic lineage across all historical records; pinpoints tampered blocks. |
| [`security/state_machine.py`](file:///d:/Go%20Lang/MP/block-chain-updated/security/state_machine.py) | `DeterministicStateMachine` | Class | Concurrency-safe state machine enforcing legal transitions and deterministic state hashing. |
| [`security/state_machine.py`](file:///d:/Go%20Lang/MP/block-chain-updated/security/state_machine.py) | `verify_deterministic_50_runs` | Function | Executes 50 independent lifecycle runs, asserting identical SHA-256 digests. |
| [`security/middleware.py`](file:///d:/Go%20Lang/MP/block-chain-updated/security/middleware.py) | `ErrorBoundaryMiddleware` | Class | Global ASGI middleware intercepting panics and unhandled exceptions (zero 500 crashes). |
| [`security/middleware.py`](file:///d:/Go%20Lang/MP/block-chain-updated/security/middleware.py) | `SecurityMiddleware` | Class | FastAPI dependency enforcing Bearer RS256 JWT auth and RBAC roles/permissions. |

---

### `evaluation_engine` Subsystem

| File | Symbol | Type | Description |
|---|---|---|---|
| [`evaluation_engine/models.py`](file:///d:/Go%20Lang/MP/block-chain-updated/evaluation_engine/models.py) | `RuleDefinition` | Pydantic Model | Architectural rule specification with weight and category. |
| [`evaluation_engine/models.py`](file:///d:/Go%20Lang/MP/block-chain-updated/evaluation_engine/models.py) | `RuleEvaluationResult` | Pydantic Model | Individual rule verdict, score, and evidence digest. |
| [`evaluation_engine/models.py`](file:///d:/Go%20Lang/MP/block-chain-updated/evaluation_engine/models.py) | `EvaluationContext` | Pydantic Model | Inbound context for candidate rule execution. |
| [`evaluation_engine/models.py`](file:///d:/Go%20Lang/MP/block-chain-updated/evaluation_engine/models.py) | `GateVerdict` | Pydantic Model | Aggregate PARIKSHAK quality gate verdict and proof digest. |
| [`evaluation_engine/rule_engine.py`](file:///d:/Go%20Lang/MP/block-chain-updated/evaluation_engine/rule_engine.py) | `RuleEngine` | Class | Multi-gate evaluator checking RBAC, Replay, Crypto, Determinism, and Robustness. |

---

### `integrations` Subsystem

| File | Symbol | Type | Description |
|---|---|---|---|
| [`integrations/local_llm_client.py`](file:///d:/Go%20Lang/MP/block-chain-updated/integrations/local_llm_client.py) | `LocalLLMClient` | Class | Resilient client connecting to local Ollama/vLLM endpoints with retries and timeout. |
| [`integrations/local_llm_client.py`](file:///d:/Go%20Lang/MP/block-chain-updated/integrations/local_llm_client.py) | `LLMReviewSummary` | Pydantic Model | Structured task compliance summary output. |

---

### `task_selector` Subsystem

| File | Symbol | Type | Description |
|---|---|---|---|
| [`task_selector/review_orchestrator.py`](file:///d:/Go%20Lang/MP/block-chain-updated/task_selector/review_orchestrator.py) | `ReviewOrchestrator` | Class | Orchestrates formal verification of submissions against PARIKSHAK quality gates. |
| [`task_selector/review_orchestrator.py`](file:///d:/Go%20Lang/MP/block-chain-updated/task_selector/review_orchestrator.py) | `TaskSubmission` | Pydantic Model | Inbound candidate submission specification. |
| [`task_selector/review_orchestrator.py`](file:///d:/Go%20Lang/MP/block-chain-updated/task_selector/review_orchestrator.py) | `EvaluationReport` | Pydantic Model | Certified evaluation report with score, gates, and proof digest. |

---

### `api` Subsystem

| File | Route | Method | Role |
|---|---|---|---|
| [`api/production.py`](file:///d:/Go%20Lang/MP/block-chain-updated/api/production.py) | `/health` / `/api/v1/health` | GET | System health & runtime version. |
| [`api/production.py`](file:///d:/Go%20Lang/MP/block-chain-updated/api/production.py) | `/api/v1/auth/public-key` | GET | RS256 RSA public key in PEM format. |
| [`api/production.py`](file:///d:/Go%20Lang/MP/block-chain-updated/api/production.py) | `/api/v1/auth/jwks` | GET | RFC 7517 JSON Web Key Set. |
| [`api/production.py`](file:///d:/Go%20Lang/MP/block-chain-updated/api/production.py) | `/api/v1/auth/token` | POST | Issue signed RS256 governance JWT. |
| [`api/production.py`](file:///d:/Go%20Lang/MP/block-chain-updated/api/production.py) | `/api/v1/auth/verify` | POST | Verify RS256 token validity and claims. |
| [`api/production.py`](file:///d:/Go%20Lang/MP/block-chain-updated/api/production.py) | `/api/v1/replay/verify` | POST | Verify monotonic nonce & dedup against replay table. |
| [`api/production.py`](file:///d:/Go%20Lang/MP/block-chain-updated/api/production.py) | `/api/v1/replay/ledger` | GET | Query registered Pravah replay records. |
| [`api/production.py`](file:///d:/Go%20Lang/MP/block-chain-updated/api/production.py) | `/api/v1/audit/emit` | POST | Emit cryptographically chained audit event. |
| [`api/production.py`](file:///d:/Go%20Lang/MP/block-chain-updated/api/production.py) | `/api/v1/audit/logs` | GET | Query recent Pravah audit ledger events. |
| [`api/production.py`](file:///d:/Go%20Lang/MP/block-chain-updated/api/production.py) | `/api/v1/audit/verify-chain` | GET | Cryptographic validation of entire audit chain. |
| [`api/production.py`](file:///d:/Go%20Lang/MP/block-chain-updated/api/production.py) | `/api/v1/governance/evaluate` | POST | Execute multi-gate rule evaluation with RBAC auth. |
| [`api/production.py`](file:///d:/Go%20Lang/MP/block-chain-updated/api/production.py) | `/api/v1/governance/transition` | POST | Atomic state machine transition with RBAC auth. |
| [`api/production.py`](file:///d:/Go%20Lang/MP/block-chain-updated/api/production.py) | `/api/v1/governance/state-hash` | GET | Retrieve deterministic SHA-256 state hash. |
| [`api/production.py`](file:///d:/Go%20Lang/MP/block-chain-updated/api/production.py) | `/api/v1/orchestrator/evaluate-task`| POST | Execute PARIKSHAK quality gate evaluation. |
