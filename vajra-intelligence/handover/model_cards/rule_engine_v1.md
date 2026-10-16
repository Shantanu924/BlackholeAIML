# Model Card: rule-engine-v1

## Model Details
- **Name**: rule-engine-v1
- **Version**: 1.0.0
- **Type**: Deterministic rule-based risk scoring engine

## I/O Schemas
- **Input Schema**: `IntelligenceEvent` (Pydantic model)
- **Output Schema**: `AssessmentResult` (Pydantic model)

## 8 Rules and Scoring
1. **Unknown/unrecognised user_id**: +3 (Currently placeholder for identity adapter)
2. **VAJRA authorization denied**: +3
3. **High-value action**: +2 (DELETE, EXPORT, TRANSFER, STAKE, BRIDGE, VOTE, UNSTAKE)
4. **Sensitive resource targeted**: +2
5. **Off-hours access**: +1 (UTC 00–06)
6. **Frequency abuse**: +2 (>5 events / minute)
7. **Cross-chain bridge activity**: +1
8. **Validator / staking operation**: +1

## Score-to-Risk Mapping
| Score | Risk Level | Status Flag | Recommended Action | Confidence |
|-------|------------|-------------|--------------------|------------|
| 0–1   | LOW        | CLEAR       | ALLOW              | 0.90       |
| 2–3   | MEDIUM     | FLAGGED     | REVIEW             | 0.75       |
| 4–5   | HIGH       | BLOCKED     | DENY               | 0.85       |
| 6+    | CRITICAL   | BLOCKED     | DENY               | 0.95       |

## Limitations
- **No ML Model**: This is purely a deterministic rule engine, not a trained machine learning model.
- **Rule-Based Only**: Cannot learn from historical trends automatically.
- **No User Identity Store**: Relies completely on VAJRA's internal authorization state.

## Ethical Considerations
- **Advisory-Only**: The output is purely an AI recommendation. It holds no cybersecurity enforcement authority. The VAJRA runtime makes the final decision.

## Performance
- **Deterministic**: Same input precisely generates the exact same output score and feature trace.
- **Compute**: No GPU required. Sub-millisecond execution time.
