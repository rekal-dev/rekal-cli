#!/usr/bin/env python3
"""Route a rekal recall JSON: knowledge vs episode inject vs silence.

The single entry after `rekal "<q>"`. Reads recall stdout JSON on stdin (or a
file path arg), gates on absolute `confidence`, and prints one agent-facing
label plus — on INJECT — a compact candidate digest (top candidates with a
trimmed snippet, the rest as one-line id+confidence), so the agent never
re-reads the raw recall JSON (which costs ~7x the tokens for the same decision).

This script reports deterministic data for the agent's judgment; it does not
decide the corpus or switch profiles. In particular, raw BM25 `mass` is reported
verbatim — never bucketed on a tuned boundary, never used to silence a confident
hit. Low mass is a lexically thin, dialogue-shaped match; whether to trust or
widen is the agent's call. Junk is already rejected by the absolute confidence
floor.

Priority:
  1. Confident episode        -> INJECT (knowledge presence must not block it)
  2. Else non-empty knowledge -> KNOWLEDGE path=score ... (Read HEAD if a file
                                 is a real prose hit — the agent judges the
                                 per-file score distribution, not one number)
  3. Else                     -> SILENCE

Two substrates, two gates — because only one has a corpus-invariant signal.

Episodes gate on absolute `confidence` — never max-normalized `score`, which
tops out near 1.0 for junk queries too. Confidence is saturating BM25
(search/confidence.go): a bounded transform of the raw score, so its junk floor
holds across corpora and a fixed `CONF_MIN` is a property, not a fit. A missing
per-result `confidence` is treated as 0.0: the engine emits `confidence`/`mass`
with `omitempty`, so an all-offtopic set (every confidence 0.0) drops the field
— that is noise, and it silences. Any real hit, even pure-semantic, carries
confidence > 0. (Pre-confidence index DBs self-heal: recall auto-rebuilds the
index on version change.)

Knowledge has no such invariant. Its `score` blends semantic cosine, whose junk
baseline drifts with corpus and model, so no fixed floor generalizes (SOUL.md:
no tuned constant decides). route.py therefore does not silence on a knowledge
threshold — it reports the top knowledge score verbatim and lets the agent judge
whether it is a real prose hit. Silence on the knowledge substrate is the
agent's call; the script only reports absence (no knowledge at all -> SILENCE).

Exit codes:
  0 — KNOWLEDGE or INJECT (act on stdout)
  1 — SILENCE
  2 — usage / parse / IO error
"""
from __future__ import annotations

import json
import signal
import sys

# Behave like a normal Unix tool when a reader closes the pipe early (e.g. the
# digest is piped through `head`): die on SIGPIPE instead of tracing back.
try:
    signal.signal(signal.SIGPIPE, signal.SIG_DFL)
except (ImportError, AttributeError, ValueError):
    pass

# Absolute confidence floor. Confidence is saturating BM25 (search/confidence.go)
# — a bounded transform whose junk baseline is a property of the transform, not
# of any one corpus, so this floor generalizes (real domain ~0.85, junk ~0.48-
# 0.63 hold across corpora by construction; SOUL.md permits a gate on an
# engine-calibrated invariant).
CONF_MIN = 0.70
# Soft path: near the hard floor with a clear gap to #2 — still above offtopic.
CONF_SOFT = 0.68
GAP_MIN = 0.04
# Raw BM25 `mass` is reported verbatim, never bucketed on a fixed boundary: mass
# is not corpus-invariant (it scales with corpus term stats and doc lengths), so
# any "low mass" cut would be a tuned constant (SOUL.md: no tuned constant
# decides). Low mass means a lexically thin, dialogue-shaped hit — the agent
# reads the number and judges whether to trust or widen. Mass never silences.
# No KNOWLEDGE floor. The knowledge `score` blends semantic cosine, whose junk
# baseline drifts per corpus and model — a fixed floor overfits the corpus it
# was measured on (SOUL.md: no tuned constant decides). The engine calibrates
# episode `confidence` to be corpus-invariant (saturating BM25), so the episode
# gate below stays; the knowledge score has no such invariant, so route.py
# reports it verbatim and the agent judges whether it is a real prose hit.

