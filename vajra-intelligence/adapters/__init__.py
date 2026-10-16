"""adapters/__init__.py — exports all adapter singletons."""
from adapters.vajra_health import vajra_health_adapter
from adapters.vajra_audit import vajra_audit_adapter
from adapters.vajra_replay import vajra_replay_adapter
from adapters.vajra_events import vajra_events_adapter

ALL_ADAPTERS = [
    vajra_health_adapter,
    vajra_audit_adapter,
    vajra_replay_adapter,
    vajra_events_adapter,
]

__all__ = [
    "vajra_health_adapter",
    "vajra_audit_adapter",
    "vajra_replay_adapter",
    "vajra_events_adapter",
    "ALL_ADAPTERS",
]
