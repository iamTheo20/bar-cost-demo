"""Command line entry point: `bar-cost-demo <feature> [args...]`."""

import sys

from bar_cost_demo.core.log import get_logger
from bar_cost_demo.features import discover

log = get_logger(__name__)
PROG = "bar-cost-demo"


def main(argv: list[str] | None = None) -> int:
    """Run one feature by name and write its output to stdout."""
    args = sys.argv[1:] if argv is None else argv
    features = discover()
    if not args or args[0] not in features:
        names = "|".join(sorted(features))
        sys.stderr.write(f"usage: {PROG} <{names}> [args...]\n")
        return 2
    log.debug("running feature %s", args[0])
    sys.stdout.write(features[args[0]].run(args[1:]) + "\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())
