# Project rules

The changelog of this project's checks. Every rule exists because of a
mistake, and every finding names its fix. The engine is `.harness/rules.go`
(`go run .harness/rules.go`, stdlib only); project settings are in
`.harness/rules.json`; tolerated old violations are in
`.harness/baseline.json`, which may only shrink. `.harness/config_guard.go`
fails CI when a PR weakens any of these, the Makefile's `check` target,
`.claude/settings.json` or `internal/arch_test.go`.

Agents cannot edit this folder (the tamper gate refuses). If a rule is wrong,
say so in your final message.

| Rule | Checks | Fix | Added | Because |
| --- | --- | --- | --- | --- |
| HR001 | suppression comments (`//nolint`, `//lint:ignore`, `//lint:file-ignore`) | fix what the check reports | harness G5 | a silenced check protects nothing and gets copied |
| HR002 | empty `if err != nil {}`, `_ = call()` | return or handle the error | harness G5 | swallowed errors become wrong output |
| HR003 | `print`, `println`, std `log.Print*`/`Fatal*`/`Panic*`, and `banned` calls | the paved road named in the finding | harness G5 | one way to do each job |
| HR004 | imports across a `forbidden` boundary | go through the allowed interface | harness G5 | layers stay independently changeable |
| HR005 | retired with the Go port: Go doc comments are part of the language's tooling | — | harness G5 | — |
| HR006 | files over 400 lines, functions over 60 | split by feature, extract steps | harness G5 | long files attract more of the same |
| HR007 | `Test*` functions that never report through their `*testing.T` | assert the result with `t.Error`/`t.Fatal` | harness G5 | such tests pass whatever the code does |

Gofmt, `go vet`, compilation and `internal/arch_test.go` (layers, feature
independence, one way to read settings and log) run alongside these through
`make check`.

New rules are added with `/new-rule` (G8): one row here per rule, with the
incident that caused it and the date.
