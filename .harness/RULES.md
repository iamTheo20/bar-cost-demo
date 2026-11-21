# Project rules

The changelog of this project's checks. Every rule exists because of a
mistake, and every finding names its fix. The engine is `.harness/rules.py`;
project settings are in `.harness/rules.toml`; tolerated old violations are
in `.harness/baseline.json`, which may only shrink.

Agents cannot edit this folder (the tamper gate refuses). If a rule is wrong,
say so in your final message.

| Rule | Checks | Fix | Added | Because |
| --- | --- | --- | --- | --- |
| HR001 | suppression comments (`noqa`, `type: ignore`, file-level forms) | fix what the check reports | harness G5 | a silenced check protects nothing and gets copied |
| HR002 | bare `except`, `except Exception: pass` | catch the specific error, or let it propagate | harness G5 | swallowed errors become wrong output |
| HR003 | `print`, `breakpoint`, `pdb.set_trace`, `os.system`, and `[[banned]]` calls | the paved road named in the finding | harness G5 | one way to do each job |
| HR004 | imports across a `[[forbidden]]` boundary | go through the allowed interface | harness G5 | layers stay independently changeable |
| HR005 | `#` comments outside docstrings (DESIGN D16) | say it in a name, docstring or test | harness G5 | stale comments are bad examples |
| HR006 | files over 400 lines, functions over 60 | split by feature, extract steps | harness G5 | long files attract more of the same |
| HR007 | tests that assert nothing, or patch their own module | assert the result; mock collaborators only | harness G5 | such tests pass whatever the code does |

New rules are added with `/new-rule` (G8): one row here per rule, with the
incident that caused it and the date.
