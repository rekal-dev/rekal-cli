#!/usr/bin/env python3
"""Episode confidence gate for HUNT.

Reads recall stdout JSON from stdin (or a file path arg).

Exit codes:
  0 — PASS_EPISODE: top >= 0.9, or (n>=2 and gap >= 0.04)
  1 — SILENCE: below bars / no results
  3 — ROUTE_KNOWLEDGE: non-empty knowledge block (do NOT inject episodes)
  2 — usage / parse / IO error

One stdout line, machine-parseable. Bars live only here — not in skill prose.
"""
from __future__ import annotations

import json
import sys

TOP_MIN = 0.9
GAP_MIN = 0.04


def main() -> int:
    if len(sys.argv) > 2 or (len(sys.argv) == 2 and sys.argv[1] in ("-h", "--help")):
        print("usage: hunt-gate.py [recall.json]  # else stdin", file=sys.stderr)
        return 2
    try:
        raw = open(sys.argv[1], encoding="utf-8").read() if len(sys.argv) == 2 else sys.stdin.read()
        data = json.loads(raw)
    except (OSError, json.JSONDecodeError) as e:
        print(f"SILENCE top=0 gap=0 reason=parse_error:{e}", file=sys.stderr)
        return 2

    knowledge = data.get("knowledge") or []
    if knowledge:
        paths = []
        for k in knowledge[:5]:
            if isinstance(k, dict) and k.get("path"):
                paths.append(str(k["path"]))
        suffix = (" " + " ".join(paths)) if paths else ""
        print(f"ROUTE_KNOWLEDGE{suffix}")
        return 3

    results = data.get("results") or []
    if not results:
        print("SILENCE top=0 gap=0 reason=no_results")
        return 1

    scores = []
    for r in results:
        try:
            scores.append(float(r.get("score", 0)))
        except (TypeError, ValueError):
            scores.append(0.0)
    scores.sort(reverse=True)
    top = scores[0]
    if len(scores) >= 2:
        gap = scores[0] - scores[1]
        if top >= TOP_MIN or gap >= GAP_MIN:
            print(f"PASS_EPISODE top={top:.4f} gap={gap:.4f}")
            return 0
        print(f"SILENCE top={top:.4f} gap={gap:.4f} reason=below_gate")
        return 1

    # Single hit: require absolute confidence. Gap alone must not pass.
    if top >= TOP_MIN:
        print(f"PASS_EPISODE top={top:.4f} gap=0.0000")
        return 0
    print(f"SILENCE top={top:.4f} gap=0.0000 reason=single_below_top")
    return 1


if __name__ == "__main__":
    sys.exit(main())
