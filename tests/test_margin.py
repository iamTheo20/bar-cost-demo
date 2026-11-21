"""What an invoice's prices do to menu margins."""

from decimal import Decimal
from pathlib import Path

import pytest

from bar_cost_demo.core.catalog import current_prices, menu, price_history, serving_cost
from bar_cost_demo.core.invoices import load_invoice
from bar_cost_demo.features.margin import feature

DATA = Path("data")


def gin_and_tonic_cost(with_invoice: bool) -> Decimal:
    history = price_history(DATA / "price_history.csv")
    invoice = load_invoice(DATA / "invoices" / "NB-48211.txt") if with_invoice else None
    item = next(m for m in menu(DATA / "menu.csv", DATA / "recipes.csv") if m.name == "Gin & Tonic")
    return serving_cost(item, current_prices(history, invoice))


def test_the_invoice_raises_the_serving_cost() -> None:
    assert gin_and_tonic_cost(with_invoice=False) == Decimal("32.187900")
    assert gin_and_tonic_cost(with_invoice=True) == Decimal("33.667900")


def test_an_ingredient_with_no_price_is_an_error_not_a_zero() -> None:
    items = menu(DATA / "menu.csv", DATA / "recipes.csv")
    item = next(m for m in items if m.name == "Cheeseburger")
    with pytest.raises(KeyError, match="Cheddar"):
        serving_cost(item, {"Beef mince": Decimal(120), "Burger bun": Decimal("6.20")})


def test_items_pushed_below_target_are_named() -> None:
    summary = feature.run([str(DATA / "invoices" / "SF-1093.txt"), "--summary"])
    assert summary == (
        "margin SF-1093: 1 of 2 affected menu items below target (Guacamole & chips 77.8% < 80%)"
    )


def test_only_items_using_an_invoiced_product_are_shown() -> None:
    report = feature.run([str(DATA / "invoices" / "NB-48211.txt")])
    assert "Gin & Tonic" in report
    assert "Cheeseburger" not in report
