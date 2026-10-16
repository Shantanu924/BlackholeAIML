"""
TANTRA Enterprise Security Subsystem
Provides RS256 JWT validation, enterprise RBAC, replay mitigation,
audit log generation, and deterministic state machine transitions.
"""

from .jwt_handler import JWTHandler, TokenPayload, RS256SecurityEngine
from .replay_mitigation import ReplayMitigationTable, ReplayRecord, ReplayProof
from .audit_emitter import PravahAuditEmitter, AuditEvent, verify_audit_chain
from .state_machine import DeterministicStateMachine, StateTransitionError, verify_deterministic_50_runs
from .middleware import SecurityMiddleware, ErrorBoundaryMiddleware

__all__ = [
    "JWTHandler",
    "TokenPayload",
    "RS256SecurityEngine",
    "ReplayMitigationTable",
    "ReplayRecord",
    "ReplayProof",
    "PravahAuditEmitter",
    "AuditEvent",
    "verify_audit_chain",
    "DeterministicStateMachine",
    "StateTransitionError",
    "verify_deterministic_50_runs",
    "SecurityMiddleware",
    "ErrorBoundaryMiddleware",
]
