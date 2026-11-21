"""What the venue knows besides invoices: past prices, recipes and the menu."""

import csv
import datetime as dt
from dataclasses import dataclass
from decimal import Decimal
from pathlib import Path

from bar_cost_demo.core.invoices import Invoice


@dataclass(frozen=True)
class PricePoint:
    """One past purchase price for a product."""

    date: dt.date
    supplier: str
    unit_price: Decimal


@dataclass(frozen=True)
class MenuItem:
    """A dish or drink: its price excluding VAT and the margin it should keep."""

    name: str
    price: Decimal
    target_margin: Decimal
    recipe: tuple[tuple[str, Decimal], ...]


def _rows(path: Path) -> list[dict[str, str]]:
    with path.open(encoding="utf-8", newline="") as f:
        return list(csv.DictReader(f))


def price_history(path: Path) -> dict[str, list[PricePoint]]:
    """Each product's past prices, oldest first."""
    history: dict[str, list[PricePoint]] = {}
    for row in _rows(path):
        point = PricePoint(
            dt.date.fromisoformat(row["date"]), row["supplier"], Decimal(row["unit_price"])
        )
        history.setdefault(row["product"], []).append(point)
    for points in history.values():
        points.sort(key=lambda p: p.date)
    return history


def last_price_before(
    history: dict[str, list[PricePoint]], product: str, day: dt.date
) -> PricePoint | None:
    """The most recent purchase of `product` strictly before `day`, if there was one."""
    earlier = [p for p in history.get(product, []) if p.date < day]
    return earlier[-1] if earlier else None


def menu(menu_path: Path, recipes_path: Path) -> list[MenuItem]:
    """The menu, each item with the ingredients one serving uses."""
    recipes: dict[str, list[tuple[str, Decimal]]] = {}
    for row in _rows(recipes_path):
        recipes.setdefault(row["menu_item"], []).append((row["product"], Decimal(row["qty"])))
    return [
        MenuItem(
            row["menu_item"],
            Decimal(row["price"]),
            Decimal(row["target_margin"]),
            tuple(recipes.get(row["menu_item"], [])),
        )
        for row in _rows(menu_path)
    ]


def current_prices(
    history: dict[str, list[PricePoint]], invoice: Invoice | None = None
) -> dict[str, Decimal]:
    """The latest known price per product; an invoice's prices replace the history's."""
    prices = {product: points[-1].unit_price for product, points in history.items() if points}
    if invoice is not None:
        prices.update({item.product: item.unit_price for item in invoice.lines})
    return prices


def serving_cost(item: MenuItem, prices: dict[str, Decimal]) -> Decimal:
    """What one serving costs in ingredients; KeyError names a product with no known price."""
    return sum((qty * prices[product] for product, qty in item.recipe), Decimal(0))