# Digest shape. These are output BUDGETS (how much to print), not judgment
# gates — they bound the digest's token cost without changing the verdict or the
# ranking. The id(conf) tail dominates the digest at large -n (≈75% of tokens at
# -n 100) yet the agent drills from the top, so the tail is capped and the
# remainder summarized as a count (drill deeper via `query --session`/SQL).
DIGEST_SNIPPET_TOP = 3
DIGEST_SNIPPET_WORDS = 30
DIGEST_TAIL_MAX = 12


def _f(v, default: float = 0.0) -> float:
    try:
        return float(v)
    except (TypeError, ValueError):
        return default


def episode_verdict(results: list) -> tuple[str, float, float, float, str]:
    """Gate on absolute confidence. Returns the top hit's raw mass, reported
    verbatim — never a veto, never bucketed.

    Missing per-result confidence is 0.0 (omitempty drops zero-confidence hits),
    so an all-offtopic set silences. Score is never used to gate.
    """
    if not results:
        return "empty", 0.0, 0.0, 0.0, "no_results"

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

    if top >= CONF_MIN:
        return "pass", top, gap, mass, ""
    if top >= CONF_SOFT and gap >= GAP_MIN:
        return "pass", top, gap, mass, ""
    if len(scored) == 1:
        return "silence", top, gap, mass, "single_below_conf"
    return "silence", top, gap, mass, "below_gate"


def knowledge_hits(knowledge: list, n: int = 5) -> str:
    """Top knowledge files as `path=score`, score-ordered. The whole point is to
    hand the agent the score *distribution*, not one number: a flat cluster near
    the noise floor (e.g. 0.51 0.49 0.48) is no real hit; a clear leader that
    then falls off (0.93 0.92 0.60) is a real prose hit. That reference point is
    what the agent judges against — no fixed floor decides (SOUL.md)."""
    out = []
    for k in knowledge[:n]:
        if isinstance(k, dict) and k.get("path"):
            out.append(f'{k["path"]}={_f(k.get("score", 0)):.2f}')
    return " ".join(out)


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
        shown = rest[:DIGEST_TAIL_MAX]
        tail = " ".join(f'{r.get("session_id")}({r.get("confidence")})' for r in shown)
        more = len(rest) - len(shown)
        suffix = f" (+{more} more, drill or -n to see them)" if more > 0 else ""
        hi = DIGEST_SNIPPET_TOP + len(shown)
        print(f"  {DIGEST_SNIPPET_TOP + 1}-{hi}: {tail}{suffix}")


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
    kind, top, gap, mass, reason = episode_verdict(results)

    sem = data.get("semantic")
    warming = isinstance(sem, dict) and bool(sem.get("retryable"))

    def done(code: int) -> int:
        # Trailing note (kept after the verdict/digest so line 1 stays the
        # verdict for tools that read it): the deep-semantic layer is still
        # loading, so these results are keyword+LSA only. Re-run with backoff
        # for full quality — the daemon warms in a few seconds.
        if warming:
            print("SEMANTIC warming — keyword+LSA only; re-run for full quality (backoff 2s/4s/8s)")
        return code

    if kind == "pass":
        print(f"INJECT top={top:.4f} gap={gap:.4f} mass={mass:.2f}")
        print_digest(data)
        return done(0)

    # Episode gate failed. Knowledge has no corpus-invariant floor, so report
    # the per-file score distribution as a signal and let the agent judge —
    # don't silence on a tuned threshold (SOUL.md: no tuned constant decides).
    if knowledge:
        hits = knowledge_hits(knowledge)
        print(f"KNOWLEDGE {hits}" if hits else "KNOWLEDGE")
        return done(0)

    # Nothing on either substrate — machine silence.
    if kind == "empty":
        print("SILENCE top=0 gap=0 reason=no_results")
        return done(1)
    print(f"SILENCE top={top:.4f} gap={gap:.4f} reason={reason or 'below_gate'}")
    return done(1)


if __name__ == "__main__":
    sys.exit(main())
