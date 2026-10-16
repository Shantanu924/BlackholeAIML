# VAJRA Intelligence Layer

> AI/ML advisory intelligence layer that attaches to the real VAJRA blockchain runtime.
> All new code lives here — the VAJRA root is **READ-ONLY** and untouched.

---

## Quick Start

```bash
# 1. Create virtual environment
python -m venv .venv
.venv\Scripts\activate          # Windows
# source .venv/bin/activate     # Linux/Mac

# 2. Install dependencies
pip install -r requirements.txt

# 3. Configure environment
copy .env.example .env
# Edit .env — set VAJRA_INTERNAL_URL when internal access is available

# 4. Run the service
uvicorn main:app --reload --port 8100

# API docs:    http://localhost:8100/docs
# Health:      http://localhost:8100/intelligence/health
# Adapters:    http://localhost:8100/intelligence/adapters
```

## Docker

```bash
docker compose up --build
```

## Run Tests

```bash
# From vajra-intelligence/ with .venv active:
pytest tests/ -v --json-report --json-report-file=test_results/report.json
```

---

## Architecture

```
User/Data Event
    → POST /intelligence/assess
        Stage 1: Event Normalisation
        Stage 2: AI/ML Risk Assessment  (rule-engine-v1, deterministic)
        Stage 3: Decision Gate          (ALLOW / FLAG / DENY — ADVISORY ONLY)
        Stage 4: Advisory Evidence      (SQLite hash-chain, not VAJRA chain)
        Stage 5: Hash Verification      (tamper detection)
        Stage 6: VAJRA Audit Emit       (LIVE or SIMULATED adapter)
        Stage 7: Observable Result      (logged to runtime_logs/intelligence_replay.jsonl)
    → IntelligenceReport (full 7-stage output)
```

## LIVE vs SIMULATED

| Component | Default Status | Flip to LIVE |
|---|---|---|
| VAJRA Health | SIMULATED | Set `VAJRA_INTERNAL_URL` |
| VAJRA Audit Emit | SIMULATED | Set `VAJRA_INTERNAL_URL` |
| VAJRA Replay Verify | SIMULATED | Set `VAJRA_INTERNAL_URL` |
| VAJRA Event Stream | SIMULATED | Set `VAJRA_INTERNAL_URL` |
| Risk Assessment Engine | VERIFIED (local) | — |
| Advisory Evidence Store | VERIFIED (local) | — |

## Absolute Rules (enforced in code)

1. **DO NOT** modify VAJRA root code — all work is in `/vajra-intelligence/`
2. AI decisions are **ADVISORY ONLY** — Cybersecurity enforcement authority is VAJRA
3. No secrets in code — use `.env` only
4. No duplicate blockchain — advisory SQLite store is explicitly labeled
5. No raw personal data in logs or evidence records

## Blockers

- **BLOCKER-001**: No public VAJRA API endpoints are accessible externally.
  Set `VAJRA_INTERNAL_URL` in `.env` to flip all adapters from SIMULATED → LIVE.

## Phases

| Phase | Status |
|---|---|
| Phase 0: Inspect | ✅ DONE |
| Phase 1: Scaffold | ✅ DONE |
| Phase 2: Adapters | ⏳ Next |
| Phase 3: Pipeline | ⏳ Pending |
| Phase 4: Tests | ⏳ Pending |
| Phase 5: Dashboard | ⏳ Pending |
| Phase 6: Handover | ⏳ Pending |
