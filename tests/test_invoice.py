"""Reading supplier invoices and checking their arithmetic."""

from decimal import Decimal
from pathlib import Path

import pytest

from bar_cost_demo.core.invoices import RuleBasedExtractor, arithmetic_issues, load_invoice
from bar_cost_demo.features.invoice import feature

DATA = Path("data/invoices")


def test_an_invoice_is_read_into_typed_lines() -> None:
    invoice = load_invoice(DATA / "NB-48211.txt")
    assert invoice.supplier == "Nordic Beverage AB"
    assert len(invoice.lines) == 4
    assert invoice.lines[1].unit_price == Decimal("12.90")
    assert invoice.subtotal == Decimal("5823.20")


def test_a_line_that_does_not_add_up_is_reported() -> None:
    issues = arithmetic_issues(load_invoice(DATA / "SF-1093.txt"))
    assert issues == ["Cheddar: 2 x 145.00 = 290.00, invoiced 300.00"]


def test_a_correct_invoice_has_no_issues() -> None:
    assert arithmetic_issues(load_invoice(DATA / "NB-48211.txt")) == []


def test_an_invoice_without_headers_is_refused() -> None:
    text = "ITEM | UNIT | QTY | UNIT PRICE | TOTAL\nLime | kg | 1 | 2 | 2\n"
    with pytest.raises(ValueError, match="missing SUPPLIER"):
        RuleBasedExtractor().parse(text)


def test_a_non_number_is_refused_with_its_product() -> None:
    text = "SUPPLIER: A\nINVOICE: 1\nDATE: 2026-11-01\nLime | kg | two | 2.00 | 4.00\nSUBTOTAL: 4\n"
    with pytest.raises(ValueError, match="Lime qty"):
        RuleBasedExtractor().parse(text)


def test_the_summary_counts_lines_total_and_issues() -> None:
    summary = feature.run([str(DATA / "SF-1093.txt"), "--summary"])
    assert summary == "invoice SF-1093: 4 lines, 2308.00 SEK, 1 arithmetic issue(s)"
