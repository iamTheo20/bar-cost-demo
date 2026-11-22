---
status: accepted
---

# Rewrite the app in Go, keeping byte-for-byte output parity with Python

Theo decided to replace the Python implementation entirely with one Go module (`cmd/barcost`, `internal/core`, `internal/features`), because the product is heading for a single self-contained server binary with few dependencies, and Go's standard library covers HTTP, templates, JSON, CSV and structured logging. The rewrite was gated on parity: the Python CLI's outputs were recorded in `testdata/golden/` first, and `TestGoldenParity` requires the Go CLI to print exactly the same.

## Consequences

- Money stays exact. Go has no standard decimal type, so `internal/core/decimal` reimplements the subset of Python's `decimal` the app uses, under its default context (28 significant digits, round half to even), so rounding of percentages and cents is unchanged. Do not "simplify" it to `float64` or to a different rounding rule; the golden tests will fail, and they should.
- The harness that guarded the Python code (`Makefile`, `pyproject.toml`, `ruff.toml`, `uv.lock`, `.harness/`, `.github/workflows/check.yml`) does not understand Go and must be replaced by the repository owner. Until then `make check` and CI fail.
- The layering rules that import-linter enforced are now `internal/arch_test.go`.
