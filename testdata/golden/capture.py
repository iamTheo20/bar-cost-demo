"""Record golden outputs from the Python implementation (commit af673a8).

Run once, from the repo root, while the Python app still existed:
    uv run python testdata/golden/capture.py
Kept as provenance for cases.json; the Python app it drove has since been
replaced by the Go module, and the Go golden test replays these cases.
LLM cases replay a recorded real `claude -p` reply through testdata/fakeclaude.
"""

import json
import os
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
GOLDEN = ROOT / "testdata" / "golden"
FAKE = ROOT / "testdata" / "fakeclaude"

CASES: list[dict[str, object]] = [
    {"name": "usage-no-args", "args": []},
    {"name": "usage-unknown-feature", "args": ["nope"]},
    {"name": "hello-default", "args": ["hello"]},
    {"name": "hello-world", "args": ["hello", "World"]},
    {"name": "hello-ada", "args": ["hello", "Ada"]},
    {"name": "invoice-missing-file", "args": ["invoice"]},
    {"name": "invoice-two-files", "args": ["invoice", "a.txt", "b.txt"]},
    {"name": "invoice-unsupported-suffix", "args": ["invoice", "data/menu.csv"]},
]
for feature in ("invoice", "prices", "margin"):
    for inv in ("NB-48211", "SF-1093"):
        path = f"data/invoices/{inv}.txt"
        CASES.append({"name": f"{feature}-{inv}", "args": [feature, path]})
        CASES.append({"name": f"{feature}-{inv}-summary", "args": [feature, path, "--summary"]})
    CASES.append(
        {"name": f"{feature}-BS-2207-pdf-llm", "args": [feature, "data/invoices/BS-2207.pdf"],
         "claude_reply": "claude/BS-2207.pdf.json"}
    )
    CASES.append(
        {"name": f"{feature}-BS-2207-pdf-llm-summary",
         "args": [feature, "data/invoices/BS-2207.pdf", "--summary"],
         "claude_reply": "claude/BS-2207.pdf.json"}
    )
    CASES.append(
        {"name": f"{feature}-SF-1093-llm-summary",
         "args": [feature, "--llm", "data/invoices/SF-1093.txt", "--summary"],
         "claude_reply": "claude/SF-1093.txt.json"}
    )
CASES.append(
    {"name": "invoice-claude-failed", "args": ["invoice", "data/invoices/BS-2207.pdf"],
     "claude_reply": "claude/failed.json"}
)


def main() -> int:
    out_dir = GOLDEN / "out"
    out_dir.mkdir(exist_ok=True)
    (GOLDEN / "claude" / "failed.json").write_text(
        json.dumps({"type": "result", "subtype": "error_max_turns", "is_error": True,
                    "result": "gave up", "structured_output": None}) + "\n"
    )
    manifest = []
    for case in CASES:
        env = {k: v for k, v in os.environ.items() if not k.startswith("BAR_COST_DEMO_")}
        args_file = None
        if "claude_reply" in case:
            env["PATH"] = f"{FAKE}{os.pathsep}{env['PATH']}"
            env["BARCOST_FAKE_CLAUDE_REPLY"] = str(GOLDEN / str(case["claude_reply"]))
            args_file = out_dir / f"{case['name']}.claude-args"
            env["BARCOST_FAKE_CLAUDE_ARGS"] = str(args_file)
        r = subprocess.run(
            [sys.executable, "-m", "bar_cost_demo", *case["args"]],
            cwd=ROOT, env=env, capture_output=True, text=True, check=False,
        )
        (out_dir / f"{case['name']}.stdout").write_text(r.stdout)
        err_tail = r.stderr.strip().splitlines()[-1] if r.stderr.strip() else ""
        entry = {**case, "exit": 0 if r.returncode == 0 else (2 if r.returncode == 2 else 1),
                 "stderr_last_line": err_tail}
        manifest.append(entry)
        sys.stdout.write(f"{case['name']}: exit {r.returncode} {err_tail}\n")
    (GOLDEN / "cases.json").write_text(json.dumps(manifest, indent=2) + "\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())
