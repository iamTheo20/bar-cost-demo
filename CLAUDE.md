# Working in bar-cost-demo

- The app is Go (standard library only). `make check` must be green before you say a task is done; the finish gate refuses otherwise.
- New features come from `make new-feature NAME=<name>`, never by hand. Copy what it generates, then register the feature in `cmd/barcost/main.go`.
- A feature is done when its row in `FEATURES.md` exists and `make verify` passes.
- Logging goes through `internal/core/log` (`log.Get(name)`), settings through `internal/core/config` (`config.Load()`).
- `.harness/`, `.claude/`, `.github/`, `Makefile` and `internal/arch_test.go` are read-only to you. If a rule is wrong, say so in your final message.
- Prefer the standard library. A new module dependency (`go get <module>`) needs Top Baldy's approval first.
