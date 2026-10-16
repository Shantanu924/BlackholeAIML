# VAJRA Intelligence Layer Handover

## Overview
This document covers the complete handover for Phase 6 of the VAJRA Intelligence Layer. The intelligence layer provides an **advisory-only** AI/ML risk assessment and decision engine. It operates with a **LIVE/SIMULATED** toggle via the `VAJRA_INTERNAL_URL` configuration.

## Architecture
The system processes events through a 7-stage intelligence pipeline:
1. **Normalisation:** Ingestion of raw VAJRA events.
2. **AI/ML Risk Assessment:** Rule-based engine scoring event risk (rule-engine-v1).
3. **Decision Gate:** Combining AI risk score with VAJRA's internal authorization.
4. **Advisory Evidence Recording:** Hash-chain logging to local SQLite for audit trails.
5. **Verification:** Validating integrity of the local evidence hash chain.
6. **VAJRA Audit Emit:** Adapter that pushes audits back to the core VAJRA system.
7. **Observable Result:** Return payload and logging of the full replay trace.

## Run Steps
1. **Setup Environment:**
   ```bash
   python -m venv venv
   source venv/bin/activate  # On Windows: venv\Scripts\activate
   pip install -r requirements.txt
   ```
2. **Start the Service:**
   ```bash
   uvicorn main:app --reload --port 8100
   ```
3. **Docker:**
   ```bash
   docker build -t vajra-intelligence .
   docker run -p 8100:8100 --env-file .env vajra-intelligence
   ```

## Configuration Guide (`.env` Variables)
- `VAJRA_INTERNAL_URL`: Internal API for LIVE mode (default is empty, triggering SIMULATED mode).
- `VAJRA_API_KEY`: Key for authenticated requests to VAJRA.
- `VAJRA_BASE_URL`: Public VAJRA host.
- `LOG_LEVEL`: Logging verbosity (default `INFO`).
- `HOST`: Server host (default `0.0.0.0`).
- `PORT`: Server port (default `8100`).

## Known Gaps and Blockers
- **BLOCKER-001**: No public VAJRA API endpoints are accessible externally. Adapters (`vajra-event-stream`, `vajra-audit-emit`, `vajra-replay-verify`) operate in SIMULATED mode by default unless `VAJRA_INTERNAL_URL` is set.

## Rollback Procedure
If the Intelligence Layer affects processing speed or stability, disable it at the source, or stop the FastAPI service. The core VAJRA platform does not depend on the Intelligence Layer to function; AI decisions are strictly advisory.

## Integration Points
- **Audit Emit:** Pushes AI verdicts to VAJRA.
- **Replay Verify:** Submits old event IDs to VAJRA to verify historical integrity.
- **Event Stream:** Pulls recent transaction data.
- **Health Check:** Liveness and adapter status metrics.

## Security Considerations
- **Advisory-Only:** AI models do not hold enforcement authority; enforcement belongs to the VAJRA runtime.
- **No Secrets in Code:** Configuration is strictly environment-based.
- **Non-Root Docker:** The provided container should run as a non-root user.
- Hash chains prevent tampering of local logs.

## Monitoring and Observability
- `GET /intelligence/health`: Service health and adapter status matrix.
- `GET /intelligence/adapters`: Complete adapter capabilities and blocker info.
- Replay logs are continually written to `data/runtime_logs/intelligence_replay.jsonl`.
