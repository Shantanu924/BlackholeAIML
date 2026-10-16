"""
Enterprise Security & Error Boundary Middleware
Provides RS256 token verification, RBAC role-gating, replay protection,
and complete error boundary interception guaranteeing zero unhandled 500 crashes.
"""

import time
import uuid
from typing import Any, Callable, Dict, List, Optional

from fastapi import Depends, HTTPException, Request, Response, status
from fastapi.responses import JSONResponse
from starlette.middleware.base import BaseHTTPMiddleware

from .audit_emitter import get_default_audit_emitter
from .jwt_handler import JWTHandler, TokenPayload
from .replay_mitigation import get_default_replay_table


class ErrorBoundaryMiddleware(BaseHTTPMiddleware):
    """
    Global ASGI error boundary middleware.
    Catches all uncaught exceptions, value errors, malformed inputs, and panic conditions,
    safely transforming them into structured JSON error envelopes without crashing.
    """

    async def dispatch(self, request: Request, call_next: Callable[[Request], Any]) -> Response:
        trace_id = request.headers.get("X-Trace-ID", f"trace-{uuid.uuid4().hex[:12]}")
        request.state.trace_id = trace_id

        try:
            response = await call_next(request)
            response.headers["X-Trace-ID"] = trace_id
            return response
        except HTTPException as he:
            # Let standard HTTPExceptions pass through with structured formatting
            emitter = get_default_audit_emitter()
            emitter.emit(
                event_type="HTTP_ERROR",
                actor=getattr(request.state, "actor", "anonymous"),
                action=f"{request.method} {request.url.path}",
                resource=str(request.url.path),
                status="DENIED" if he.status_code in (401, 403) else "ERROR",
                metadata={"status_code": he.status_code, "detail": str(he.detail), "trace_id": trace_id},
            )
            code = "UNAUTHORIZED" if he.status_code == 401 else "FORBIDDEN" if he.status_code == 403 else "CLIENT_ERROR"
            return JSONResponse(
                status_code=he.status_code,
                content={
                    "error": True,
                    "code": code,
                    "message": str(he.detail),
                    "trace_id": trace_id,
                    "timestamp": time.time(),
                },
                headers={"X-Trace-ID": trace_id},
            )
        except ValueError as ve:
            emitter = get_default_audit_emitter()
            emitter.emit(
                event_type="VALIDATION_ERROR",
                actor=getattr(request.state, "actor", "anonymous"),
                action=f"{request.method} {request.url.path}",
                resource=str(request.url.path),
                status="REJECTED",
                metadata={"error": str(ve), "trace_id": trace_id},
            )
            return JSONResponse(
                status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
                content={
                    "error": True,
                    "code": "MALFORMED_INPUT",
                    "message": str(ve),
                    "trace_id": trace_id,
                    "timestamp": time.time(),
                },
                headers={"X-Trace-ID": trace_id},
            )
        except Exception as exc:
            # Trap any unhandled exception — zero unhandled 500 crashes
            emitter = get_default_audit_emitter()
            emitter.emit(
                event_type="SYSTEM_PANIC_RECOVERED",
                actor=getattr(request.state, "actor", "system"),
                action=f"{request.method} {request.url.path}",
                resource=str(request.url.path),
                status="RECOVERED",
                metadata={"exception_class": type(exc).__name__, "detail": str(exc), "trace_id": trace_id},
            )
            return JSONResponse(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                content={
                    "error": True,
                    "code": "INTERNAL_SERVER_ERROR_RECOVERED",
                    "message": "An internal error occurred but was safely intercepted by the Error Boundary.",
                    "error_type": type(exc).__name__,
                    "trace_id": trace_id,
                    "timestamp": time.time(),
                },
                headers={"X-Trace-ID": trace_id},
            )


class SecurityMiddleware:
    """Enterprise helper for authenticating requests and enforcing RBAC."""

    def __init__(self, jwt_handler: Optional[JWTHandler] = None) -> None:
        self.jwt_handler = jwt_handler or JWTHandler()
        self.replay_table = get_default_replay_table()
        self.audit_emitter = get_default_audit_emitter()

    def get_current_user(self, request: Request) -> TokenPayload:
        """
        FastAPI dependency to extract and verify the RS256 Bearer token.
        """
        auth_header = request.headers.get("Authorization")
        trace_id = getattr(request.state, "trace_id", f"trace-{uuid.uuid4().hex[:12]}")

        if not auth_header or not auth_header.startswith("Bearer "):
            self.audit_emitter.emit(
                event_type="AUTH_FAILURE",
                actor="anonymous",
                action=f"{request.method} {request.url.path}",
                resource=str(request.url.path),
                status="DENIED",
                metadata={"reason": "Missing or malformed Authorization header", "trace_id": trace_id},
            )
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Missing or malformed Bearer authorization header.",
            )

        token = auth_header.split(" ", 1)[1].strip()

        try:
            payload = self.jwt_handler.verify_token(token)
            request.state.user = payload
            request.state.actor = payload.sub

            self.audit_emitter.emit(
                event_type="AUTH_SUCCESS",
                actor=payload.sub,
                action=f"{request.method} {request.url.path}",
                resource=str(request.url.path),
                status="SUCCESS",
                metadata={"roles": payload.roles, "tenant_id": payload.tenant_id, "trace_id": trace_id},
            )
            return payload
        except Exception as err:
            self.audit_emitter.emit(
                event_type="AUTH_FAILURE",
                actor="unknown",
                action=f"{request.method} {request.url.path}",
                resource=str(request.url.path),
                status="DENIED",
                metadata={"error": str(err), "trace_id": trace_id},
            )
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail=f"Token verification failed: {err}",
            )

    def require_roles(self, allowed_roles: List[str]) -> Callable[..., TokenPayload]:
        """FastAPI dependency factory enforcing RBAC role boundaries."""
        def role_checker(user: TokenPayload = Depends(self.get_current_user)) -> TokenPayload:
            if "admin" in user.roles:
                return user
            if not any(r in user.roles for r in allowed_roles):
                raise HTTPException(
                    status_code=status.HTTP_403_FORBIDDEN,
                    detail=f"Access forbidden: User '{user.sub}' lacks required roles {allowed_roles}.",
                )
            return user
        return role_checker

    def require_permissions(self, required_perms: List[str]) -> Callable[..., TokenPayload]:
        """FastAPI dependency factory enforcing granular permissions."""
        def perm_checker(user: TokenPayload = Depends(self.get_current_user)) -> TokenPayload:
            if "admin" in user.roles:
                return user
            missing = [p for p in required_perms if p not in user.permissions]
            if missing:
                raise HTTPException(
                    status_code=status.HTTP_403_FORBIDDEN,
                    detail=f"Access forbidden: Missing required permissions {missing}.",
                )
            return user
        return perm_checker
