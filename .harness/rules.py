#!/usr/bin/env python3
"""Project rules (G5): checks whose errors name the fix.

    python3 .harness/rules.py [PATH ...]     check these files, or the whole project
    python3 .harness/rules.py --init-baseline     record today's violations (once)
    python3 .harness/rules.py --update-baseline   shrink the baseline to what is left

Every finding is printed as

    <RULE> <path>:<line> <what is wrong>
      FIX: <the replacement>
      WHY: <one line>
      SEE: <a correct example or RULES.md>

    HR001  suppression comments (noqa, type: ignore, pyright: ignore)
    HR002  bare except, or except Exception that only passes
    HR003  banned APIs: print, breakpoint, pdb.set_trace, os.system, plus [[banned]] in rules.toml
    HR004  imports across a [[forbidden]] layer boundary in rules.toml
    HR005  `#` comments outside docstrings (D16); a shebang and an encoding line are fine
    HR006  files over 400 lines, functions over 60 (max_file_lines / max_function_lines)
    HR007  test functions that assert nothing, or patch the module they test

The baseline (.harness/baseline.json) maps "RULE path" to the number of old
violations that are tolerated. A key at or under its count is quiet; one over
it is reported in full. --update-baseline can only lower counts and drop keys,
never raise or add them, so re-baselining cannot absorb a new violation.

Exit 0 clean, 1 findings, 2 a usage or config error.
"""

from __future__ import annotations

import ast
import fnmatch
import io
import json
import re
import sys
import tokenize
import tomllib
from dataclasses import dataclass
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
HARNESS = ROOT / ".harness"
BASELINE = HARNESS / "baseline.json"
SKIP_DIRS = {
    ".venv",
    "venv",
    ".git",
    "node_modules",
    "__pycache__",
    ".harness",
    "build",
    "dist",
    ".tox",
}
DEFAULT_SEE = ".harness/RULES.md"
SUPPRESSION = re.compile(r"#\s*((ruff|flake8):\s*)?noqa|#\s*(type|pyright|mypy):\s*ignore|#\s*pylint:\s*disable", re.I)
DEFAULT_BANS = {
    "print": (
        "import logging; log = logging.getLogger(__name__); log.info(...)",
        "print output has no level, no logger name and vanishes in production",
    ),
    "breakpoint": (
        "remove it; reproduce the problem in a test instead",
        "a breakpoint hangs whatever runs this code",
    ),
    "pdb.set_trace": (
        "remove it; reproduce the problem in a test instead",
        "a breakpoint hangs whatever runs this code",
    ),
    "os.system": (
        "subprocess.run([...], check=True) with an argument list",
        "os.system goes through a shell and ignores failures",
    ),
}
TEST_ASSERT_CALLS = re.compile(r"(^|\.)(assert\w*|raises|warns|fail)$")


@dataclass(frozen=True)
class Finding:
    rule: str
    path: str
    line: int
    what: str
    fix: str
    why: str
    see: str = DEFAULT_SEE

    def render(self) -> str:
        return (
            f"{self.rule} {self.path}:{self.line} {self.what}\n  FIX: {self.fix}\n  WHY: {self.why}\n  SEE: {self.see}"
        )


class ConfigError(Exception):
    """rules.toml or the baseline cannot be used; the message says why."""


def load_config() -> dict:
    path = HARNESS / "rules.toml"
    if not path.is_file():
        return {}
    try:
        return tomllib.loads(path.read_text())
    except tomllib.TOMLDecodeError as e:
        raise ConfigError(f".harness/rules.toml: {e}") from None


def dotted(node: ast.AST) -> str:
    if isinstance(node, ast.Name):
        return node.id
    if isinstance(node, ast.Attribute):
        head = dotted(node.value)
        return f"{head}.{node.attr}" if head else ""
    return ""


def module_name(rel: str) -> str:
    parts = list(Path(rel).with_suffix("").parts)
    if parts and parts[0] == "src":
        parts = parts[1:]
    if parts and parts[-1] == "__init__":
        parts = parts[:-1]
    return ".".join(parts)


def is_test_file(rel: str) -> bool:
    name = Path(rel).name
    return name.startswith("test_") or name.endswith("_test.py") or Path(rel).parts[:1] == ("tests",)


def within(name: str, prefix: str) -> bool:
    return name == prefix or name.startswith(prefix + ".")


def comment_rules(rel: str, source: str) -> list[Finding]:
    found = []
    tokens = tokenize.generate_tokens(io.StringIO(source).readline)
    for tok in tokens:
        if tok.type != tokenize.COMMENT:
            continue
        line, text = tok.start[0], tok.string
        if SUPPRESSION.search(text):
            found.append(
                Finding(
                    "HR001",
                    rel,
                    line,
                    "suppression comment",
                    "fix what the check reports; if it is a false positive, change the rule, not the line",
                    "a silenced check protects nothing, and the next agent copies the silencing",
                )
            )
        elif not (line == 1 and text.startswith("#!")) and not (line <= 2 and re.match(r"#.*coding[:=]", text)):
            found.append(
                Finding(
                    "HR005",
                    rel,
                    line,
                    "comment outside a docstring",
                    "say it in a name, a docstring or a test; delete the comment",
                    "comments go stale and a stale comment is a bad example (D16)",
                )
            )
    return found


