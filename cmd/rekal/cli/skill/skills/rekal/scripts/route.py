#!/usr/bin/env python3
"""Route a rekal recall JSON: knowledge and/or episode inject vs silence.

The single entry after `rekal "<q>"`. Reads recall stdout JSON on stdin (or a
file path arg), applies a *super-low* episode floor so only empty/near-zero
confidence is machine-silenced, and prints agent-facing labels plus — on
INJECT — a seed digest: the top-DIGEST_WINDOW candidates each as
`sid conf=… t<turn> "snippet"`, in rank order, so the agent has enough context
to synthesize a (multi-hop) answer without drilling each, at a fraction of the
raw recall JSON's tokens.

Substrates are inclusive, not if/else. An episode hit and a knowledge hit can
both be real for a mixed question (convention at HEAD *and* why we chose it).
The script reports every substrate that has signal; the agent judges how to
combine them. Line 1 stays the primary verdict (`INJECT` / `KNOWLEDGE` /
`SILENCE`) for tools that read only the first line.

Labels are recommendations. The router is biased toward more data than
decision: emit `confidence` on the INJECT header (`top=` / `gap=`) and per
seed so the agent can weigh or drill — do not freeze a high corpus-shaped bar
(SOUL.md: no tuned constant decides). `mass` stays inside the script (never a
veto, never emitted). Knowledge `score` is not gated — reported verbatim for
the agent to judge.

Priority (inclusive):
  1. Episode above the low floor -> INJECT digest (+ KNOWLEDGE line when knowledge present)
  2. Else knowledge               -> KNOWLEDGE path=score ...
  3. Else                         -> SILENCE

Two substrates, one report — because mixed answers are real.

Episodes gate on absolute `confidence` — never max-normalized `score`, which
tops out near 1.0 for junk queries too. Confidence is saturating BM25
(search/confidence.go). A missing per-result `confidence` is treated as 0.0:
the engine emits `confidence`/`mass` with `omitempty`, so an all-offtopic set
(every confidence 0.0) drops the field — that is noise, and it silences. Any
real hit, even pure-semantic, carries confidence > 0. (Pre-confidence index
DBs self-heal: recall auto-rebuilds the index on version change.)

The shipped floor is deliberately low: dialogue-shaped hits often land well
below the old 0.70 coding bar while still carrying real evidence. Grey-band
confidence is reported for the agent to weigh; only empty / near-zero is an
obvious machine silence.

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
import os
import signal
import sys

# Behave like a normal Unix tool when a reader closes the pipe early (e.g. the
# digest is piped through `head`): die on SIGPIPE instead of tracing back.
try:
    signal.signal(signal.SIGPIPE, signal.SIG_DFL)
except (ImportError, AttributeError, ValueError):
    pass


def _env_float(name: str, default: float) -> float:
    """Optional harness override (REKAL_HUNT_*). Defaults are the shipped bars."""
    raw = os.environ.get(name)
    if raw is None or raw == "":
        return default
    try:
        return float(raw)
    except ValueError:
        return default


# Super-low episode floor. Only empty / near-zero confidence is an obvious
# machine silence; grey-band hits inject with `conf=` for the agent to weigh
# (SOUL.md: suggest, don't freeze a high tuned bar). Override via REKAL_HUNT_*
# for industry-bench harnesses.
CONF_MIN = _env_float("REKAL_HUNT_CONF_MIN", 0.25)
# Soft path: slightly below the hard floor with a clear gap to #2.
CONF_SOFT = _env_float("REKAL_HUNT_CONF_SOFT", 0.20)
GAP_MIN = _env_float("REKAL_HUNT_GAP_MIN", 0.02)
# Super-low knowledge *report* floor. Below this, scores are mathematically
# ignorable marker noise (LoCoMo CLAUDE.md ≈ 0.05–0.17) — omit the line to
# save tokens. Above it, report the distribution for the agent to judge.
# Not a "this is true" decision; just don't emit junk. Override via REKAL_HUNT_*.
KNOWLEDGE_MIN = _env_float("REKAL_HUNT_KNOWLEDGE_MIN", 0.25)
# Raw BM25 `mass` is reported verbatim, never bucketed on a fixed boundary: mass
# is not corpus-invariant (it scales with corpus term stats and doc lengths), so
# any "low mass" cut would be a tuned constant (SOUL.md: no tuned constant
# decides). Low mass means a lexically thin, dialogue-shaped hit — the agent
# reads the number and judges whether to trust or widen. Mass never silences.

# Digest shape. These are output BUDGETS (how much to print), not judgment
# gates — they bound the digest's token cost without changing the verdict or the
# ranking. The id(conf) tail dominates the digest at large -n (≈75% of tokens at
# -n 100) yet the agent drills from the top, so the tail is capped and the
# remainder summarized as a count (drill deeper via `query --session`/SQL).
# Seed-coverage window: inject the top-N candidates each WITH a snippet, so the
# agent has enough context to synthesize a (multi-hop) answer without drilling
# every session — LoCoMo-style top-K retrieval. Beyond N the agent reformulates
# / multi-searches (a sharper second query beats one query's tail) or raises -n.
DIGEST_WINDOW = 20
# Per-candidate snippet length, trimmed so 20 snippets stay cheap.
DIGEST_SNIPPET_WORDS = 15


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


def knowledge_reportable(knowledge: list) -> list:
    """Knowledge entries at/above the super-low report floor, score-ordered."""
    scored = []
    for k in knowledge:
        if not isinstance(k, dict) or not k.get("path"):
            continue
        sc = _f(k.get("score", 0))
        if sc >= KNOWLEDGE_MIN:
            scored.append((sc, k))
    scored.sort(key=lambda x: -x[0])
    return [k for _, k in scored]


def knowledge_hits(knowledge: list, n: int = 5) -> str:
    """Top knowledge files as `path=score`. Agent judges the distribution
    (clear leader vs flat cluster). Entries below KNOWLEDGE_MIN are omitted."""
    out = []
    for k in knowledge_reportable(knowledge)[:n]:
        out.append(f'{k["path"]}={_f(k.get("score", 0)):.2f}')
    return " ".join(out)


def print_digest(data: dict) -> None:
    """Seed context: every candidate in the window as
    `sid conf=… t<turn> "snippet"`, in rank order. Confidence is the
    signal the agent may weigh; mass stays inside the script.
    Wide enough (top-20) to seed a multi-hop answer without drilling each."""
    results = data.get("results") or []
    for r in results[:DIGEST_WINDOW]:
        words = (r.get("snippet") or "").split()
        snip = " ".join(words[:DIGEST_SNIPPET_WORDS]) + ("…" if len(words) > DIGEST_SNIPPET_WORDS else "")
        turn = r.get("snippet_turn_index")
        turn_s = f" t{turn}" if turn is not None else ""
        conf = _f(r["confidence"]) if isinstance(r, dict) and "confidence" in r else 0.0
        print(f'  {r.get("session_id")} conf={conf:.2f}{turn_s} "{snip}"')
    more = len(results) - DIGEST_WINDOW
    if more > 0:
        print(f"  (+{more} more — reformulate/multi-search or -n)")


def print_knowledge(knowledge: list) -> bool:
    """Emit knowledge distribution when above the report floor. Returns True if emitted."""
    hits = knowledge_hits(knowledge)
    if not hits:
        return False
    print(f"KNOWLEDGE {hits}")
    return True


def main() -> int:
    if len(sys.argv) > 2 or (len(sys.argv) == 2 and sys.argv[1] in ("-h", "--help")):
        print("usage: route.py [recall.json]  # else stdin", file=sys.stderr)
        return 2
    try:
        raw = open(sys.argv[1], encoding="utf-8").read() if len(sys.argv) == 2 else sys.stdin.read()
        data = json.loads(raw)
    except (OSError, json.JSONDecodeError) as e:
        print(f"SILENCE reason=parse_error:{e}", file=sys.stderr)
        return 2

    results = data.get("results") or []
    knowledge = data.get("knowledge") or []
    # top/gap decide the episode verdict; confidence is also emitted so the
    # agent can weigh seeds. mass stays inside (never a veto).
    kind, top, gap, _, reason = episode_verdict(results)

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
        shown = min(len(results), DIGEST_WINDOW)
        # Primary verdict on line 1; confidence available for optional weighing.
        print(
            f"INJECT top={top:.2f} gap={gap:.2f} {shown} seed candidates — "
            f"drill <sid> at t<n>, reformulate for more"
        )
        print_digest(data)
        # Inclusive: mixed questions can need HEAD prose *and* an episode.
        # Only report knowledge above the super-low floor (junk marker omit).
        print_knowledge(knowledge)
        return done(0)

    # Episode below floor. Report knowledge only when above the report floor;
    # otherwise machine silence (near-zero prose score is not a substrate).
    if print_knowledge(knowledge):
        return done(0)

    # Nothing on either substrate — machine silence. The reason is diagnostic;
    # the agent may still reformulate.
    print(f"SILENCE reason={reason or 'below_gate'}")
    return done(1)


if __name__ == "__main__":
    sys.exit(main())
