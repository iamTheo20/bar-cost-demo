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
