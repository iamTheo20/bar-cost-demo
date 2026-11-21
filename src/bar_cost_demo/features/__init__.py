"""Feature discovery: every folder here with a `feature.py` is a feature.

A feature module defines `NAME: str` and `run(args: list[str]) -> str`.
Add one with `make new-feature NAME=<name>`.
"""

import importlib
import pkgutil
from types import ModuleType


def discover() -> dict[str, ModuleType]:
    """Map each feature's NAME to its module, found by convention."""
    found: dict[str, ModuleType] = {}
    for info in pkgutil.iter_modules(__path__):
        if info.ispkg:
            module = importlib.import_module(f"{__name__}.{info.name}.feature")
            found[module.NAME] = module
    return found
