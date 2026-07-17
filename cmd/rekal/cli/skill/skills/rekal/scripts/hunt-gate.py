#!/usr/bin/env python3
"""Episode confidence gate for HUNT.

Reads recall stdout JSON from stdin (or a file path arg).

Priority:
  1. Confident episode  → PASS_EPISODE (knowledge presence must not block)
  2. Else non-empty knowledge → ROUTE_KNOWLEDGE (Read HEAD; do not inject weak episodes)
  3. Else SILENCE

Exit codes:
  0 — PASS_EPISODE: top >= 0.9, or (n>=2 and gap >= 0.04)
  1 — SILENCE: below bars / no results / no knowledge fallback
  3 — ROUTE_KNOWLEDGE: knowledge present and episode below bars
  2 — usage / parse / IO error

One stdout line, machine-parseable. Bars live only here — not in skill prose.
"""
from __future__ import annotations

import json
import sys

TOP_MIN = 0.9
GAP_MIN = 0.04


def episode_verdict(results: list) -> tuple[str, float, float]:
    """Return (kind, top, gap) where kind is pass|silence|empty."""
    if not results:
        return "empty", 0.0, 0.0
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
            return "pass", top, gap
        return "silence", top, gap
    # Single hit: require absolute confidence. Gap alone must not pass.
    if top >= TOP_MIN:
        return "pass", top, 0.0
    return "silence", top, 0.0


def knowledge_line(knowledge: list) -> str:
    paths = []
    for k in knowledge[:5]:
        if isinstance(k, dict) and k.get("path"):
            paths.append(str(k["path"]))
    suffix = (" " + " ".join(paths)) if paths else ""
    return f"ROUTE_KNOWLEDGE{suffix}"


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

    results = data.get("results") or []
    knowledge = data.get("knowledge") or []
    kind, top, gap = episode_verdict(results)

    if kind == "pass":
        print(f"PASS_EPISODE top={top:.4f} gap={gap:.4f}")
        return 0

    # Knowledge is a fallback when the episode gate fails — not an override
    # of a confident session match (repos with prose always have docs).
    if knowledge:
        print(knowledge_line(knowledge))
        return 3

    if kind == "empty":
        print("SILENCE top=0 gap=0 reason=no_results")
        return 1
    if len(results) == 1:
        print(f"SILENCE top={top:.4f} gap={gap:.4f} reason=single_below_top")
        return 1
    print(f"SILENCE top={top:.4f} gap={gap:.4f} reason=below_gate")
    return 1


if __name__ == "__main__":
    sys.exit(main())
