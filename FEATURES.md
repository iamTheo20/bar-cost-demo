# Features

Every user-facing feature, how to run it and what correct output is.
`make verify` runs each row and compares the output exactly. A feature is not
done until it has a row here and `make verify` passes. Changes to this
file are reviewed like code: an expected output may only change when the
behaviour is meant to.

| feature | command | expected |
| --- | --- | --- |
| hello | `uv run bar-cost-demo hello World` | `Hello, World!` |
| invoice-clean | `uv run bar-cost-demo invoice data/invoices/NB-48211.txt --summary` | `invoice NB-48211: 4 lines, 5823.20 SEK, 0 arithmetic issue(s)` |
| invoice-error | `uv run bar-cost-demo invoice data/invoices/SF-1093.txt --summary` | `invoice SF-1093: 4 lines, 2308.00 SEK, 1 arithmetic issue(s)` |
| prices-drinks | `uv run bar-cost-demo prices data/invoices/NB-48211.txt --summary` | `prices NB-48211: 2 of 4 items moved 5% or more (largest: Fever-Tree Tonic 20cl +12.2%)` |
| prices-food | `uv run bar-cost-demo prices data/invoices/SF-1093.txt --summary` | `prices SF-1093: 2 of 4 items moved 5% or more (largest: Avocado +23.4%)` |
| margin-drinks | `uv run bar-cost-demo margin data/invoices/NB-48211.txt --summary` | `margin NB-48211: 1 of 2 affected menu items below target (Gin & Tonic 75.2% < 76%)` |
| margin-food | `uv run bar-cost-demo margin data/invoices/SF-1093.txt --summary` | `margin SF-1093: 1 of 2 affected menu items below target (Guacamole & chips 77.8% < 80%)` |
