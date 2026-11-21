"""The one way to log: `log = get_logger(__name__)`."""

import logging
from functools import cache

from bar_cost_demo.core.config import load


@cache
def _configure() -> None:
    logging.basicConfig(level=load().log_level, format="%(levelname)s %(name)s: %(message)s")


def get_logger(name: str) -> logging.Logger:
    """A logger configured once from the project settings."""
    _configure()
    return logging.getLogger(name)
