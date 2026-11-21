#!/usr/bin/env python3
"""Config ratchet (G9): fail when this branch weakens the checks against a base ref.

    python3 .harness/config_guard.py --base origin/main

CI runs the base branch's copy of this file, so a PR cannot weaken the guard
that judges it. Each finding names what got weaker:

    CONFIG001  ruff.toml: a selected rule dropped, a rule ignored, a longer line limit, more excludes
    CONFIG002  pyproject.toml: pyright below strict, fewer paths checked, an import contract removed
    CONFIG003  .harness/rules.toml: a rule removed, a higher length limit, a default ban allowed
    CONFIG004  .harness/baseline.json: a count raised, or a key added for a rule that is not new
    CONFIG005  Makefile: a `check` prerequisite dropped
    CONFIG006  .claude/settings.json: a deny rule dropped

Stricter and unchanged config passes. A deliberate weakening is merged by Top
Baldy over a red CI, never by an agent. Exit 0 pass, 1 weaker, 2 unreadable.
"""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
import tomllib
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
RANKS = {"off": 0, "basic": 1, "standard": 2, "strict": 3}


class Unreadable(Exception):
    """A config file cannot be parsed; the guard cannot judge it."""


def at_base(base: str, rel: str) -> str | None:
    r = subprocess.run(["git", "-C", str(ROOT), "show", f"{base}:{rel}"], capture_output=True, text=True, check=False)
    return r.stdout if r.returncode == 0 else None


def now(rel: str) -> str | None:
    p = ROOT / rel
    return p.read_text() if p.is_file() else None


def toml(text: str | None, rel: str) -> dict:
    if text is None:
        return {}
    try:
        return tomllib.loads(text)
    except tomllib.TOMLDecodeError as e:
        raise Unreadable(f"{rel}: {e}") from None


def as_json(text: str | None, rel: str) -> dict:
    if text is None:
        return {}
    try:
        return json.loads(text)
    except ValueError as e:
        raise Unreadable(f"{rel}: {e}") from None


def ruff(old: dict, new: dict) -> list[str]:
    o, n = old.get("lint", old), new.get("lint", new)
    out = [f"rule {r} no longer selected" for r in sorted(set(o.get("select", [])) - set(n.get("select", [])))]
    out += [f"rule {r} now ignored" for r in sorted(set(n.get("ignore", [])) - set(o.get("ignore", [])))]
    if new.get("line-length", 88) > old.get("line-length", 88):
        out.append(f"line-length {old.get('line-length', 88)} -> {new.get('line-length', 88)}")
    for key in ("exclude", "extend-exclude"):
        out += [f"{key} adds {p}" for p in sorted(set(new.get(key, [])) - set(old.get(key, [])))]
    o_pfi, n_pfi = o.get("per-file-ignores", {}), n.get("per-file-ignores", {})
    for glob, rules in n_pfi.items():
        out += [f"per-file-ignores {glob} adds {r}" for r in sorted(set(rules) - set(o_pfi.get(glob, [])))]
    return out


def pyproject(old: dict, new: dict) -> list[str]:
    out = []
    op, np_ = old.get("tool", {}).get("pyright", {}), new.get("tool", {}).get("pyright", {})
    if RANKS.get(np_.get("typeCheckingMode", "standard"), 0) < RANKS.get(op.get("typeCheckingMode", "standard"), 0):
        out.append(f"pyright {op.get('typeCheckingMode')} -> {np_.get('typeCheckingMode', 'standard')}")
    out += [f"pyright no longer checks {p}" for p in sorted(set(op.get("include", [])) - set(np_.get("include", [])))]
    oc = old.get("tool", {}).get("importlinter", {}).get("contracts", [])
    nc = new.get("tool", {}).get("importlinter", {}).get("contracts", [])
    kept = [json.dumps(c, sort_keys=True) for c in nc]
    out += [f"import-linter contract removed or changed: {c.get('name')}" for c in oc
            if json.dumps(c, sort_keys=True) not in kept]
    return out


