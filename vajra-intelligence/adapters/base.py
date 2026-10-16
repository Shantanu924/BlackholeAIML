"""
adapters/base.py — Base class and status enum for all VAJRA adapters.

Every adapter follows the same contract:
  - .status  → AdapterStatus (LIVE / SIMULATED / ERROR / BLOCKED)
  - .call()  → raises NotImplementedError if not overridden

Adapters flip from SIMULATED to LIVE automatically when
config.settings.vajra_live is True (i.e. VAJRA_INTERNAL_URL is set in .env).
"""
from __future__ import annotations

from abc import ABC, abstractmethod
from typing import Any

from intelligence.models import AdapterStatus


class BaseVajraAdapter(ABC):
    """Abstract base for all VAJRA integration adapters."""

    @property
    @abstractmethod
    def name(self) -> str:
        """Human-readable adapter name."""

    @property
    @abstractmethod
    def endpoint(self) -> str:
        """Target endpoint path (relative to VAJRA_INTERNAL_URL)."""

    @property
    def status(self) -> AdapterStatus:
        """
        Return LIVE if VAJRA_INTERNAL_URL is configured, else SIMULATED.
        Subclasses may override to return ERROR after a failed call.
        """
        from config import settings
        return AdapterStatus.LIVE if settings.vajra_live else AdapterStatus.SIMULATED

    @abstractmethod
    def call(self, **kwargs: Any) -> dict[str, Any]:
        """
        Execute the adapter call.

        If SIMULATED, returns a clearly-labelled synthetic response.
        If LIVE, calls the real VAJRA endpoint and returns its response.
        Raises on non-2xx responses.
        """

    def health(self) -> dict[str, Any]:
        """Return a health snapshot for the integration health matrix."""
        return {
            "name": self.name,
            "status": self.status.value,
            "endpoint": self.endpoint,
        }
