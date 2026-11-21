"""Read a supplier invoice and check its arithmetic: `bar-cost-demo invoice <file> [--summary]`."""

from bar_cost_demo.core.cli import invoice_args
from bar_cost_demo.core.invoices import arithmetic_issues, load_invoice

NAME = "invoice"


def run(args: list[str]) -> str:
    """The invoice's lines, every arithmetic issue, and a one-line summary last."""
    parsed = invoice_args(args)
    invoice = load_invoice(parsed.invoice)
    issues = arithmetic_issues(invoice)
    summary = (
        f"invoice {invoice.number}: {len(invoice.lines)} lines, {invoice.subtotal} SEK,"
        f" {len(issues)} arithmetic issue(s)"
    )
    if parsed.summary_only:
        return summary
    report = [f"{invoice.supplier}, invoice {invoice.number}, {invoice.date}"]
    report += [
        f"  {i.product:<24} {i.qty:>5} {i.unit:<7} x {i.unit_price:>8} = {i.total:>9}"
        for i in invoice.lines
    ]
    report += [f"  ISSUE {issue}" for issue in issues]
    report.append(summary)
    return "\n".join(report)
