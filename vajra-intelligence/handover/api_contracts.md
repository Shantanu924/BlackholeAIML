# API Contracts

## 1. POST `/intelligence/assess`
Run full 7-stage intelligence pipeline.

**Request Body (AssessRequest):**
```json
{
  "user_id": "string",
  "action": "string",
  "resource": "string",
  "timestamp": "string",
  "reason": "string",
  "authorized": true,
  "auth_reason": "string",
  "granted_scopes": ["string"],
  "source_system": "VAJRA",
  "metadata": {}
}
```
**Response Body (AssessResponse):**
```json
{
  "event_id": "uuid-string",
  "trace_id": "uuid-string",
  "stage1_event": {},
  "stage2_assessment": {},
  "stage3_decision": {},
  "stage4_evidence": {},
  "stage5_verification": {},
  "stage6_vajra_audit": {},
  "stage7_result": {},
  "pipeline_status": "COMPLETE"
}
```

## 2. GET `/intelligence/verify/{id}`
Verify evidence chain for an event.

**Response:**
```json
{
  "event_id": "string",
  "entry_found": true,
  "integrity_ok": true,
  "detail": "string",
  "verified_at": "datetime",
  "evidence_record": {} 
}
```

## 3. GET `/intelligence/chain`
Full advisory evidence chain.

**Response:**
```json
{
  "total_blocks": 5,
  "chain_intact": true,
  "integrity_detail": "Chain verified.",
  "chain": [
    {}
  ]
}
```

## 4. GET `/intelligence/health`
Service liveness and adapter status matrix.

**Response:**
```json
{
  "status": "ok",
  "service": "vajra-intelligence",
  "version": "1.0.0",
  "timestamp": 123456789.0,
  "vajra_live_mode": false,
  "adapters": [
    {
      "name": "vajra-audit-emit",
      "status": "SIMULATED",
      "endpoint": "/audit"
    }
  ]
}
```

## 5. GET `/intelligence/adapters`
Adapter capability truth matrix.

**Response:**
```json
{
  "vajra_live_mode": false,
  "vajra_base_url": "...",
  "vajra_internal_url": false,
  "adapters": [],
  "blockers": [
    {
       "id": "BLOCKER-001",
       "description": "...",
       "affected_adapters": []
    }
  ]
}
```

## 6. POST `/intelligence/replay`
Replay pipeline on a past event (generates new record).

**Request (ReplayRequest):**
```json
{
  "event_id": "string"
}
```
**Response (ReplayResponse):**
```json
{
  "original_event_id": "string",
  "replayed": true,
  "detail": "Replay complete. New event_id=...",
  "new_report": {}
}
```

## 7. GET `/intelligence/events`
Pull recent events from VAJRA.

**Query Params:** `limit` (integer, default 20)
**Response:** Adapter output payload containing simulated or live events.
