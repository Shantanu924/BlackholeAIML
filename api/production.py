"""
Production FastAPI Application for TANTRA Governance & VAJRA Trust Layer
Equipped with RS256 RBAC security, ErrorBoundaryMiddleware, Pravah replay ledger,
and PARIKSHAK quality gate evaluation endpoints.
"""

import time
import uuid
from typing import Any, Dict, List

from fastapi import Depends, FastAPI, HTTPException, Request, Response, status
from fastapi.exceptions import RequestValidationError
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

from evaluation_engine.models import EvaluationContext, GateVerdict
from evaluation_engine.rule_engine import RuleEngine, get_default_rule_engine
from security.audit_emitter import PravahAuditEmitter, get_default_audit_emitter, verify_audit_chain
from security.jwt_handler import JWTHandler, TokenPayload, get_default_security_engine
from security.middleware import ErrorBoundaryMiddleware, SecurityMiddleware
from security.replay_mitigation import ReplayMitigationTable, get_default_replay_table
from security.state_machine import (
    DeterministicStateMachine,
    State,
    StateTransitionError,
    verify_deterministic_50_runs,
)
from task_selector.review_orchestrator import ReviewOrchestrator, TaskSubmission, get_default_orchestrator

from .schemas import (
    AuditEventResponse,
    AuditEmitRequest,
    AuditVerifyChainResponse,
    ErrorEnvelope,
    ReplayVerifyRequest,
    ReplayVerifyResponse,
    RuleEvaluateRequest,
    StateHashResponse,
    StateTransitionRequest,
    StateTransitionResponse,
    TaskReviewRequest,
    TaskReviewResponse,
    TokenRequest,
    TokenResponse,
    VerifyTokenRequest,
    VerifyTokenResponse,
)

app = FastAPI(
    title="TANTRA Governance & VAJRA Trust Layer API",
    description="Enterprise RBAC, Replay Proof Verification, and Automated Audit Emitter Runtime (T-GOV-002)",
    version="2.0.0",
)

# 1. Global Error Boundary Middleware (First to wrap all requests)
app.add_middleware(ErrorBoundaryMiddleware)

# 2. CORS Middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Custom validation exception handler to ensure standard structured JSON envelopes on 422
@app.exception_handler(RequestValidationError)
async def validation_exception_handler(request: Request, exc: RequestValidationError) -> JSONResponse:
    trace_id = getattr(request.state, "trace_id", f"trace-{uuid.uuid4().hex[:12]}")
    return JSONResponse(
        status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
        content={
            "error": True,
            "code": "MALFORMED_INPUT",
            "message": "Inbound request schema validation failed.",
            "detail": str(exc.errors()),
            "details": exc.errors(),
            "trace_id": trace_id,
            "timestamp": time.time(),
        },
        headers={"X-Trace-ID": trace_id},
    )


@app.exception_handler(HTTPException)
async def http_exception_handler(request: Request, exc: HTTPException) -> JSONResponse:
    trace_id = getattr(request.state, "trace_id", f"trace-{uuid.uuid4().hex[:12]}")
    code = (
        "UNAUTHORIZED"
        if exc.status_code == 401
        else "FORBIDDEN"
        if exc.status_code == 403
        else "REPLAY_DETECTED"
        if exc.status_code == 409
        else "HTTP_ERROR"
    )
    return JSONResponse(
        status_code=exc.status_code,
        content={
            "error": True,
            "code": code,
            "message": str(exc.detail),
            "detail": exc.detail,
            "trace_id": trace_id,
            "timestamp": time.time(),
        },
        headers={"X-Trace-ID": trace_id},
    )


# Core Shared State & Singletons
security_engine = get_default_security_engine()
jwt_handler = JWTHandler(security_engine)
sec_middleware = SecurityMiddleware(jwt_handler)
replay_table = get_default_replay_table()
audit_emitter = get_default_audit_emitter()
rule_engine = get_default_rule_engine()
orchestrator = get_default_orchestrator()
global_state_machine = DeterministicStateMachine(task_id="T-GOV-002", candidate_id="Prakash Yadav")


