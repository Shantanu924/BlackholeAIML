# Reviewer Guide: VAJRA Intelligence Layer

## What to Review
- **Code Structure:** The 7-stage pipeline separation in `api/routes.py`, `intelligence/assessment.py`, etc.
- **Test Coverage:** Ensure rules logic and pipeline components execute reliably.
- **Adapter Degradation:** How the system smoothly falls back to SIMULATED mode if `VAJRA_INTERNAL_URL` is unavailable.
- **Dashboard API Readiness:** Validate the outputs for `GET /intelligence/adapters` and `GET /intelligence/health`.

## How to Verify
1. **Start the server:** `uvicorn main:app --reload --port 8100`
2. **Hit health endpoint:** `curl http://localhost:8100/intelligence/health`
3. **Assess an event:**
   ```bash
   curl -X POST http://localhost:8100/intelligence/assess \
        -H "Content-Type: application/json" \
        -d '{"user_id": "U123", "action": "TRANSFER", "resource": "financial_data", "timestamp": "2026-10-08T00:00:00Z"}'
   ```
4. **Check Dashboard:** Observe how the risk score (Stage 2) propagates into the hash-chained SQLite evidence store.

## Key Design Decisions
- **Advisory-Only AI:** The system cannot block an authorized VAJRA transaction directly. It advises.
- **SIMULATED vs LIVE Toggle:** Decouples Intelligence Layer testing from the core VAJRA API, unblocking development.
- **Hash-Chain Evidence:** Uses sequential local SQLite storage to log verdicts. Provides internal auditable continuity without tampering with VAJRA's blockchain logic.
- **No VAJRA Code Modification:** Entirely independent microservice.
