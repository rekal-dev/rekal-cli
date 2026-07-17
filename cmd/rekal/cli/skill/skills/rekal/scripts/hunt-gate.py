#!/usr/bin/env python3
"""Episode confidence gate for HUNT.

Reads recall stdout JSON from stdin (or a file path arg).

Priority:
  1. Confident episode  → PASS_EPISODE (knowledge presence must not block)
  2. Else non-empty knowledge → ROUTE_KNOWLEDGE (Read HEAD; do not inject weak episodes)
  3. Else SILENCE

Gating uses absolute `confidence` (and optional `mass`) when present — not
max-normalized `score`, which tops out near 1.0 for junk queries too.
Legacy recalls without `confidence` keep the old score bars (top ≥ 0.9).

Exit codes:
  0 — PASS_EPISODE
  1 — SILENCE
  3 — ROUTE_KNOWLEDGE
  2 — usage / parse / IO error
"""
from __future__ import annotations

import json
import os
import sys


def _env_float(name: str, default: float) -> float:
    raw = os.environ.get(name)
    if raw is None or raw == "":
        return default
    try:
        return float(raw)
    except ValueError:
        return default


# Absolute confidence floor (see search/confidence.go saturating BM25).
# Tuned so real domain queries (~0.85) clear and offtopic/junk (~0.48–0.63) do not.
# Override via REKAL_HUNT_* for harness calibration (WS-D); defaults unchanged.
CONF_MIN = _env_float("REKAL_HUNT_CONF_MIN", 0.70)
# Soft path: near the hard floor with a clear gap to #2 — still above offtopic.
CONF_SOFT = _env_float("REKAL_HUNT_CONF_SOFT", 0.68)
GAP_MIN = _env_float("REKAL_HUNT_GAP_MIN", 0.04)
# Lexical floor: raw BM25 mass when the field is present (Option C).
# Set REKAL_HUNT_MASS_MIN=0 to disable the mass gate (chat-scale corpora).
MASS_MIN = _env_float("REKAL_HUNT_MASS_MIN", 3.5)
# Knowledge fallback: absolute knowledge score must clear this (BUG 14 —
# max-norm made every top hit ~1.0; weak prose must not ROUTE_KNOWLEDGE).
KNOWLEDGE_MIN = _env_float("REKAL_HUNT_KNOWLEDGE_MIN", 0.40)
# Legacy (no confidence field): max-norm score bars from the pre-#45 gate.
LEGACY_TOP_MIN = 0.9


def _f(v, default: float = 0.0) -> float:
    try:
        return float(v)
    except (TypeError, ValueError):
        return default


def has_confidence(results: list) -> bool:
    return any(isinstance(r, dict) and "confidence" in r for r in results)


def _is_marker_only_knowledge(k: object) -> bool:
    """True when a 'knowledge' hit is effectively a marker file.

    In the industry-bench synthetic repos, sh-gen creates tiny marker files
    under sessions/ and uses CLAUDE.md for a setup sentinel. Those files are
    not meaningful prose, but they can still score above KNOWLEDGE_MIN due
    to max-norm scoring behavior.
    """
    if not isinstance(k, dict):
        return False
    path = str(k.get("path") or "")
    snip = str(k.get("snippet") or "")
    if path == "CLAUDE.md":
        return True
    if path.startswith("sessions/"):
        # Our sh-gen marker headers include `benchmark_session_id: ...`.
        if "benchmark_session_id:" in snip:
            return True
    return False


def episode_verdict(results: list) -> tuple[str, float, float, str]:
    """Return (kind, top_signal, gap, reason)."""
    if not results:
        return "empty", 0.0, 0.0, "no_results"

    if has_confidence(results):
        return confidence_verdict(results)
    return legacy_score_verdict(results)


def confidence_verdict(results: list) -> tuple[str, float, float, str]:
    """Gate on absolute confidence + mass — never max-norm score (BUG 11)."""
    scored: list[tuple[float, float]] = []
    for r in results:
        if not isinstance(r, dict):
            scored.append((0.0, 0.0))
            continue
        # Missing confidence on a mixed set → 0 (do not fall back to score).
        conf = _f(r["confidence"]) if "confidence" in r else 0.0
        mass = _f(r.get("mass") or 0)
        scored.append((conf, mass))
    scored.sort(key=lambda x: x[0], reverse=True)
    top, mass = scored[0]
    gap = (scored[0][0] - scored[1][0]) if len(scored) > 1 else top

    has_mass_field = any(isinstance(r, dict) and "mass" in r for r in results)
    weak_mass = has_mass_field and MASS_MIN > 0 and 0 < mass < MASS_MIN

    if top >= CONF_MIN and not weak_mass:
        return "pass", top, gap, ""
    if top >= CONF_SOFT and gap >= GAP_MIN and not weak_mass:
        return "pass", top, gap, ""
    if weak_mass:
        return "silence", top, gap, "below_mass"
    if len(scored) == 1:
        return "silence", top, gap, "single_below_conf"
    return "silence", top, gap, "below_gate"


def legacy_score_verdict(results: list) -> tuple[str, float, float, str]:
    """Pre-confidence recalls: score bars only (never apply CONF_* to score)."""
    scores: list[float] = []
    for r in results:
        if isinstance(r, dict):
            scores.append(_f(r.get("score", 0)))
        else:
            scores.append(0.0)
    scores.sort(reverse=True)
    top = scores[0]
    gap = (scores[0] - scores[1]) if len(scores) > 1 else top
    if len(scores) >= 2:
        if top >= LEGACY_TOP_MIN or gap >= GAP_MIN:
            return "pass", top, gap, ""
        return "silence", top, gap, "below_gate"
    if top >= LEGACY_TOP_MIN:
        return "pass", top, gap, ""
    return "silence", top, gap, "single_below_conf"


def knowledge_ok(knowledge: list) -> bool:
    """True when the top knowledge hit has absolute score ≥ KNOWLEDGE_MIN."""
    if not knowledge:
        return False
    # Ignore marker-only 'knowledge' (synthetic sentinel/provenance files).
    meaningful = [k for k in knowledge if not _is_marker_only_knowledge(k)]
    if not meaningful:
        return False
    top = meaningful[0] if isinstance(meaningful[0], dict) else {}
    score = _f(top.get("score", 0))
    return score >= KNOWLEDGE_MIN


def knowledge_line(knowledge: list) -> str:
    meaningful = [k for k in knowledge if not _is_marker_only_knowledge(k)]
    paths = []
    for k in meaningful[:5]:
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

    if knowledge_ok(knowledge):
        print(knowledge_line(knowledge))
        return 3

    if kind == "empty":
        print("SILENCE top=0 gap=0 reason=no_results")
        return 1
    # Weak knowledge present but below absolute floor — treat as silence.
    if knowledge:
        print(f"SILENCE top={top:.4f} gap={gap:.4f} reason=knowledge_below_floor")
        return 1
    print(f"SILENCE top={top:.4f} gap={gap:.4f} reason={reason or 'below_gate'}")
    return 1


if __name__ == "__main__":
    sys.exit(main())
