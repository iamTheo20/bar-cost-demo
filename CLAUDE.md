# Working in bar-cost-demo

- `make check` must be green before you say a task is done; the finish gate refuses otherwise.
- New features come from `make new-feature NAME=<name>`, never by hand. Copy what it generates.
- A feature is done when its row in `FEATURES.md` exists and `make verify` passes.
- Logging goes through `bar_cost_demo.core.log.get_logger`, settings through `bar_cost_demo.core.config.load`.
- `.harness/`, `.claude/`, `pyproject.toml`, `ruff.toml`, `Makefile` and `tests/conftest.py` are read-only to you. If a rule is wrong, say so in your final message.
- Add dependencies with `uv add <pkg>`.
