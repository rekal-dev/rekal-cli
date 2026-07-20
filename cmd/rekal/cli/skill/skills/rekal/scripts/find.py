#!/usr/bin/env python3
"""Every mention of a term across the ledger — enumeration, not ranking.

Complete-set questions ("all / every / how many / which of the N") need the
whole mention list, and hand-writing the SQL means re-discovering the schema
every time (wrong column names -> Binder errors -> wrongful absence). This
script owns that toil: it runs the schema-correct sweep over `turns` and
prints one compact line per mention, in time order.

  find.py "<term>"            # all mentions, any role
  find.py "<term>" <role>     # filter: human | assistant | human_steering | summary

Output: `<session_id> t<turn> <ts> <role>: …context around the match…`
followed by a total. Drill a mention with
`rekal query --session <session_id> --offset <turn-2> --limit 5 | view.py`.

The sweep is COMPLETE (no LIMIT): every row is printed, each trimmed to a
window around the match — enumeration first, judgment (class-mapping, set
size, uptake) stays with the agent. Engine errors are forwarded verbatim,
never swallowed as an empty set.

Exit: 0 mentions found, 1 none, 2 usage / engine / parse error.
"""
from __future__ import annotations

import json
import os
import signal
import subprocess
import sys

try:
    signal.signal(signal.SIGPIPE, signal.SIG_DFL)
except (ImportError, AttributeError, ValueError):
    pass

# Context words shown on each side of the match — a display budget, not a gate.
WINDOW_WORDS = 12
ROLES = ("human", "assistant", "human_steering", "summary")


def sql_quote(term: str) -> str:
    """Escape for a single-quoted SQL string + ILIKE pattern."""
    term = term.replace("'", "''")
    # Literal ILIKE match: escape the pattern metacharacters.
    term = term.replace("\\", "\\\\").replace("%", "\\%").replace("_", "\\_")
    return term


def snippet(content: str, term: str) -> str:
    """A word-window around the first match, ellipsized. Case-insensitive."""
    flat = " ".join(content.split())
    low = flat.lower()
    pos = low.find(term.lower())
    if pos < 0:
        return flat[: WINDOW_WORDS * 12]
    before = flat[:pos].split()[-WINDOW_WORDS:]
    after = flat[pos:].split()[: WINDOW_WORDS + max(1, len(term.split()))]
    head = "…" if len(flat[:pos].split()) > WINDOW_WORDS else ""
    tail = "…" if len(flat[pos:].split()) > len(after) else ""
    return f"{head}{' '.join(before + after)}{tail}"


def main() -> int:
    args = [a for a in sys.argv[1:] if a not in ("-h", "--help")]
    if len(args) != len(sys.argv) - 1 or not 1 <= len(args) <= 2 or not args[0].strip():
        print(
            'usage: find.py "<term>" [role]\n'
            "  Enumerate every ledger mention of a term (time order, complete).\n"
            f"  role ∈ {'|'.join(ROLES)}",
            file=sys.stderr,
        )
        return 2
    term = args[0].strip()
    role = args[1] if len(args) == 2 else ""
    if role and role not in ROLES:
        print(f"find: unknown role {role!r} (want {'|'.join(ROLES)})", file=sys.stderr)
        return 2

    where = f"content ILIKE '%{sql_quote(term)}%' ESCAPE '\\'"
    if role:
        where += f" AND role = '{role}'"
    sql = (
        "SELECT session_id, turn_index, ts, role, content "
        f"FROM turns WHERE {where} ORDER BY ts, session_id, turn_index"
    )

    rekal = os.environ.get("REKAL_BIN", "rekal")
    try:
        p = subprocess.run([rekal, "query", sql], capture_output=True, text=True, timeout=120)
    except (OSError, subprocess.TimeoutExpired) as e:
        print(f"find: cannot run {rekal}: {e}", file=sys.stderr)
        return 2
    if p.returncode != 0:
        # Forward the engine's own words — a query error is not an empty set.
        sys.stderr.write(p.stderr or p.stdout or f"find: rekal exit {p.returncode}\n")
        return 2

    rows = []
    for line in p.stdout.splitlines():
        line = line.strip()
        if not line:
            continue
        try:
            rows.append(json.loads(line))
        except json.JSONDecodeError:
            print(f"find: parse_error on: {line[:120]}", file=sys.stderr)
            return 2

    if not rows:
        print(f'(no mentions of "{term}"{" role=" + role if role else ""} — '
              "reformulate: synonyms, entity names, the other speaker's uptake)")
        return 1

    sessions: dict[str, int] = {}
    for r in rows:
        sid = str(r.get("session_id") or "?")
        sessions[sid] = sessions.get(sid, 0) + 1
        ts = str(r.get("ts") or "")[:16]
        print(f'{sid} t{r.get("turn_index")} {ts} {r.get("role") or "?"}: {snippet(str(r.get("content") or ""), term)}')
    print(f'total {len(rows)} mentions in {len(sessions)} sessions — "{term}"')
    return 0


if __name__ == "__main__":
    sys.exit(main())