# ==========================================
# Health & JWKS Discovery Routes
# ==========================================

@app.get("/health", tags=["System"])
@app.get("/api/v1/health", tags=["System"])
def get_health() -> Dict[str, Any]:
    """Production health check returning active runtime status."""
    return {
        "status": "HEALTHY",
        "service": "TANTRA Governance & VAJRA Trust Layer",
        "task_id": "T-GOV-002",
        "authority": "PARIKSHAK Production Runtime Engine",
        "version": "2.0.0",
        "timestamp": time.time(),
    }


@app.get("/api/v1/auth/public-key", tags=["Authentication"])
def get_public_key() -> Dict[str, str]:
    """Retrieve active RS256 RSA public key for peer and external verification."""
    return {
        "alg": "RS256",
        "format": "PKCS#8 PEM",
        "public_key": security_engine.get_public_key_pem(),
    }


@app.get("/api/v1/auth/jwks", tags=["Authentication"])
def get_jwks() -> Dict[str, Any]:
    """Retrieve RFC 7517 compliant JSON Web Key Set (JWKS)."""
    return security_engine.get_jwks()


# ==========================================
# Authentication & RBAC Routes
# ==========================================

@app.post("/api/v1/auth/token", response_model=TokenResponse, tags=["Authentication"])
def generate_token(req: TokenRequest) -> TokenResponse:
    """Issue RS256-signed enterprise governance JWT."""
    token = jwt_handler.issue_token(
        sub=req.sub,
        roles=req.roles,
        permissions=req.permissions,
        tenant_id=req.tenant_id,
        candidate_id=req.candidate_id,
        ttl_seconds=req.ttl_seconds,
    )
    payload = jwt_handler.verify_token(token)
    return TokenResponse(
        access_token=token,
        token_type="Bearer",
        expires_in=req.ttl_seconds,
        claims=payload.model_dump(),
    )


@app.post("/api/v1/auth/verify", response_model=VerifyTokenResponse, tags=["Authentication"])
def verify_token_endpoint(req: VerifyTokenRequest) -> VerifyTokenResponse:
    """Verify RS256 JWT signature, expiration, and payload integrity."""
    payload = jwt_handler.verify_token(req.token)
    return VerifyTokenResponse(valid=True, claims=payload.model_dump())


# ==========================================
# Pravah Replay Ledger & Defense Routes
# ==========================================

@app.post("/api/v1/replay/verify", response_model=ReplayVerifyResponse, tags=["Replay Defense"])
def verify_replay(req: ReplayVerifyRequest, request: Request) -> ReplayVerifyResponse:
    """
    Verify transaction uniqueness against Pravah replay ledger.
    Rejects duplicate nonces or replayed payloads with HTTP 409 Conflict.
    """
    trace_id = req.trace_id or getattr(request.state, "trace_id", f"trace-{uuid.uuid4().hex[:12]}")
    is_valid, proof = replay_table.verify_and_register(
        sender_id=req.sender_id,
        nonce=req.nonce,
        payload=req.payload,
        trace_id=trace_id,
    )

    if not is_valid:
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail=f"Replay rejected: {proof.reason} (Proof Digest: {proof.proof_digest})",
        )

    return ReplayVerifyResponse(
        valid=True,
        status=proof.status,
        reason=proof.reason,
        proof_digest=proof.proof_digest,
        trace_id=proof.trace_id,
    )


@app.get("/api/v1/replay/ledger", tags=["Replay Defense"])
def get_replay_ledger(limit: int = 50) -> Dict[str, Any]:
    """Retrieve registered records from Pravah replay ledger."""
    records = replay_table.get_ledger_records(limit=limit)
    return {
        "count": len(records),
        "records": [r.model_dump() for r in records],
    }


# ==========================================
# Pravah Immutable Audit Stream Routes
# ==========================================

