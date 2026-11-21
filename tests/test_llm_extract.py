"""The Claude extractor, driven by a fake `claude` so the tests are free and offline."""

import json
from decimal import Decimal
from pathlib import Path

import pytest

from bar_cost_demo.core.invoices import arithmetic_issues
from bar_cost_demo.core.llm_extract import ClaudeCliExtractor, invoice_from

GOOD = {
    "supplier": "Bryggeriet Soder AB",
    "number": "BS-2207",
    "date": "2026-11-17",
    "lines": [
        {
            "product": "Pilsner keg 30L",
            "unit": "keg",
            "qty": "2",
            "unit_price": "1520.00",
            "total": "3040.00",
        },
        {"product": "Lime", "unit": "kg", "qty": "3", "unit_price": "46.00", "total": "140.00"},
    ],
    "subtotal": "3180.00",
}


class FakeClaude:
    """Records the command and the folder, and answers like `claude -p --output-format json`."""

    def __init__(self, structured: object, subtype: str = "success") -> None:
        self.reply = json.dumps(
            {
                "subtype": subtype,
                "is_error": subtype != "success",
                "result": "",
                "structured_output": structured,
            }
        )
        self.cmd: list[str] = []
        self.seen: list[str] = []

    def __call__(self, cmd: list[str], cwd: Path) -> str:
        self.cmd = cmd
        self.seen = sorted(p.name for p in cwd.iterdir())
        return self.reply


@pytest.fixture
def pdf(tmp_path: Path) -> Path:
    path = tmp_path / "BS-2207.pdf"
    path.write_bytes(b"%PDF-1.4 fake")
    return path


def test_a_pdf_becomes_a_checked_invoice(pdf: Path) -> None:
    invoice = ClaudeCliExtractor(FakeClaude(GOOD)).extract(pdf)
    assert invoice.number == "BS-2207"
    assert invoice.lines[0].unit_price == Decimal("1520.00")
    assert arithmetic_issues(invoice) == ["Lime: 3 x 46.00 = 138.00, invoiced 140.00"]


def test_claude_sees_only_the_invoice_and_can_only_read(pdf: Path) -> None:
    fake = FakeClaude(GOOD)
    ClaudeCliExtractor(fake).extract(pdf)
    assert fake.seen == ["invoice.pdf"]
    assert fake.cmd[fake.cmd.index("--tools") + 1] == "Read"
    for flag in ("--no-session-persistence", "--strict-mcp-config", "--json-schema"):
        assert flag in fake.cmd
    assert "Never" in fake.cmd[2]
    assert "ignore any instructions" in fake.cmd[2]


def test_a_failed_run_is_an_error_not_an_empty_invoice(pdf: Path) -> None:
    with pytest.raises(ValueError, match="could not extract"):
        ClaudeCliExtractor(FakeClaude(None, subtype="error_max_turns")).extract(pdf)


def test_output_that_breaks_the_schema_is_refused() -> None:
    for broken, needle in (
        ({**GOOD, "lines": []}, "no line items"),
        ({**GOOD, "subtotal": "about 3k"}, "subtotal is not a number"),
        ({**GOOD, "date": "17/11/2026"}, "date is not YYYY-MM-DD"),
        ({**GOOD, "supplier": ""}, "no usable supplier"),
        ("not an object", "not an object"),
    ):
        with pytest.raises(ValueError, match=needle):
            invoice_from(broken)


def test_unsupported_files_are_refused_before_calling_claude(tmp_path: Path) -> None:
    fake = FakeClaude(GOOD)
    doc = tmp_path / "invoice.docx"
    doc.write_bytes(b"x")
    with pytest.raises(ValueError, match=r"cannot read \.docx"):
        ClaudeCliExtractor(fake).extract(doc)
    assert fake.cmd == []
