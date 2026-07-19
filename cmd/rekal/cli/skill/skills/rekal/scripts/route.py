#!/usr/bin/env python3
"""Route a rekal recall JSON: knowledge vs episode inject vs silence.

The single entry after `rekal "<q>"`. Reads recall stdout JSON on stdin (or a
file path arg), gates on absolute `confidence`, and prints one agent-facing
label plus — on INJECT — a compact candidate digest (top candidates with a
trimmed snippet, the rest as one-line id+confidence), so the agent never
re-reads the raw recall JSON (which costs ~7x the tokens for the same decision).

This script reports deterministic data for the agent's judgment; it does not
decide the corpus or switch profiles. In particular, BM25 `mass` is reported
as a `low_mass` signal — never used to silence a confident hit. A confident
low-mass hit is a real dialogue-shaped match; whether to trust or widen is the
agent's call. Junk is already rejected by the absolute confidence floor.

Priority:
  1. Confident episode        -> INJECT (knowledge presence must not block it)
  2. Else non-empty knowledge -> KNOWLEDGE (Read HEAD; don't inject weak episodes)
  3. Else                     -> SILENCE

Gating uses absolute `confidence` when present — not max-normalized `score`,
which tops out near 1.0 for junk queries too. Legacy recalls without
`confidence` keep the old score bars (top >= 0.9).

Exit codes:
  0 — KNOWLEDGE or INJECT (act on stdout)
  1 — SILENCE
  2 — usage / parse / IO error
"""
from __future__ import annotations

import json
import sys

# Absolute confidence floor (see search/confidence.go saturating BM25).
# Tuned so real domain queries (~0.85) clear and offtopic/junk (~0.48-0.63) do not.
CONF_MIN = 0.70
# Soft path: near the hard floor with a clear gap to #2 — still above offtopic.
CONF_SOFT = 0.68
GAP_MIN = 0.04
# "Low mass" boundary: raw BM25 mass below this is lexically thin (typical of
# dialogue / semantic hits). REPORTED as a signal, never used to silence.
LOW_MASS = 3.5
# Knowledge fallback: absolute knowledge score must clear this.
KNOWLEDGE_MIN = 0.40
# Legacy (no confidence field): max-norm score bars from the pre-confidence gate.
LEGACY_TOP_MIN = 0.9

# Digest shape.
DIGEST_SNIPPET_TOP = 3
DIGEST_SNIPPET_WORDS = 30


def _f(v, default: float = 0.0) -> float:
    try:
        return float(v)
    except (TypeError, ValueError):
        return default


def has_confidence(results: list) -> bool:
    return any(isinstance(r, dict) and "confidence" in r for r in results)


def episode_verdict(results: list) -> tuple[str, float, float, bool, str]:
    """Return (kind, top_signal, gap, low_mass, reason)."""
    if not results:
        return "empty", 0.0, 0.0, False, "no_results"
    if has_confidence(results):
        return confidence_verdict(results)
    kind, top, gap, reason = legacy_score_verdict(results)
    return kind, top, gap, False, reason


def confidence_verdict(results: list) -> tuple[str, float, float, bool, str]:
    """Gate on absolute confidence. Mass is reported, never a veto."""
    scored: list[tuple[float, float]] = []
    for r in results:
        if not isinstance(r, dict):
            scored.append((0.0, 0.0))
            continue
        conf = _f(r["confidence"]) if "confidence" in r else 0.0
        mass = _f(r.get("mass") or 0)
        scored.append((conf, mass))
    scored.sort(key=lambda x: x[0], reverse=True)
    top, mass = scored[0]
    gap = (scored[0][0] - scored[1][0]) if len(scored) > 1 else top

    has_mass_field = any(isinstance(r, dict) and "mass" in r for r in results)
    low_mass = has_mass_field and 0 < mass < LOW_MASS

    if top >= CONF_MIN:
        return "pass", top, gap, low_mass, ""
    if top >= CONF_SOFT and gap >= GAP_MIN:
        return "pass", top, gap, low_mass, ""
    if len(scored) == 1:
        return "silence", top, gap, low_mass, "single_below_conf"
    return "silence", top, gap, low_mass, "below_gate"


def legacy_score_verdict(results: list) -> tuple[str, float, float, str]:
    """Pre-confidence recalls: score bars only (never apply CONF_* to score)."""
    scores: list[float] = []
    for r in results:
        scores.append(_f(r.get("score", 0)) if isinstance(r, dict) else 0.0)
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
    if not knowledge:
        return False
    top = knowledge[0] if isinstance(knowledge[0], dict) else {}
    return _f(top.get("score", 0)) >= KNOWLEDGE_MIN


def knowledge_paths(knowledge: list) -> str:
    paths = [str(k["path"]) for k in knowledge[:5] if isinstance(k, dict) and k.get("path")]
    return (" " + " ".join(paths)) if paths else ""


def print_digest(data: dict) -> None:
    """Compact candidate view: enough to pick a drill target, nothing more."""
    results = data.get("results") or []
    for i, r in enumerate(results[:DIGEST_SNIPPET_TOP]):
        words = (r.get("snippet") or "").split()
        snip = " ".join(words[:DIGEST_SNIPPET_WORDS]) + ("…" if len(words) > DIGEST_SNIPPET_WORDS else "")
        turn = r.get("snippet_turn_index")
        turn_s = f" t{turn}" if turn is not None else ""
        print(f'  {i + 1}. {r.get("session_id")} conf={r.get("confidence")}{turn_s} "{snip}"')
    rest = results[DIGEST_SNIPPET_TOP:]
    if rest:
        tail = " ".join(f'{r.get("session_id")}({r.get("confidence")})' for r in rest)
        print(f"  {DIGEST_SNIPPET_TOP + 1}-{len(results)}: {tail}")


def main() -> int:
    if len(sys.argv) > 2 or (len(sys.argv) == 2 and sys.argv[1] in ("-h", "--help")):
        print("usage: route.py [recall.json]  # else stdin", file=sys.stderr)
        return 2
    try:
        raw = open(sys.argv[1], encoding="utf-8").read() if len(sys.argv) == 2 else sys.stdin.read()
        data = json.loads(raw)
    except (OSError, json.JSONDecodeError) as e:
        print(f"SILENCE top=0 gap=0 reason=parse_error:{e}", file=sys.stderr)
        return 2

    results = data.get("results") or []
    knowledge = data.get("knowledge") or []
    kind, top, gap, low_mass, reason = episode_verdict(results)

    if kind == "pass":
        print(f"INJECT top={top:.4f} gap={gap:.4f} low_mass={'true' if low_mass else 'false'}")
        print_digest(data)
        return 0

    if knowledge_ok(knowledge):
        print(f"KNOWLEDGE{knowledge_paths(knowledge)}")
        return 0

    if kind == "empty":
        print("SILENCE top=0 gap=0 reason=no_results")
        return 1
    if knowledge:
        print(f"SILENCE top={top:.4f} gap={gap:.4f} reason=knowledge_below_floor")
        return 1
    print(f"SILENCE top={top:.4f} gap={gap:.4f} reason={reason or 'below_gate'}")
    return 1


if __name__ == "__main__":
    sys.exit(main())
