"""The arguments every invoice feature takes: `<invoice file> [--summary] [--llm]`."""

from dataclasses import dataclass
from pathlib import Path

from bar_cost_demo.core.config import load
from bar_cost_demo.core.invoices import Extractor, Invoice, RuleBasedExtractor, load_invoice
from bar_cost_demo.core.llm_extract import ClaudeCliExtractor

FLAGS = ("--summary", "--llm")


@dataclass(frozen=True)
class InvoiceArgs:
    """Which invoice to read and how, where the venue's data lives, and how much to say."""

    invoice: Path
    data_dir: Path
    summary_only: bool
    use_llm: bool

    def extractor(self) -> Extractor:
        """Claude for PDFs, photos or --llm; the rule-based reader for the demo's text layout."""
        if self.use_llm or self.invoice.suffix.lower() != ".txt":
            return ClaudeCliExtractor()
        return RuleBasedExtractor()

    def load(self) -> Invoice:
        """The invoice, read by the right extractor."""
        return load_invoice(self.invoice, self.extractor())


def invoice_args(args: list[str]) -> InvoiceArgs:
    """Parse `<invoice file> [--summary] [--llm]`; ValueError says what is wrong."""
    rest = [a for a in args if a not in FLAGS]
    if len(rest) != 1:
        raise ValueError("usage: <feature> <invoice file> [--summary] [--llm]")
    return InvoiceArgs(Path(rest[0]), load().data_dir, "--summary" in args, "--llm" in args)
