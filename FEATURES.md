# Features

Every user-facing feature, how to run it and what correct output is.
`make verify` runs each row and compares the output exactly. A feature is not
done until it has a row here and `make verify` passes. Changes to this
file are reviewed like code: an expected output may only change when the
behaviour is meant to.

| feature | command | expected |
| --- | --- | --- |
| hello | `uv run bar-cost-demo hello World` | `Hello, World!` |
