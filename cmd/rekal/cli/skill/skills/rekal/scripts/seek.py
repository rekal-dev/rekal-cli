#!/usr/bin/env python3
"""Multi-framing recall, fused — one seed from several phrasings.

`rekal-seek "framing A" "framing B" …` runs recall once per framing and
Reciprocal-Rank-Fusion-merges the candidate lists into one ranked seed. The
ledger indexes the words the *past* session used, not the words you asked
with, so a single phrasing is one guess; a session that surfaces under two
framings is almost always the answer. This is the ledger's "widen before you
conclude" move made a function: the agent supplies the framings (judgment
stays with the agent), the script does the deterministic fuse (toil leaves
the agent).

Output is the route.py INJECT digest over the fused list — the same
`sid conf=… t<turn> "snippet"` rows the agent already reads — with a header
that names the fusion. `conf` per session is the MAX confidence that session
earned across framings (its strongest phrasing), reported for the agent to
weigh; fusion only reorders candidates, it never gates.

RRF: score(d) := Σ_framings 1/(k + rank_f(d)), rank 1-indexed, k=60. k=60 is
the standard RRF constant (Cormack, Clarke & Büttcher, 2009) — a rank-
smoothing convention, not a number read off any corpus, and it decides nothing
(no silence rides on it). So it is not a tuned gate (SOUL.md: no tuned constant
decides).

  rekal-seek "token refresh expiry" "JWT expiry auth" "session timeout"

Exit: 0 fused seed printed, 1 no candidates under any framing, 2 usage/error.
"""
from __future__ import annotations

import json
import os
import signal
import subprocess
import sys

# Don't byte-compile the sibling import: a __pycache__ next to the scripts
# would be picked up by the Go `//go:embed all:skills` and break the build.
sys.dont_write_bytecode = True

try:
    signal.signal(signal.SIGPIPE, signal.SIG_DFL)
except (ImportError, AttributeError, ValueError):
    pass

try:
    import route  # sibling script — single source of truth for the digest format
except ImportError:  # pragma: no cover - scripts always ship together
    route = None

RRF_K = 60  # standard RRF smoothing constant; never a gate (see module docstring)


def _f(v, default: float = 0.0) -> float:
    try:
        return float(v)
    except (TypeError, ValueError):
        return default


def recall(bin_: str, framing: str) -> tuple[dict | None, str | None]:
    try:
        p = subprocess.run([bin_, framing], capture_output=True, text=True, timeout=120)
    except (OSError, subprocess.TimeoutExpired) as e:
        return None, f"seek: cannot run {bin_}: {e}"
    if p.returncode != 0:
        # Forward the engine's own words — a recall error is not an empty set.
        return None, (p.stderr or p.stdout or f"seek: rekal exit {p.returncode}").rstrip()
    try:
        return json.loads(p.stdout), None
    except json.JSONDecodeError as e:
        return None, f"seek: non-JSON recall for {framing!r}: {e}"


def print_digest(fused: list) -> None:
    """Prefer route.py's digest so the format stays identical to a plain
    INJECT; fall back to an inline copy only if the sibling import failed."""
    if route is not None:
        route.print_digest({"results": fused})
        return
    for r in fused[:20]:
        sid = r.get("sid") or r.get("session_id")
        snip = " ".join((r.get("snippet") or "").split()[:15])
        turn = r.get("snippet_turn_index")
        turn_s = f" t{turn}" if turn is not None else ""
        print(f'  {sid} conf={_f(r.get("confidence")):.2f}{turn_s} "{snip}"')
    more = len(fused) - 20
    if more > 0:
        print(f"  (+{more} more)")


def main() -> int:
    framings = [a for a in sys.argv[1:] if a not in ("-h", "--help")]
    if len(framings) != len(sys.argv) - 1 or not any(f.strip() for f in framings):
        print('usage: seek.py "<framing1>" "<framing2>" …  # ≥1 phrasing of the same question',
              file=sys.stderr)
        return 2

    bin_ = os.environ.get("REKAL_BIN", "rekal")
    rrf: dict[str, float] = {}       # session_id -> fused score
    best: dict[str, dict] = {}       # session_id -> max-confidence representative
    used = 0
    for framing in framings:
        if not framing.strip():
            continue
        data, err = recall(bin_, framing)
        if err:
            sys.stderr.write(err + "\n")
            return 2
        used += 1
        for rank, r in enumerate(data.get("results") or []):
            if not isinstance(r, dict):
                continue
            sid = r.get("session_id") or r.get("sid")
            if not sid:
                continue
            rrf[sid] = rrf.get(sid, 0.0) + 1.0 / (RRF_K + rank + 1)
            if sid not in best or _f(r.get("confidence")) > _f(best[sid].get("confidence")):
                best[sid] = r

    if not rrf:
        print(f"(no candidates under {used} framing(s) — reformulate with different "
              "words, or the ledger may not hold it)")
        return 1

    fused = [best[s] for s in sorted(rrf, key=lambda s: rrf[s], reverse=True)]
    top_conf = max(_f(r.get("confidence")) for r in fused)
    print(f"SEEK top={top_conf:.2f} {len(fused)} fused seeds ({used} framings, RRF)")
    print_digest(fused)
    return 0


if __name__ == "__main__":
    sys.exit(main())
