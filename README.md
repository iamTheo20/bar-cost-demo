# bar-cost-demo

A demo of automating supplier invoices and price swings for a bar or
restaurant: read each invoice, catch the supplier's mistakes, flag price
jumps against what you paid last time, and show which dishes and drinks a
price change pushes below their target margin.

Written in Go (1.24+, standard library only).

```sh
go run ./cmd/barcost invoice data/invoices/SF-1093.txt   # catches 2 x 145.00 invoiced as 300.00
go run ./cmd/barcost prices  data/invoices/SF-1093.txt   # avocado +23.4%, beef +10.0%
go run ./cmd/barcost margin  data/invoices/NB-48211.txt  # Gin & Tonic drops to 75.2%, target 76%
go vet ./... && go test ./...
```

Add `--summary` to any command for its one-line result. `go build -o barcost
./cmd/barcost` gives a single static binary.

### Real invoices: PDFs and photos, read by Claude

```sh
go run ./cmd/barcost invoice data/invoices/BS-2207.pdf          # Swedish number formats and VAT
go run ./cmd/barcost prices  data/invoices/BS-2207.pdf          # lime +21.1%, gin +6.6%
go run ./cmd/barcost invoice data/invoices/SF-1093.txt --llm    # force Claude on a text invoice too
```

Any `.pdf`, `.png`, `.jpg`, `.jpeg` or `.webp` invoice (or a text one with
`--llm`) is read by `internal/core/llmextract`, which calls the local Claude
Code CLI (`claude -p`) on your own login, so no API key is needed. Each call
takes about 7 seconds and, on the recorded runs, cost about USD 0.02–0.03.
What it does:

- Claude sees the invoice in an empty temporary folder, has only the Read tool,
  saves no session and loads no MCP servers.
- Its answer must match a JSON schema: amounts as decimal strings, ISO date,
  one object per line.
- It is told to copy numbers as printed and never correct one, so the
  arithmetic check still catches supplier mistakes. On `BS-2207.pdf` it keeps
  the wrong 140,00 for 3 kg of limes at 46,00, and `invoice` flags it.
- It is told to ignore instructions written inside the invoice.
- Every field is validated again in Go before it becomes an `Invoice`.

Set `BAR_COST_DEMO_CLAUDE_MODEL` (e.g. `sonnet`) to pick the model. The
`claude -p` route suits a demo on one machine; a product for customers would
call the Anthropic API with the same schema and an API key (see
`docs/MVP_PLAN.md`).

Tests never call a model: `llmextract_test.go` drives the extractor with a fake
runner, and the golden test replays real recorded Claude replies through
`testdata/fakeclaude/claude`.

## Settings

| variable | default | meaning |
| --- | --- | --- |
| `BAR_COST_DEMO_LOG_LEVEL` | `WARNING` | `DEBUG`, `INFO`, `WARNING` or `ERROR`; logs go to stderr via `log/slog` |
| `BAR_COST_DEMO_DATA_DIR` | `data` | folder with `price_history.csv`, `recipes.csv`, `menu.csv` |
| `BAR_COST_DEMO_CLAUDE_MODEL` | (CLI default) | model passed to `claude --model` |

## How it fits together

- `cmd/barcost`: the command. It lists the features by name and prints what
  they return; exit 2 is a usage error, exit 1 a failed feature.
- `internal/core/invoices`: the invoice model and the `Extractor` interface.
  `RuleBasedExtractor` reads the demo's plain-text layout; `llmextract`
  reads everything else behind the same interface, and nothing downstream
  changes. `ArithmeticIssues` checks whatever was extracted.
- `internal/core/catalog`: price history, recipes and the menu.
- `internal/core/decimal`: exact decimal money. Never float. It mirrors
  Python's `decimal` default context (28 digits, round half to even) so the Go
  CLI prints exactly what the Python version did.
- `internal/core/config`, `internal/core/log`: the one way to read settings
  and the one way to log.
- `internal/features/`: `hello`, `invoice`, `prices` and `margin`, one package
  each.
- `data/`: sample invoices (Nordic Beverage AB, Stockholm Fresh AB as text;
  Bryggeriet Soder AB as PDF), `price_history.csv`, `recipes.csv` and
  `menu.csv`. Amounts are SEK, menu prices exclude VAT.

Features use core, never each other, and core never imports a feature;
`internal/arch_test.go` enforces that, and that settings and logging go
through their one package.

## Tests

- `go test ./...` runs everything, offline.
- `TestFeaturesMD` runs each row of `FEATURES.md` and compares the output
  exactly. A feature is done when it has a row there and this passes.
- `TestGoldenParity` replays the 30 cases recorded from the Python version in
  `testdata/golden/` (every feature, full reports, error cases, the Claude
  path) and requires identical stdout, exit code, error message and `claude`
  command line.

## Adding a feature

1. Create `internal/features/<name>/<name>.go` with `const Name = "<name>"`
   and `func Run(args []string) (string, error)`, plus a table-driven test.
2. Register it in `Features` in `cmd/barcost/main.go`.
3. Add a row to `FEATURES.md` and run `go test ./...`.

## Status of the Python harness

The app used to be Python. The harness files around it (`Makefile`,
`pyproject.toml`, `ruff.toml`, `uv.lock`, `.harness/`, `.claude/`,
`.github/workflows/check.yml`) are read-only to agents and still describe the
Python toolchain, so `make check`, `make verify` and CI will fail until they
are replaced. See the pull request that introduced the Go version for the
exact list.

The old Python source (`src/`, `tests/`, `scripts/`) is still in the tree on
this branch: the overnight run that wrote the Go version was not permitted to
delete it. It is no longer the implementation; remove it with
`git rm -r src tests scripts` once the Go version is accepted.

## Planning

`docs/MVP_PLAN.md` is the plan for turning this demo into a sellable MVP,
`docs/grilling-log.md` the decisions behind it, `GLOSSARY.md` the domain
language and `docs/adr/` the decisions that are hard to reverse.
