"""
RS256 Cryptographic Token Engine & Enterprise RBAC
Implements RFC 7519 JWT format with RSA-256 asymmetric signatures using the cryptography library.
Guarantees zero unpinned third-party JWT dependencies.
"""

import base64
import json
import os
import time
import uuid
from datetime import datetime, timezone, timedelta
from typing import Any, Dict, List, Optional, Tuple

from cryptography.exceptions import InvalidSignature
from cryptography.hazmat.backends import default_backend
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import padding, rsa
from pydantic import BaseModel, Field


def _b64url_encode(data: bytes) -> str:
    """Encode bytes to base64url string without trailing padding."""
    return base64.urlsafe_b64encode(data).rstrip(b"=").decode("utf-8")


def _b64url_decode(segment: str) -> bytes:
    """Decode base64url string with proper padding restoration."""
    rem = len(segment) % 4
    if rem > 0:
        segment += "=" * (4 - rem)
    return base64.urlsafe_b64decode(segment.encode("utf-8"))


class TokenPayload(BaseModel):
    """Strict model for enterprise token payload with RBAC claims."""
    sub: str = Field(..., description="Subject / User or Service ID")
    roles: List[str] = Field(default_factory=list, description="Assigned enterprise roles (e.g. admin, auditor, validator, operator)")
    permissions: List[str] = Field(default_factory=list, description="Granular permission strings (e.g. governance:evaluate, replay:verify)")
    tenant_id: str = Field(default="tantra-mainnet", description="Constitutional tenant / ecosystem ID")
    candidate_id: Optional[str] = Field(default=None, description="Assigned candidate ID for governance evaluation")
    jti: str = Field(default_factory=lambda: str(uuid.uuid4()), description="Unique JWT ID for replay prevention")
    iss: str = Field(default="tantra.governance.niyantran", description="Issuer authority")
    aud: str = Field(default="vajra.production.runtime", description="Intended audience")
    iat: int = Field(default_factory=lambda: int(time.time()), description="Issued-at timestamp")
    exp: int = Field(..., description="Expiration timestamp")
    nbf: Optional[int] = Field(default=None, description="Not-before timestamp")

    model_config = {"extra": "forbid"}


