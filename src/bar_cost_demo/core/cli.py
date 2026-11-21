"""The arguments every invoice feature takes: `<invoice file> [--summary]`."""

from dataclasses import dataclass
from pathlib import Path

from bar_cost_demo.core.config import load


@dataclass(frozen=True)
class InvoiceArgs:
    """Which invoice to read, where the venue's data lives, and how much to say."""

    invoice: Path
    data_dir: Path
    summary_only: bool


def invoice_args(args: list[str]) -> InvoiceArgs:
    """Parse `<invoice file> [--summary]`; ValueError says what is wrong."""
    rest = [a for a in args if a != "--summary"]
    if len(rest) != 1:
        raise ValueError("usage: <feature> <invoice file> [--summary]")
    return InvoiceArgs(Path(rest[0]), load().data_dir, "--summary" in args)