@app.post("/api/v1/audit/emit", response_model=AuditEventResponse, tags=["Audit Stream"])
def emit_audit_event(req: AuditEmitRequest) -> AuditEventResponse:
    """Emit an immutable structured event into the Pravah audit stream."""
    evt = audit_emitter.emit(
        event_type=req.event_type,
        actor=req.actor,
        action=req.action,
        resource=req.resource,
        status=req.status,
        metadata=req.metadata,
        payload=req.payload,
    )
    return AuditEventResponse(
        event_id=evt.event_id,
        timestamp=evt.timestamp,
        event_type=evt.event_type,
        actor=evt.actor,
        action=evt.action,
        resource=evt.resource,
        status=evt.status,
        current_hash=evt.current_hash,
        previous_hash=evt.previous_hash,
    )


@app.get("/api/v1/audit/logs", tags=["Audit Stream"])
def get_audit_logs(limit: int = 50) -> Dict[str, Any]:
    """Retrieve recent cryptographically chained audit events."""
    events = audit_emitter.get_recent_events(limit=limit)
    return {
        "count": len(events),
        "events": [e.model_dump() for e in events],
    }


@app.get("/api/v1/audit/verify-chain", response_model=AuditVerifyChainResponse, tags=["Audit Stream"])
def verify_audit_hash_chain() -> AuditVerifyChainResponse:
    """Cryptographically verify the entire Pravah audit hash chain."""
    events = audit_emitter._events
    valid, message = verify_audit_chain(events=events)
    return AuditVerifyChainResponse(valid=valid, message=message)


# ==========================================
# Governance & Deterministic State Machine Routes
# ==========================================

@app.post("/api/v1/governance/evaluate", tags=["Governance"])
def evaluate_governance_rules(
    req: RuleEvaluateRequest,
    user: TokenPayload = Depends(sec_middleware.require_roles(["admin", "auditor", "operator"])),
) -> GateVerdict:
    """Execute multi-gate rule evaluation with RBAC role authorization."""
    ctx = EvaluationContext(
        task_id=req.task_id,
        candidate_id=req.candidate_id,
        actor=user.sub,
        token=req.token,
        sender_id=req.sender_id,
        nonce=req.nonce,
        payload=req.payload,
        run_50_determinism_check=req.run_50_determinism_check,
    )
    return rule_engine.evaluate(ctx)


@app.post("/api/v1/governance/transition", response_model=StateTransitionResponse, tags=["Governance"])
def transition_state(
    req: StateTransitionRequest,
    user: TokenPayload = Depends(sec_middleware.require_roles(["admin", "operator"])),
) -> StateTransitionResponse:
    """Execute an atomic deterministic state transition."""
    try:
        target = State(req.target_state)
    except ValueError:
        raise HTTPException(
            status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
            detail=f"Invalid target state '{req.target_state}'. Allowed states: {[s.value for s in State]}",
        )

    try:
        step_hash = global_state_machine.transition_to(
            target_state=target,
            actor=user.sub,
            reason=req.reason,
            metadata=req.metadata,
        )
        state_hash = global_state_machine.compute_state_hash()
        return StateTransitionResponse(
            success=True,
            current_state=global_state_machine.current_state.value,
            step_hash=step_hash,
            state_hash=state_hash,
        )
    except StateTransitionError as ste:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(ste),
        )


@app.get("/api/v1/governance/state-hash", response_model=StateHashResponse, tags=["Governance"])
def get_state_hash() -> StateHashResponse:
    """Retrieve current deterministic SHA-256 state hash."""
    return StateHashResponse(
        state_hash=global_state_machine.compute_state_hash(),
        current_state=global_state_machine.current_state.value,
        step_count=global_state_machine._step_counter,
    )


# ==========================================
# PARIKSHAK Review Orchestration Routes
# ==========================================

@app.post("/api/v1/orchestrator/evaluate-task", tags=["PARIKSHAK Orchestration"])
def orchestrate_task_review(
    req: TaskReviewRequest,
    user: TokenPayload = Depends(sec_middleware.require_roles(["admin", "auditor"])),
) -> Dict[str, Any]:
    """Execute full automated PARIKSHAK evaluation protocol for candidate task."""
    submission = TaskSubmission(
        task_id=req.task_id,
        candidate_id=req.candidate_id,
        requirements_path=req.requirements_path,
    )
    report = orchestrator.evaluate_submission(submission)
    return report.model_dump()
