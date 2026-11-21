"""The one way to read settings: `settings = load()`."""

import os
from dataclasses import dataclass

LEVELS = ("DEBUG", "INFO", "WARNING", "ERROR")


@dataclass(frozen=True)
class Settings:
    """Validated settings, read from the environment once."""

    log_level: str = "WARNING"


def load() -> Settings:
    """Settings from `BAR_COST_DEMO_*` environment variables, validated."""
    name = "BAR_COST_DEMO_LOG_LEVEL"
    level = os.environ.get(name, "WARNING").upper()
    if level not in LEVELS:
        allowed = ", ".join(LEVELS)
        raise ValueError(f"{name} must be one of {allowed}, not {level!r}")
    return Settings(log_level=level)
