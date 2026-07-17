#!/usr/bin/env python3
"""Episode confidence gate for HUNT.

Reads recall stdout JSON from stdin (or a file path arg).

Priority:
  1. Confident episode  → PASS_EPISODE (knowledge presence must not block)
  2. Else non-empty knowledge → ROUTE_KNOWLEDGE (Read HEAD; do not inject weak episodes)
  3. Else SILENCE

Gating uses absolute `confidence` (and optional `mass`) when present — not
max-normalized `score`, which tops out near 1.0 for junk queries too.
Legacy recalls without `confidence` fall back to `score` (weaker).

Exit codes:
  0 — PASS_EPISODE
  1 — SILENCE
  3 — ROUTE_KNOWLEDGE
  2 — usage / parse / IO error
"""
from __future__ import annotations

import json
import sys

# Absolute confidence floor (see search/confidence.go saturating BM25).
CONF_MIN = 0.70
# Soft path: slightly lower confidence but clear gap to #2.
CONF_SOFT = 0.55
GAP_MIN = 0.04
# Lexical floor: raw BM25 mass when the field is present (Option C).
MASS_MIN = 3.5


def episode_signal(r: dict) -> tuple[float, float]:
    """Return (confidence_or_score, mass) for one result."""
    conf = r.get("confidence")
    if conf is None:
        conf = r.get("score", 0)
    try:
        conf_f = float(conf)
    except (TypeError, ValueError):
        conf_f = 0.0
    try:
        mass = float(r.get("mass") or 0)
    except (TypeError, ValueError):
        mass = 0.0
    return conf_f, mass


def episode_verdict(results: list) -> tuple[str, float, float, str]:
    """Return (kind, top_conf, gap, reason)."""
    if not results:
        return "empty", 0.0, 0.0, "no_results"

    scored = []
    for r in results:
        conf, mass = episode_signal(r)
        scored.append((conf, mass))
    scored.sort(key=lambda x: x[0], reverse=True)
    top, mass = scored[0]
    gap = (scored[0][0] - scored[1][0]) if len(scored) > 1 else top

    # Lexical floor when mass is present and non-zero: weak BM25 must not
    # PASS via facet/max-norm inflation. mass==0 ⇒ pure semantic candidate.
    has_mass_field = any(isinstance(r, dict) and "mass" in r for r in results)
    weak_mass = has_mass_field and 0 < mass < MASS_MIN

    if top >= CONF_MIN and not weak_mass:
        return "pass", top, gap, ""
    if top >= CONF_SOFT and gap >= GAP_MIN and not weak_mass:
        return "pass", top, gap, ""
    if weak_mass:
        return "silence", top, gap, "below_mass"
    if len(scored) == 1:
        return "silence", top, gap, "single_below_conf"
    return "silence", top, gap, "below_gate"


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
    kind, top, gap, reason = episode_verdict(results)

    if kind == "pass":
        print(f"PASS_EPISODE top={top:.4f} gap={gap:.4f}")
        return 0

    if knowledge:
        print(knowledge_line(knowledge))
        return 3

    if kind == "empty":
        print("SILENCE top=0 gap=0 reason=no_results")
        return 1
    print(f"SILENCE top={top:.4f} gap={gap:.4f} reason={reason or 'below_gate'}")
    return 1


if __name__ == "__main__":
    sys.exit(main())
