# bar-cost-demo

Made from the harness Python template: every check is `make check`, every
feature is proven by `make verify`.

```sh
uv sync
make check verify
uv run bar-cost-demo hello World
make new-feature NAME=greet
```

Layout: one folder per feature in `src/bar_cost_demo/features/`, shared code in
`src/bar_cost_demo/core/`. Features may import core, never each other, and core
never imports a feature (import-linter enforces both).