def except_rule(rel: str, node: ast.ExceptHandler) -> Finding | None:
    caught = dotted(node.type) if node.type is not None else ""
    only_passes = all(
        isinstance(s, ast.Pass) or (isinstance(s, ast.Expr) and isinstance(s.value, ast.Constant)) for s in node.body
    )
    if node.type is None or (caught in ("Exception", "BaseException") and only_passes):
        return Finding(
            "HR002",
            rel,
            node.lineno,
            "bare or swallowing except",
            "catch the specific exception you expect and handle it, or let it propagate",
            "a swallowed error turns a crash you can see into wrong output you cannot",
        )
    return None


def ban_rule(rel: str, node: ast.Call, bans: dict[str, tuple[str, str, str, frozenset[str]]]) -> Finding | None:
    name = dotted(node.func)
    if name in bans:
        fix, why, see, exempt = bans[name]
        if rel in exempt:
            return None
        return Finding("HR003", rel, node.lineno, f"{name}() call", fix, why, see)
    return None


def import_rule(rel: str, node: ast.AST, forbidden: list[dict]) -> list[Finding]:
    here = module_name(rel)
    names: list[str] = []
    if isinstance(node, ast.Import):
        names = [a.name for a in node.names]
    elif isinstance(node, ast.ImportFrom):
        base = node.module or ""
        if node.level:
            package = here.split(".")[: -node.level] if node.level <= here.count(".") else []
            base = ".".join([*package, base] if base else package)
        names = [base] + [f"{base}.{a.name}" for a in node.names]
    found = []
    for rule in forbidden:
        if within(here, rule["source"]) and any(within(n, rule["target"]) for n in names):
            found.append(
                Finding(
                    "HR004",
                    rel,
                    node.lineno,
                    f"{rule['source']} imports {rule['target']}",
                    rule.get(
                        "fix",
                        f"move the code, or go through the interface {rule['source']} may use",
                    ),
                    rule.get("why", "the layer boundary keeps the two sides independently changeable"),
                    rule.get("see", DEFAULT_SEE),
                )
            )
            break
    return found


def test_rules(rel: str, tree: ast.AST) -> list[Finding]:
    found = []
    under_test = Path(rel).stem.removeprefix("test_").removesuffix("_test")
    for node in ast.walk(tree):
        if isinstance(node, ast.FunctionDef | ast.AsyncFunctionDef) and node.name.startswith("test_"):
            asserts = any(
                isinstance(n, ast.Assert) or (isinstance(n, ast.Call) and TEST_ASSERT_CALLS.search(dotted(n.func)))
                for n in ast.walk(node)
            )
            if not asserts:
                found.append(
                    Finding(
                        "HR007",
                        rel,
                        node.lineno,
                        f"{node.name} asserts nothing",
                        "assert the observable result: a return value, a raised error, a written file",
                        "a test without an assertion passes whatever the code does",
                    )
                )
        if isinstance(node, ast.Call) and dotted(node.func).split(".")[-1] == "patch" and node.args:
            target = node.args[0]
            mocked = target.value if isinstance(target, ast.Constant) and isinstance(target.value, str) else ""
            parts = mocked.split(".")
            if len(parts) >= 2 and parts[-2] == under_test:
                found.append(
                    Finding(
                        "HR007",
                        rel,
                        node.lineno,
                        f"patches {mocked}, inside the module under test",
                        "patch the module's collaborators (I/O, clock, network), never the module itself",
                        "a test that mocks what it tests only tests the mock",
                    )
                )
    return found


def length_rules(rel: str, source: str, tree: ast.AST, cfg: dict) -> list[Finding]:
    max_file, max_func = cfg.get("max_file_lines", 400), cfg.get("max_function_lines", 60)
    found = []
    lines = source.count("\n") + (0 if source.endswith("\n") else 1)
    if lines > max_file:
        found.append(
            Finding(
                "HR006",
                rel,
                max_file + 1,
                f"file has {lines} lines (max {max_file})",
                "split it by feature: one folder per feature, one job per module",
                "long files are where agents pile on more of the same",
            )
        )
    for node in ast.walk(tree):
        if isinstance(node, ast.FunctionDef | ast.AsyncFunctionDef):
            length = (node.end_lineno or node.lineno) - node.lineno + 1
            if length > max_func:
                found.append(
                    Finding(
                        "HR006",
                        rel,
                        node.lineno,
                        f"{node.name} has {length} lines (max {max_func})",
                        "extract the steps into named functions",
                        "a long function hides its steps and its tests",
                    )
                )
    return found


