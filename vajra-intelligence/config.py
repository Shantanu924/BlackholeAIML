"""
config.py — VAJRA Intelligence Layer configuration.

All values are read from the environment / .env file via pydantic-settings.
No secrets or hardcoded endpoints in this file.
"""
from __future__ import annotations

from pathlib import Path
from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict

BASE_DIR = Path(__file__).parent


class Settings(BaseSettings):
    """
    All config is sourced from environment variables or the .env file.
    Defaults to SIMULATED mode when VAJRA_INTERNAL_URL is blank.
    """

    model_config = SettingsConfigDict(
        env_file=str(BASE_DIR / ".env"),
        env_file_encoding="utf-8",
        extra="ignore",
    )

    # ── VAJRA connectivity ─────────────────────────────────────────────────
    VAJRA_BASE_URL: str = Field(
        default="http://vajra.blackholeinfiverse.com",
        description="Public VAJRA host (read-only, for reference).",
    )
    VAJRA_INTERNAL_URL: str = Field(
        default="",
        description=(
            "Internal VAJRA API base URL. "
            "Leave blank → all adapters run in SIMULATED mode."
        ),
    )
    VAJRA_API_KEY: str = Field(
        default="",
        description="API key for VAJRA authenticated endpoints.",
    )
    VAJRA_TIMEOUT_SECONDS: int = Field(
        default=5,
        description="HTTP timeout (seconds) for outbound VAJRA calls.",
    )

    # ── Intelligence service ───────────────────────────────────────────────
    SERVICE_NAME: str = "vajra-intelligence"
    SERVICE_VERSION: str = "1.0.0"
    LOG_LEVEL: str = "INFO"
    HOST: str = "0.0.0.0"
    PORT: int = 8100

    # ── Evidence store ─────────────────────────────────────────────────────
    EVIDENCE_DB_PATH: str = str(BASE_DIR / "data" / "intelligence_evidence.db")

    # ── Risk thresholds ────────────────────────────────────────────────────
    RISK_SCORE_MEDIUM: int = 2
    RISK_SCORE_HIGH: int = 4
    RISK_SCORE_CRITICAL: int = 6
    FREQ_ABUSE_THRESHOLD: int = 5
    OFF_HOURS_START: int = 0    # UTC hour (inclusive)
    OFF_HOURS_END: int = 6      # UTC hour (exclusive)

    @property
    def vajra_live(self) -> bool:
        """True only if an internal VAJRA URL has been configured."""
        return bool(self.VAJRA_INTERNAL_URL.strip())


# Singleton — import from here everywhere
settings = Settings()

# ── Derived paths (created at import time) ─────────────────────────────────
EVIDENCE_DIR: Path = BASE_DIR / "data" / "evidence"
RUNTIME_LOGS_DIR: Path = BASE_DIR / "data" / "runtime_logs"
TEST_RESULTS_DIR: Path = BASE_DIR / "test_results"

for _d in [EVIDENCE_DIR, RUNTIME_LOGS_DIR, TEST_RESULTS_DIR]:
    _d.mkdir(parents=True, exist_ok=True)

# ── Domain constants ────────────────────────────────────────────────────────
SENSITIVE_RESOURCES: frozenset[str] = frozenset(
    {
        "personal_data",
        "health_records",
        "financial_data",
        "auth_tokens",
        "private_keys",
        "wallet_keys",
        "validator_keys",
        "governance_keys",
    }
)

HIGH_VALUE_ACTIONS: frozenset[str] = frozenset(
    {"DELETE", "EXPORT", "TRANSFER", "STAKE", "UNSTAKE", "VOTE", "BRIDGE"}
)
