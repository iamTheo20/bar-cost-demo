"""Flag price moves against the last purchase: `bar-cost-demo prices <file> [--summary] [--llm]`."""

from decimal import Decimal

from bar_cost_demo.core.catalog import last_price_before, price_history
from bar_cost_demo.core.cli import invoice_args

NAME = "prices"
THRESHOLD = Decimal("0.05")


def run(args: list[str]) -> str:
    """Each line's change since the last purchase, flagged at 5% or more, and a summary last."""
    parsed = invoice_args(args)
    invoice = parsed.load()
    history = price_history(parsed.data_dir / "price_history.csv")
    report = [f"Price changes on {invoice.number} vs the last purchase:"]
    moved: list[tuple[str, Decimal]] = []
    for item in invoice.lines:
        last = last_price_before(history, item.product, invoice.date)
        if last is None:
            report.append(f"  {item.product:<24} {'new':>8} -> {item.unit_price:>8}   no history")
            continue
        change = (item.unit_price - last.unit_price) / last.unit_price
        flag = abs(change) >= THRESHOLD
        if flag:
            moved.append((item.product, change))
        mark = "  FLAG" if flag else ""
        prices = f"{last.unit_price:>8} -> {item.unit_price:>8}"
        report.append(f"  {item.product:<24} {prices}  {change:+7.1%}{mark}")
    summary = (
        f"prices {invoice.number}: {len(moved)} of {len(invoice.lines)} items moved 5% or more"
    )
    if moved:
        name, change = max(moved, key=lambda m: abs(m[1]))
        summary += f" (largest: {name} {change:+.1%})"
    return summary if parsed.summary_only else "\n".join([*report, summary])
