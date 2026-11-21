# bar-cost-demo

A demo of automating supplier invoices and price swings for a bar or
restaurant: read each invoice, catch the supplier's mistakes, flag price
jumps against what you paid last time, and show which dishes and drinks a
price change pushes below their target margin.

```sh
uv sync
uv run bar-cost-demo invoice data/invoices/SF-1093.txt   # catches 2 x 145.00 invoiced as 300.00
uv run bar-cost-demo prices  data/invoices/SF-1093.txt   # avocado +23.4%, beef +10.0%
uv run bar-cost-demo margin  data/invoices/NB-48211.txt  # Gin & Tonic drops to 75.2%, target 76%
make check verify
```

Add `--summary` to any command for its one-line result.

### Real invoices: PDFs and photos, read by Claude

```sh
uv run bar-cost-demo invoice data/invoices/BS-2207.pdf   # a PDF with Swedish number formats and VAT
uv run bar-cost-demo prices  data/invoices/BS-2207.pdf   # lime +21.1%, gin +6.6%
uv run bar-cost-demo invoice data/invoices/SF-1093.txt --llm   # force Claude on a text invoice too
```

Any `.pdf`, `.png`, `.jpg`, `.jpeg` or `.webp` invoice (or a text one with
`--llm`) is read by `core/llm_extract.py`, which calls the local Claude Code
CLI (`claude -p`) on your own login, so no API key is needed. Each call takes
about 8 seconds. What it does:

- Claude sees the invoice in an empty temporary folder, has only the Read tool,
  saves no session and loads no MCP servers.
- Its answer must match a JSON schema: amounts as decimal strings, ISO date,
  one object per line.
- It is told to copy numbers as printed and never correct one, so the
  arithmetic check still catches supplier mistakes. On `BS-2207.pdf` it keeps
  the wrong 140,00 for 3 kg of limes at 46,00, and `invoice` flags it.
- It is told to ignore instructions written inside the invoice.
- Every field is validated again in Python before it becomes an `Invoice`.

Set `BAR_COST_DEMO_CLAUDE_MODEL` (e.g. `sonnet`) to pick the model. The
`claude -p` route suits a demo on one machine. A product for customers would
call the Anthropic API with the same schema and an API key.

Claude reads are not in `make verify`, because they cost a model call each
time. `tests/test_llm_extract.py` pins the extractor's behaviour with a fake
`claude`.

## How it fits together

- `src/bar_cost_demo/core/invoices.py`: the invoice model and the `Extractor`
  seam. The demo reads a plain-text layout with `RuleBasedExtractor`; reading
  scanned PDFs or emailed invoices with an LLM means adding one more
  extractor behind the same protocol. Nothing downstream changes, and
  `arithmetic_issues` still checks whatever it extracted.
- `src/bar_cost_demo/core/catalog.py`: price history, recipes and the menu.
- `src/bar_cost_demo/features/`: `invoice`, `prices` and `margin`, one
  folder each, discovered by name.
- `data/`: two sample invoices (Nordic Beverage AB, Stockholm Fresh AB),
  `price_history.csv`, `recipes.csv` and `menu.csv`. Amounts are SEK, menu
  prices exclude VAT.

Money is `Decimal` throughout, never float. Features use core, never each
other, and import-linter enforces it.

Made from the harness Python template: the finish, tamper and after-edit
gates apply, `make new-feature NAME=<name>` adds a feature, and every
feature has a row in `FEATURES.md` that `make verify` checks.