def rules_toml(old: dict, new: dict) -> list[str]:
    out = []
    for kind in ("banned", "forbidden", "pattern"):
        kept = [json.dumps(e, sort_keys=True) for e in new.get(kind, [])]
        for e in old.get(kind, []):
            if json.dumps(e, sort_keys=True) not in kept:
                label = e.get("call") or e.get("id") or f"{e.get('source')} -> {e.get('target')}"
                out.append(f"{kind} rule removed or changed: {label}")
    for key, default in (("max_file_lines", 400), ("max_function_lines", 60)):
        if new.get(key, default) > old.get(key, default):
            out.append(f"{key} {old.get(key, default)} -> {new.get(key, default)}")
    out += [f"default ban now allowed: {a}" for a in sorted(set(new.get("allow", [])) - set(old.get("allow", [])))]
    return out


def baseline(old: dict, new: dict, old_rules: dict, new_rules: dict) -> list[str]:
    old_ids = {e.get("id") for e in old_rules.get("pattern", [])}
    new_ids = {e.get("id") for e in new_rules.get("pattern", [])} - old_ids
    out = []
    for key, count in sorted(new.items()):
        if key in old and count > old[key]:
            out.append(f"baseline {key} {old[key]} -> {count}")
        elif key not in old and key.split(" ")[0] not in new_ids:
            out.append(f"baseline adds {key} ({count}) for a rule that is not new")
    return out


def check_prereqs(text: str | None) -> set[str]:
    m = re.search(r"^check:([^#\n]*)", text or "", re.M)
    return set(m[1].split()) if m else set()


def denies(data: dict) -> set[str]:
    return set(data.get("permissions", {}).get("deny", []))


def findings(base: str) -> list[str]:
    out = []
    out += [f"CONFIG001 ruff.toml: {f}" for f in
            ruff(toml(at_base(base, "ruff.toml"), "ruff.toml"), toml(now("ruff.toml"), "ruff.toml"))]
    out += [f"CONFIG002 pyproject.toml: {f}" for f in
            pyproject(toml(at_base(base, "pyproject.toml"), "pyproject.toml"), toml(now("pyproject.toml"), "pyproject.toml"))]
    rel = ".harness/rules.toml"
    old_rules, new_rules = toml(at_base(base, rel), rel), toml(now(rel), rel)
    out += [f"CONFIG003 {rel}: {f}" for f in rules_toml(old_rules, new_rules)]
    rel = ".harness/baseline.json"
    out += [f"CONFIG004 {rel}: {f}" for f in
            baseline(as_json(at_base(base, rel), rel), as_json(now(rel), rel), old_rules, new_rules)]
    dropped = check_prereqs(at_base(base, "Makefile")) - check_prereqs(now("Makefile"))
    out += [f"CONFIG005 Makefile: check no longer runs {t}" for t in sorted(dropped)]
    rel = ".claude/settings.json"
    out += [f"CONFIG006 {rel}: deny rule dropped: {d}" for d in
            sorted(denies(as_json(at_base(base, rel), rel)) - denies(as_json(now(rel), rel)))]
    return out


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--base", required=True, help="the ref to compare against, e.g. origin/main")
    ap.add_argument("--root", type=Path, help="the project to judge (default: the folder above .harness/)")
    args = ap.parse_args()
    global ROOT
    if args.root:
        ROOT = args.root.resolve()
    try:
        found = findings(args.base)
    except Unreadable as e:
        print(f"config guard: cannot read {e}; a config the guard cannot read is not a pass")
        return 2
    for f in found:
        print(f)
    if found:
        print(f"config guard: {len(found)} weakening(s) against {args.base}. FIX: restore the check, or ask Top "
              "Baldy, who alone merges a deliberate weakening over a red CI.")
        return 1
    print(f"config guard: no check config is weaker than {args.base}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