class RS256SecurityEngine:
    """
    Manages RSA key pairs and provides high-performance, asymmetric
    RS256 signing and verification for enterprise governance tokens.
    """

    def __init__(
        self,
        private_key_pem: Optional[str] = None,
        public_key_pem: Optional[str] = None,
        key_size: int = 2048,
    ) -> None:
        if private_key_pem:
            self._private_key = serialization.load_pem_private_key(
                private_key_pem.encode("utf-8"),
                password=None,
                backend=default_backend(),
            )
            self._public_key = self._private_key.public_key()
        elif public_key_pem:
            self._private_key = None
            self._public_key = serialization.load_pem_public_key(
                public_key_pem.encode("utf-8"),
                backend=default_backend(),
            )
        else:
            # Generate new RSA key pair
            self._private_key = rsa.generate_private_key(
                public_exponent=65537,
                key_size=key_size,
                backend=default_backend(),
            )
            self._public_key = self._private_key.public_key()

    def get_public_key_pem(self) -> str:
        """Export RSA public key in PKCS#8 PEM format."""
        return self._public_key.public_bytes(
            encoding=serialization.Encoding.PEM,
            format=serialization.PublicFormat.SubjectPublicKeyInfo,
        ).decode("utf-8")

    def get_private_key_pem(self) -> str:
        """Export RSA private key in PKCS#8 PEM format (if available)."""
        if not self._private_key:
            raise ValueError("Private key is not available on this engine instance.")
        return self._private_key.private_bytes(
            encoding=serialization.Encoding.PEM,
            format=serialization.PrivateFormat.PKCS8,
            encryption_algorithm=serialization.NoEncryption(),
        ).decode("utf-8")

    def get_jwks(self) -> Dict[str, Any]:
        """Generate JSON Web Key Set (JWKS) representation."""
        public_numbers = self._public_key.public_numbers()
        e_bytes = public_numbers.e.to_bytes((public_numbers.e.bit_length() + 7) // 8, byteorder="big")
        n_bytes = public_numbers.n.to_bytes((public_numbers.n.bit_length() + 7) // 8, byteorder="big")
        return {
            "keys": [
                {
                    "kty": "RSA",
                    "use": "sig",
                    "alg": "RS256",
                    "kid": "tantra-primary-v1",
                    "n": _b64url_encode(n_bytes),
                    "e": _b64url_encode(e_bytes),
                }
            ]
        }

    def sign_jwt(self, claims: Dict[str, Any], expires_delta: Optional[timedelta] = None) -> str:
        """
        Produce an RS256-signed JWT token from a claims dictionary.
        """
        if not self._private_key:
            raise ValueError("Cannot sign JWT without private key.")

        now = int(time.time())
        if "iat" not in claims:
            claims["iat"] = now
        if "exp" not in claims:
            ttl = int(expires_delta.total_seconds()) if expires_delta else 3600
            claims["exp"] = now + ttl
        if "jti" not in claims:
            claims["jti"] = str(uuid.uuid4())
        if "iss" not in claims:
            claims["iss"] = "tantra.governance.niyantran"
        if "aud" not in claims:
            claims["aud"] = "vajra.production.runtime"
        if "tenant_id" not in claims:
            claims["tenant_id"] = "tantra-mainnet"

        header = {"alg": "RS256", "typ": "JWT", "kid": "tantra-primary-v1"}
        header_b64 = _b64url_encode(json.dumps(header, separators=(",", ":")).encode("utf-8"))
        payload_b64 = _b64url_encode(json.dumps(claims, separators=(",", ":")).encode("utf-8"))

        signing_input = f"{header_b64}.{payload_b64}".encode("ascii")
        signature = self._private_key.sign(
            signing_input,
            padding.PKCS1v15(),
            hashes.SHA256(),
        )
        sig_b64 = _b64url_encode(signature)
        return f"{header_b64}.{payload_b64}.{sig_b64}"

    def verify_jwt(self, token: str, expected_audience: Optional[str] = None) -> Dict[str, Any]:
        """
        Verify RS256 JWT signature, format, expiration, and claims.
        Raises ValueError with specific message on failure.
        """
        parts = token.strip().split(".")
        if len(parts) != 3:
            raise ValueError("Malformed token: JWT must consist of three dot-separated segments.")

        header_b64, payload_b64, sig_b64 = parts

        try:
            header_bytes = _b64url_decode(header_b64)
            header = json.loads(header_bytes.decode("utf-8"))
        except Exception as err:
            raise ValueError(f"Malformed token header: {err}")

        if header.get("alg") != "RS256":
            raise ValueError(f"Unsupported algorithm '{header.get('alg')}'. Only RS256 is permitted.")

        try:
            payload_bytes = _b64url_decode(payload_b64)
            payload = json.loads(payload_bytes.decode("utf-8"))
        except Exception as err:
            raise ValueError(f"Malformed token payload: {err}")

        try:
            signature = _b64url_decode(sig_b64)
        except Exception as err:
            raise ValueError(f"Invalid base64url signature encoding: {err}")

        signing_input = f"{header_b64}.{payload_b64}".encode("ascii")

        try:
            self._public_key.verify(
                signature,
                signing_input,
                padding.PKCS1v15(),
                hashes.SHA256(),
            )
        except InvalidSignature:
            raise ValueError("Cryptographic verification failed: Invalid RS256 signature.")
        except Exception as err:
            raise ValueError(f"Verification error: {err}")

        # Expiration & Time checks
        now = int(time.time())
        exp = payload.get("exp")
        if exp is not None and now > exp:
            raise ValueError(f"Token has expired at timestamp {exp} (current time: {now}).")

        nbf = payload.get("nbf")
        if nbf is not None and now < nbf:
            raise ValueError(f"Token not valid before timestamp {nbf} (current time: {now}).")

        # Audience check
        if expected_audience:
            token_aud = payload.get("aud")
            if token_aud != expected_audience:
                raise ValueError(f"Audience mismatch: expected '{expected_audience}', got '{token_aud}'.")

        return payload


# Singleton instance for runtime convenience
_default_engine: Optional[RS256SecurityEngine] = None


def get_default_security_engine() -> RS256SecurityEngine:
    """Retrieve or initialize the system default RS256 security engine."""
    global _default_engine
    if _default_engine is None:
        priv_path = os.environ.get("JWT_PRIVATE_KEY_PATH")
        pub_path = os.environ.get("JWT_PUBLIC_KEY_PATH")
        priv_pem = None
        pub_pem = None
        if priv_path and os.path.exists(priv_path):
            with open(priv_path, "r", encoding="utf-8") as f:
                priv_pem = f.read()
        if pub_path and os.path.exists(pub_path):
            with open(pub_path, "r", encoding="utf-8") as f:
                pub_pem = f.read()

        _default_engine = RS256SecurityEngine(private_key_pem=priv_pem, public_key_pem=pub_pem)
    return _default_engine


class JWTHandler:
    """Enterprise wrapper for high-level token operations and RBAC checking."""

    def __init__(self, engine: Optional[RS256SecurityEngine] = None) -> None:
        self.engine = engine or get_default_security_engine()

    def issue_token(
        self,
        sub: str,
        roles: Optional[List[str]] = None,
        permissions: Optional[List[str]] = None,
        tenant_id: str = "tantra-mainnet",
        candidate_id: Optional[str] = "Prakash Yadav",
        ttl_seconds: int = 3600,
    ) -> str:
        """Issue an RS256 JWT for a given identity and role set."""
        now = int(time.time())
        claims = {
            "sub": sub,
            "roles": roles or ["operator"],
            "permissions": permissions or [],
            "tenant_id": tenant_id,
            "candidate_id": candidate_id,
            "jti": str(uuid.uuid4()),
            "iss": "tantra.governance.niyantran",
            "aud": "vajra.production.runtime",
            "iat": now,
            "exp": now + ttl_seconds,
        }
        return self.engine.sign_jwt(claims)

    def verify_token(self, token: str) -> TokenPayload:
        """Verify token and return strictly validated TokenPayload."""
        claims = self.engine.verify_jwt(token)
        return TokenPayload(**claims)

    def check_permissions(
        self,
        payload: TokenPayload,
        required_roles: Optional[List[str]] = None,
        required_permissions: Optional[List[str]] = None,
    ) -> Tuple[bool, str]:
        """
        Verify that token payload contains either any of required_roles
        or all of required_permissions.
        """
        if "admin" in payload.roles:
            return True, "Authorized via admin role override."

        if required_roles:
            has_role = any(r in payload.roles for r in required_roles)
            if not has_role:
                return False, f"Missing required role. Requires one of: {required_roles}."

        if required_permissions:
            missing = [p for p in required_permissions if p not in payload.permissions]
            if missing:
                return False, f"Missing required permissions: {missing}."

        return True, "Authorized."