def pattern_rules(rel: str, source: str, patterns: list[dict]) -> list[Finding]:
    """Project [[pattern]] rules: a regex per source line, limited to `paths` globs."""
    found = []
    for rule in patterns:
        if not any(fnmatch.fnmatch(rel, glob) for glob in rule.get("paths", ["*"])):
            continue
        try:
            regex = re.compile(rule["regex"])
        except (KeyError, re.error) as e:
            raise ConfigError(f".harness/rules.toml: pattern {rule.get('id', '?')} has a bad regex ({e})") from None
        for n, line in enumerate(source.splitlines(), 1):
            if regex.search(line):
                found.append(
                    Finding(
                        rule["id"],
                        rel,
                        n,
                        rule.get("what", f"matches {rule['regex']}"),
                        rule["fix"],
                        rule.get("why", "the project has one way to do this"),
                        rule.get("see", DEFAULT_SEE),
                    )
                )
    return found


def check_file(rel: str, cfg: dict, bans: dict[str, tuple[str, str, str, frozenset[str]]]) -> list[Finding]:
    source = (ROOT / rel).read_text(errors="replace")
    try:
        tree = ast.parse(source, filename=rel)
    except SyntaxError as e:
        return [
            Finding(
                "HR000",
                rel,
                e.lineno or 1,
                f"syntax error: {e.msg}",
                "fix the syntax",
                "nothing else can be checked",
            )
        ]
    found = comment_rules(rel, source) + length_rules(rel, source, tree, cfg)
    found += pattern_rules(rel, source, cfg.get("pattern", []))
    for node in ast.walk(tree):
        if isinstance(node, ast.ExceptHandler):
            found += [f for f in [except_rule(rel, node)] if f]
        elif isinstance(node, ast.Call):
            found += [f for f in [ban_rule(rel, node, bans)] if f]
        elif isinstance(node, ast.Import | ast.ImportFrom):
            found += import_rule(rel, node, cfg.get("forbidden", []))
    if is_test_file(rel):
        found += test_rules(rel, tree)
    return sorted(found, key=lambda f: (f.line, f.rule))


def project_files() -> list[str]:
    return sorted(
        str(p.relative_to(ROOT)) for p in ROOT.rglob("*.py") if not SKIP_DIRS & set(p.relative_to(ROOT).parts[:-1])
    )


def scan(paths: list[str]) -> list[Finding]:
    cfg = load_config()
    allow = set(cfg.get("allow", []))
    bans = {k: (fix, why, DEFAULT_SEE, frozenset[str]()) for k, (fix, why) in DEFAULT_BANS.items() if k not in allow}
    for b in cfg.get("banned", []):
        bans[b["call"]] = (
            b["fix"],
            b.get("why", "the project has one way to do this"),
            b.get("see", DEFAULT_SEE),
            frozenset(b.get("exempt", [])),
        )
    found = []
    for raw in paths:
        p = Path(raw)
        rel = str((p if p.is_absolute() else ROOT / p).resolve().relative_to(ROOT))
        if (ROOT / rel).is_file():
            found += check_file(rel, cfg, bans)
    return found


def counts(found: list[Finding]) -> dict[str, int]:
    out: dict[str, int] = {}
    for f in found:
        out[f"{f.rule} {f.path}"] = out.get(f"{f.rule} {f.path}", 0) + 1
    return dict(sorted(out.items()))


def read_baseline() -> dict[str, int]:
    if not BASELINE.is_file():
        return {}
    try:
        return json.loads(BASELINE.read_text())
    except ValueError as e:
        raise ConfigError(f".harness/baseline.json: {e}") from None


def report(found: list[Finding], baseline: dict[str, int]) -> int:
    current = counts(found)
    over = {k for k, n in current.items() if n > baseline.get(k, 0)}
    shown = [f for f in found if f"{f.rule} {f.path}" in over]
    for f in shown:
        print(f.render())
    for k in sorted(over):
        if baseline.get(k):
            print(f"{k}: {current[k]} found, baseline allows {baseline[k]}; fix the new ones")
    return 1 if shown else 0


def main(argv: list[str]) -> int:
    try:
        if argv[:1] == ["--init-baseline"]:
            if BASELINE.exists():
                print(
                    "rules: .harness/baseline.json exists; use --update-baseline to shrink it",
                    file=sys.stderr,
                )
                return 2
            BASELINE.write_text(json.dumps(counts(scan(project_files())), indent=2) + "\n")
            return 0
        if argv[:1] == ["--update-baseline"]:
            now, base = counts(scan(project_files())), read_baseline()
            shrunk = {k: min(n, now[k]) for k, n in base.items() if now.get(k)}
            BASELINE.write_text(json.dumps(dict(sorted(shrunk.items())), indent=2) + "\n")
            print(f"rules: baseline {sum(base.values())} -> {sum(shrunk.values())}")
            return 0
        return report(scan(argv or project_files()), read_baseline())
    except ConfigError as e:
        print(f"rules: {e}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
