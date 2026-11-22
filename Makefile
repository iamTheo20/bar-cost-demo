# The one gate. Hooks, CI and agents all run `make check` and `make verify`.
GO ?= go

.DEFAULT_GOAL := help
.PHONY: help check lint types imports rules test verify new-feature

help: ## list targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "} {printf "  %-12s %s\n", $$1, $$2}'

check: lint types imports rules test ## everything that must be green before "done"

lint: ## gofmt (no unformatted file) and go vet
	@unformatted="$$(gofmt -l .)"; if [ -n "$$unformatted" ]; then echo "gofmt -w needed:"; echo "$$unformatted"; exit 1; fi
	$(GO) vet ./...

types: ## compile every package and every test binary (Go's type check)
	$(GO) build ./...
	$(GO) test -count=1 -run '^$$' ./... > /dev/null

imports: ## architecture tests: layers, feature independence, one paved road
	$(GO) test -count=1 ./internal

rules: ## .harness/rules.go: the project's FIX/WHY/SEE rules, with baseline
	$(GO) run .harness/rules.go

test: ## go test, with the race detector
	$(GO) test -count=1 -race ./...

verify: ## run every feature in FEATURES.md and compare its output
	$(GO) test -count=1 ./cmd/barcost -run '^TestFeaturesMD$$'

new-feature: ## scaffold internal/features/$(NAME) with a passing test
	$(GO) run ./tools/newfeature $(NAME)
