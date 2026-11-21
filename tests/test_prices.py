"""Flagging price moves against the last purchase."""

import datetime as dt
from decimal import Decimal
from pathlib import Path

from bar_cost_demo.core.catalog import last_price_before, price_history
from bar_cost_demo.features.prices import feature

DATA = Path("data")


def test_the_last_purchase_is_the_latest_before_the_invoice_date() -> None:
    history = price_history(DATA / "price_history.csv")
    last = last_price_before(history, "Avocado", dt.date(2026, 11, 4))
    assert last is not None
    assert last.unit_price == Decimal("64.00")
    assert last_price_before(history, "Avocado", dt.date(2026, 9, 1)) is None


def test_moves_of_five_percent_or_more_are_flagged() -> None:
    report = feature.run([str(DATA / "invoices" / "NB-48211.txt")])
    assert "Fever-Tree Tonic 20cl" in report
    flagged = [line for line in report.splitlines() if line.endswith("FLAG")]
    assert len(flagged) == 2


def test_the_summary_names_the_largest_move() -> None:
    summary = feature.run([str(DATA / "invoices" / "SF-1093.txt"), "--summary"])
    assert summary == "prices SF-1093: 2 of 4 items moved 5% or more (largest: Avocado +23.4%)"
