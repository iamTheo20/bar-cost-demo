"""Scaffold a feature: `make new-feature NAME=<name>`.

Writes a feature folder with a feature.py, and a test that passes, so the
shortest path to a new feature is also the one that passes every check.
"""

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
FEATURES = ROOT / "src" / "bar_cost_demo" / "features"

FEATURE = '''"""The {name} feature: `bar-cost-demo {name} [args...]`."""

NAME = "{name}"


def run(args: list[str]) -> str:
    """Describe what {name} returns for its arguments."""
    return " ".join(args)
'''

TEST = '''"""The {name} feature."""

from bar_cost_demo.features.{name} import feature


def test_{name}_echoes_its_arguments() -> None:
    assert feature.run(["a", "b"]) == "a b"
'''


def main(argv: list[str]) -> int:
    """Create the feature folder and its test, refusing to overwrite."""
    if len(argv) != 1 or not re.fullmatch(r"[a-z][a-z0-9_]*", argv[0]):
        sys.stderr.write("usage: make new-feature NAME=<lowercase_name>\n")
        return 2
    name = argv[0]
    folder = FEATURES / name
    if folder.exists():
        sys.stderr.write(f"{folder} exists already\n")
        return 2
    folder.mkdir()
    (folder / "__init__.py").write_text(f'"""The {name} feature."""\n')
    (folder / "feature.py").write_text(FEATURE.format(name=name))
    (ROOT / "tests" / f"test_{name}.py").write_text(TEST.format(name=name))
    sys.stdout.write(
        f"created {folder.relative_to(ROOT)} and tests/test_{name}.py\n"
        f"next: add a `{name}` row to FEATURES.md with its command and exact output,"
        " then make check verify\n"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
