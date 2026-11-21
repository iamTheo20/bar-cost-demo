"""The one way to read settings: `settings = load()`."""

import os
from dataclasses import dataclass, field
from pathlib import Path

LEVELS = ("DEBUG", "INFO", "WARNING", "ERROR")


@dataclass(frozen=True)
class Settings:
    """Validated settings, read from the environment once."""

    log_level: str = "WARNING"
    data_dir: Path = field(default_factory=lambda: Path("data"))
    claude_model: str = ""


def load() -> Settings:
    """Settings from `BAR_COST_DEMO_*` environment variables, validated."""
    name = "BAR_COST_DEMO_LOG_LEVEL"
    level = os.environ.get(name, "WARNING").upper()
    if level not in LEVELS:
        allowed = ", ".join(LEVELS)
        raise ValueError(f"{name} must be one of {allowed}, not {level!r}")
    data_dir = Path(os.environ.get("BAR_COST_DEMO_DATA_DIR", "data"))
    claude_model = os.environ.get("BAR_COST_DEMO_CLAUDE_MODEL", "")
    return Settings(log_level=level, data_dir=data_dir, claude_model=claude_model)
