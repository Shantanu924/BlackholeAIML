"""
TANTRA Integrations Package
Provides external and local client integrations, including resilient Local LLM clients.
"""

from .local_llm_client import LocalLLMClient, LLMReviewSummary, get_default_llm_client

__all__ = [
    "LocalLLMClient",
    "LLMReviewSummary",
    "get_default_llm_client",
]
