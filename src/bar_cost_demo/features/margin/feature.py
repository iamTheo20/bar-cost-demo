"""Show what an invoice's prices do to menu margins.

    bar-cost-demo margin <file> [--summary] [--llm]
"""

from decimal import Decimal

from bar_cost_demo.core.catalog import current_prices, menu, price_history, serving_cost
from bar_cost_demo.core.cli import invoice_args

NAME = "margin"


def run(args: list[str]) -> str:
    """Margin before and after for every menu item the invoice touches, and a summary last."""
    parsed = invoice_args(args)
    invoice = parsed.load()
    history = price_history(parsed.data_dir / "price_history.csv")
    items = menu(parsed.data_dir / "menu.csv", parsed.data_dir / "recipes.csv")
    bought = {item.product for item in invoice.lines}
    before, after = current_prices(history), current_prices(history, invoice)
    affected = [m for m in items if any(product in bought for product, _ in m.recipe)]
    report = [f"Menu margins after {invoice.number} (price ex VAT, target in brackets):"]
    below: list[tuple[str, Decimal, Decimal]] = []
    for item in affected:
        old = 1 - serving_cost(item, before) / item.price
        new = 1 - serving_cost(item, after) / item.price
        flag = new < item.target_margin
        if flag:
            below.append((item.name, new, item.target_margin))
        report.append(
            f"  {item.name:<20} {old:6.1%} -> {new:6.1%}  ({item.target_margin:.0%})"
            + ("  BELOW TARGET" if flag else "")
        )
    summary = (
        f"margin {invoice.number}: {len(below)} of {len(affected)} affected menu items below target"
    )
    if below:
        summary += " (" + ", ".join(f"{n} {m:.1%} < {t:.0%}" for n, m, t in below) + ")"
    return summary if parsed.summary_only else "\n".join([*report, summary])
