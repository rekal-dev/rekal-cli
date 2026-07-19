---
name: rekal
description: >
  Use in a repo with Rekal initialized (.rekal/ exists). Rekal is memory of
  prior AI sessions — who changed what, why, and when. Before spending a token,
  decide WHERE the answer lives: TREE / KNOWLEDGE / LEDGER / MAP. Route to one
  substrate, act, and stay silent when memory is not the tool. The scripts
  return deterministic data; the judgment is yours.
---

# Rekal — which substrate answers this?

Most wasted effort is the wrong substrate. Decide before you grep or recall.

```bash
ROOT="${CLAUDE_SKILL_DIR:-$(git rev-parse --show-toplevel)/.claude/skills/rekal}"
```

| Substrate | Holds | Tense | Reach it with | Answers |
|---|---|---|---|---|
| Tree | current code | now | grep / read | what does X do, where is it |
| Knowledge | current prose | now | `rekal` → `route.py` → Read HEAD | convention / what we know |
| Ledger | session intent | past | `rekal` → `route.py`, then drill | why, tried, rejected |
| Map | structure | — | `map.sh` + workflow | how is it built |

## Boundary

1. Is it true **now**, or something that **was**?
   - **Was** — a reason, a rejected path, a past correction, or a fact whose only
     record is a past conversation → **Ledger**. (On a pure-dialogue corpus only
     the ledger has content — routing is trivial, go there.)
2. If now — **code** or **prose**?
   - **Code** (path/symbol, present tense) → **Tree**. Grep; do not recall.
   - **Prose** → **Knowledge**. Never invent an episode when HEAD prose answers.

grep for code that is · knowledge for prose that is · ledger for the why that was.

## Silence is a machine event

Pipe recall through `route.py` — it gates on absolute `confidence`, not the
max-normalized `score` that tops out near 1.0 for junk too. `INJECT` wins over a
non-empty knowledge block; `KNOWLEDGE` is the fallback when the episode gate
fails; no label → stay silent on memory. A confident `low_mass=true` hit is a
real dialogue-shaped match — trust it or widen, don't abstain on it. Near-misses
are noise.

## Dispatch — one route, then stop

| The question is… | Do |
|---|---|
| Present prose / convention | `rekal "<q>" \| python3 "$ROOT/scripts/route.py"` → on `KNOWLEDGE`, Read `path`@`lines`, **stop** |
| Past episode / why / tried / rejected | same pipeline → on `INJECT`, `Read references/ledger.md` and drill |
| Temporal, complete-set, analytical, decision-arc, provenance | `Read references/ledger.md` — decompose to SQL, enumerate, navigate by time; don't rank a set |
| Breadth / structure | `bash "$ROOT/scripts/map.sh" fresh` then `Read references/map.md` |
| Publish `docs/wiki/` | `bash "$ROOT/scripts/wiki-gate.sh"` then `Read references/wiki.md` |
| Flags, SQL, PATH, schema | `Read references/reference.md` |

One question, one substrate. The route returns data; you decide the move. Cite
session / turn / commit with every memory claim.
