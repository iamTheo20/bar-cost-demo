# Features

Every user-facing feature, how to run it and what correct output is.
`go test ./cmd/barcost -run TestFeaturesMD` runs each row and compares the
output exactly (it replaces the Python `make verify`). A feature is not done
until it has a row here and that test passes. Changes to this file are
reviewed like code: an expected output may only change when the behaviour is
meant to.

Full reports, error cases and the Claude (PDF / `--llm`) path are pinned
against the Python version's recorded output by `TestGoldenParity` and
`testdata/golden/`.

| feature | command | expected |
| --- | --- | --- |
| hello | `go run ./cmd/barcost hello World` | `Hello, World!` |
| invoice-clean | `go run ./cmd/barcost invoice data/invoices/NB-48211.txt --summary` | `invoice NB-48211: 4 lines, 5823.20 SEK, 0 arithmetic issue(s)` |
| invoice-error | `go run ./cmd/barcost invoice data/invoices/SF-1093.txt --summary` | `invoice SF-1093: 4 lines, 2308.00 SEK, 1 arithmetic issue(s)` |
| prices-drinks | `go run ./cmd/barcost prices data/invoices/NB-48211.txt --summary` | `prices NB-48211: 2 of 4 items moved 5% or more (largest: Fever-Tree Tonic 20cl +12.2%)` |
| prices-food | `go run ./cmd/barcost prices data/invoices/SF-1093.txt --summary` | `prices SF-1093: 2 of 4 items moved 5% or more (largest: Avocado +23.4%)` |
| margin-drinks | `go run ./cmd/barcost margin data/invoices/NB-48211.txt --summary` | `margin NB-48211: 1 of 2 affected menu items below target (Gin & Tonic 75.2% < 76%)` |
| margin-food | `go run ./cmd/barcost margin data/invoices/SF-1093.txt --summary` | `margin SF-1093: 1 of 2 affected menu items below target (Guacamole & chips 77.8% < 80%)` |
