# Capability Matrix

| Capability | Status | Description / Notes |
|------------|--------|---------------------|
| Risk assessment | VERIFIED LIVE | Fully implemented via `rule-engine-v1`. |
| Decision gate | VERIFIED LIVE | Fully implemented, reconciling AI risk with VAJRA auth. |
| Evidence recording | VERIFIED LIVE | Hash-chained logging to local SQLite. |
| Hash verification | VERIFIED LIVE | Verification of evidence blocks and previous hashes. |
| Replay | VERIFIED LIVE | Implemented via POST `/intelligence/replay`. |
| VAJRA health check | SIMULATED / LIVE | Available on `/intelligence/health` & `/adapters`. |
| VAJRA audit emit | SIMULATED | Blocked by lack of public VAJRA endpoints. Configurable to LIVE via `.env`. |
| VAJRA replay verify | SIMULATED | Blocked by lack of public VAJRA endpoints. Configurable to LIVE via `.env`. |
| VAJRA event stream | SIMULATED | Blocked by lack of public VAJRA endpoints. Configurable to LIVE via `.env`. |
| Dashboard | PARTIAL | Exposed APIs, no frontend UI provided in this layer. |
| Identity/consent (VAJRA) | BLOCKED / NOT IMPLEMENTED | Pending identity adapter integration (placeholder in rules). |
| Anomaly detection | NOT IMPLEMENTED | Currently uses deterministic rules, not ML-based anomaly models. |
| Model registry | NOT IMPLEMENTED | Single static rule-engine hardcoded. |

**Blockers:**
- **BLOCKER-001**: No public VAJRA API is accessible. Thus, adapters default to SIMULATED mode. Setting `VAJRA_INTERNAL_URL` resolves this when deployed alongside VAJRA.
