#!/usr/bin/env python3
"""Route a rekal recall JSON: knowledge vs episode inject vs silence.

Preferred tip entry after `rekal "<q>"`. Wraps hunt-gate semantics and prints
an agent-facing label. On INJECT it also prints a compact candidate digest —
top candidates with a trimmed snippet, the rest as one-line id+confidence —
so the agent never needs to re-read the raw recall JSON (which costs ~7x the
tokens for the same decision information).

Exit codes:
  0 — KNOWLEDGE or INJECT (act on stdout)
  1 — SILENCE
  2 — error

Stdout:
  KNOWLEDGE [path…]
  INJECT top=… gap=…      (+ digest lines below)
  SILENCE top=… gap=… reason=…
"""
from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

DIGEST_SNIPPET_TOP = 3  # candidates that keep a snippet
DIGEST_SNIPPET_WORDS = 30


def print_digest(data: dict) -> None:
    """Compact candidate view: enough to pick a drill target, nothing more."""
    results = data.get("results") or []
    for i, r in enumerate(results[:DIGEST_SNIPPET_TOP]):
        words = (r.get("snippet") or "").split()
        snip = " ".join(words[:DIGEST_SNIPPET_WORDS]) + ("…" if len(words) > DIGEST_SNIPPET_WORDS else "")
        turn = r.get("snippet_turn_index")
        turn_s = f" t{turn}" if turn is not None else ""
        print(f"  {i + 1}. {r.get('session_id')} conf={r.get('confidence')}{turn_s} \"{snip}\"")
    rest = results[DIGEST_SNIPPET_TOP :]
    if rest:
        tail = " ".join(f"{r.get('session_id')}({r.get('confidence')})" for r in rest)
        print(f"  {DIGEST_SNIPPET_TOP + 1}-{len(results)}: {tail}")


def main() -> int:
    if len(sys.argv) > 2 or (len(sys.argv) == 2 and sys.argv[1] in ("-h", "--help")):
        print("usage: recall-route.py [recall.json]  # else stdin", file=sys.stderr)
        return 2
    try:
        raw = open(sys.argv[1], encoding="utf-8").read() if len(sys.argv) == 2 else sys.stdin.read()
        # Validate JSON early so we fail loud before the child.
        json.loads(raw)
    except (OSError, json.JSONDecodeError) as e:
        print(f"SILENCE top=0 gap=0 reason=parse_error:{e}", file=sys.stderr)
        return 2

    gate = Path(__file__).resolve().parent / "hunt-gate.py"
    proc = subprocess.run(
        [sys.executable, str(gate)],
        input=raw,
        text=True,
        capture_output=True,
        check=False,
    )
    line = (proc.stdout or proc.stderr or "").strip().splitlines()
    line = line[0] if line else "SILENCE top=0 gap=0 reason=empty_gate"
    code = proc.returncode

    if code == 3 or line.startswith("ROUTE_KNOWLEDGE"):
        rest = line[len("ROUTE_KNOWLEDGE") :].strip()
        print(f"KNOWLEDGE{(' ' + rest) if rest else ''}")
        return 0
    if code == 0 or line.startswith("PASS_EPISODE"):
        # Normalize label for the tip.
        print(line.replace("PASS_EPISODE", "INJECT", 1))
        print_digest(json.loads(raw))
        return 0
    if code == 1 or line.startswith("SILENCE"):
        print(line if line.startswith("SILENCE") else f"SILENCE reason={line}")
        return 1
    print(line, file=sys.stderr)
    return 2


if __name__ == "__main__":
    sys.exit(main())
