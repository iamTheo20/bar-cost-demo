# The one gate. Hooks, CI and agents all run `make check` and `make verify`.
UV ?= uv

.DEFAULT_GOAL := help
.PHONY: help check lint types imports rules test verify new-feature

help: ## list targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "} {printf "  %-12s %s\n", $$1, $$2}'

check: lint types imports rules test ## everything that must be green before "done"

lint: ## ruff
	$(UV) run --frozen ruff check .

types: ## pyright --strict
	$(UV) run --frozen pyright

imports: ## import-linter layer and independence contracts
	$(UV) run --frozen lint-imports

rules: ## .harness/rules.py: the project's FIX/WHY/SEE rules, with baseline
	$(UV) run --frozen python .harness/rules.py

test: ## pytest
	$(UV) run --frozen pytest

verify: ## run every feature in FEATURES.md and compare its output
	$(UV) run --frozen python scripts/verify.py

new-feature: ## scaffold src/bar_cost_demo/features/$(NAME) with a passing test
	$(UV) run --frozen python scripts/new_feature.py $(NAME)
