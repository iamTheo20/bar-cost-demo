"""Supplier invoices: one data model, one way to read them.

An `Extractor` turns the raw text of an invoice into an `Invoice`. The demo
ships `RuleBasedExtractor` for the plain-text layout in data/invoices/; an
LLM-backed extractor for scanned or emailed invoices plugs in behind the
same protocol, and nothing downstream changes.
"""

import datetime as dt
from dataclasses import dataclass
from decimal import Decimal, InvalidOperation
from pathlib import Path
from typing import Protocol

CENT = Decimal("0.01")


@dataclass(frozen=True)
class LineItem:
    """One invoice row: what was bought, how much, at what price."""

    product: str
    unit: str
    qty: Decimal
    unit_price: Decimal
    total: Decimal


@dataclass(frozen=True)
class Invoice:
    """A supplier invoice, amounts in SEK."""

    supplier: str
    number: str
    date: dt.date
    lines: tuple[LineItem, ...]
    subtotal: Decimal


class Extractor(Protocol):
    """Turns an invoice file into an Invoice, or raises ValueError saying why it cannot."""

    def extract(self, path: Path) -> Invoice: ...


def _amount(raw: str, what: str) -> Decimal:
    try:
        return Decimal(raw.strip().replace(" ", "").replace(",", "."))
    except InvalidOperation:
        raise ValueError(f"{what} is not a number: {raw!r}") from None


def _headers(lines: list[str]) -> dict[str, str]:
    found: dict[str, str] = {}
    for line in lines:
        key, sep, value = line.partition(":")
        if sep and key.strip().isupper() and "|" not in line:
            found[key.strip()] = value.strip()
    return found


def _rows(lines: list[str]) -> tuple[LineItem, ...]:
    items: list[LineItem] = []
    for line in lines:
        cells = [c.strip() for c in line.split("|")]
        if len(cells) != 5 or cells[0].upper() == "ITEM":
            continue
        product, unit, qty, price, total = cells
        items.append(
            LineItem(
                product,
                unit,
                _amount(qty, f"{product} qty"),
                _amount(price, f"{product} unit price"),
                _amount(total, f"{product} total"),
            )
        )
    return tuple(items)


class RuleBasedExtractor:
    """Reads the demo's plain-text layout: headers, a pipe table, a subtotal."""

    def extract(self, path: Path) -> Invoice:
        """Read and parse one plain-text invoice file."""
        return self.parse(path.read_text(encoding="utf-8"))

    def parse(self, text: str) -> Invoice:
        """Parse one invoice, refusing anything with a header or the table missing."""
        lines = [line for line in text.splitlines() if line.strip()]
        head = _headers(lines)
        missing = [k for k in ("SUPPLIER", "INVOICE", "DATE", "SUBTOTAL") if k not in head]
        if missing:
            raise ValueError(f"invoice is missing {', '.join(missing)}")
        items = _rows(lines)
        if not items:
            raise ValueError("invoice has no line items")
        return Invoice(
            head["SUPPLIER"],
            head["INVOICE"],
            dt.date.fromisoformat(head["DATE"]),
            items,
            _amount(head["SUBTOTAL"], "subtotal"),
        )


def load_invoice(path: Path, extractor: Extractor | None = None) -> Invoice:
    """The invoice at `path`, read by `extractor` (the rule-based one by default)."""
    return (extractor or RuleBasedExtractor()).extract(path)


def arithmetic_issues(invoice: Invoice) -> list[str]:
    """Every place the supplier's own numbers do not add up."""
    issues: list[str] = []
    for item in invoice.lines:
        expected = (item.qty * item.unit_price).quantize(CENT)
        if expected != item.total:
            sum_text = f"{item.qty} x {item.unit_price} = {expected}"
            issues.append(f"{item.product}: {sum_text}, invoiced {item.total}")
    lines_sum = sum((item.total for item in invoice.lines), Decimal(0))
    if lines_sum != invoice.subtotal:
        issues.append(f"subtotal: lines add up to {lines_sum}, invoiced {invoice.subtotal}")
    return issues
