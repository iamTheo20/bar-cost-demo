"""Run every feature in FEATURES.md, compare its output, and keep the evidence.

Each run writes .harness/evidence/<timestamp>/ with one `<feature>.out` per
feature and a summary.json the finish gate reads. The expected column is
either exact output in backticks, or `loss falls`: every `loss=<number>` in
the output is collected into `<feature>.losses.json`, and the run passes only
if the last loss is below the first.
"""

import json
import re
import shlex
import subprocess
import sys
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
EVIDENCE = ROOT / ".harness" / "evidence"
ROW = re.compile(
    r"^\|\s*(?P<feature>[^|`]+?)\s*\|\s*`(?P<command>[^`]+)`\s*\|"
    r"\s*(?:`(?P<expected>[^`]*)`|(?P<check>loss falls))\s*\|$"
)
LOSS = re.compile(r"loss\s*[=:]\s*(-?\d+(?:\.\d+)?(?:e-?\d+)?)", re.IGNORECASE)


def rows() -> list[tuple[str, str, str | None]]:
    """(feature, command, expected output or None for `loss falls`) per table row."""
    found: list[tuple[str, str, str | None]] = []
    for line in (ROOT / "FEATURES.md").read_text().splitlines():
        m = ROW.match(line.strip())
        if m:
            found.append((m["feature"], m["command"], None if m["check"] else m["expected"]))
    return found


def judge(feature: str, got: str, expected: str | None, run: Path) -> str | None:
    """None when the output is right, else why it is not."""
    if expected is not None:
        return None if got == expected else f"gave {got!r}, expected {expected!r}"
    losses = [float(v) for v in LOSS.findall(got)]
    (run / f"{feature}.losses.json").write_text(json.dumps(losses) + "\n")
    if len(losses) < 2:
        return "printed no loss curve (need two or more `loss=<number>` values)"
    if losses[-1] >= losses[0]:
        return f"loss did not fall: {losses[0]} -> {losses[-1]}"
    return None


def main() -> int:
    """Exit 1 on any failure, 2 when FEATURES.md lists nothing."""
    table = rows()
    if not table:
        sys.stderr.write("verify: FEATURES.md lists no features\n")
        return 2
    stamp = time.strftime("%Y%m%d-%H%M%S") + f"-{time.time_ns() // 1000 % 1_000_000:06d}"
    run = EVIDENCE / stamp
    run.mkdir(parents=True, exist_ok=True)
    results: list[dict[str, object]] = []
    for feature, command, expected in table:
        r = subprocess.run(
            shlex.split(command), cwd=ROOT, capture_output=True, text=True, check=False
        )
        got = r.stdout.strip()
        (run / f"{feature}.out").write_text(
            r.stdout + (f"\n--- stderr ---\n{r.stderr}" if r.stderr else "")
        )
        problem = (
            f"exited {r.returncode}" if r.returncode != 0 else judge(feature, got, expected, run)
        )
        results.append(
            {
                "feature": feature,
                "command": command,
                "exit": r.returncode,
                "ok": problem is None,
                "problem": problem,
            }
        )
        sys.stdout.write(f"ok   {feature}\n" if problem is None else f"FAIL {feature}: {problem}\n")
    ok = all(res["ok"] for res in results)
    summary = {"timestamp": stamp, "ok": ok, "features": results}
    (run / "summary.json").write_text(json.dumps(summary, indent=2) + "\n")
    sys.stdout.write(f"evidence: {run.relative_to(ROOT)}\n")
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
