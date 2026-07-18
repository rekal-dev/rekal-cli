---
name: rekal
description: >
  Use in a repo with Rekal initialized (.rekal/ exists). Rekal is memory of
  prior AI sessions — who changed what, why, and when. Before spending a token,
  decide WHERE the answer lives: TREE / KNOWLEDGE / MAP / LEDGER — and for the
  ledger, which PROFILE (coding defaults vs chat gates for dialogue haystacks).
  Classify, dispatch to one substrate and one profile, stay silent when memory
  is not the tool. On analytical asks, decompose to SQL before hybrid search.
  Route, do not stack.
---

# Rekal — which substrate answers this?

Most wasted effort is the wrong substrate. Decide before you grep or recall.

```bash
ROOT="${CLAUDE_SKILL_DIR:-$(git rev-parse --show-toplevel)/.claude/skills/rekal}"
```

| Substrate | Holds | Tense | Primitive | Answers |
|---|---|---|---|---|
| Tree | current code | now | grep / read | what does X do, where is it |
| Knowledge | current prose | now | `rekal` → route script → Read | convention / what we know |
| Map | structure | — | map scripts + workflow | how is it built |
| Ledger | session intent | past | gated `rekal` + drill | why, tried, rejected |

## Boundary

1. True **now**, or something that **was**?
   - **Was** (reason, rejected, past correction) → Ledger.
2. If now — **code** or **prose**?
   - **Code** (path/symbol, present tense) → Tree. Do not recall.
   - **Prose** → Knowledge. Never invent episodes when HEAD prose answers.

grep for code that is · knowledge for prose that is · ledger for the why that was.

## Profile — coding ledger vs chat ledger (before you hunt)

Shipped gate defaults are for **coding** sessions (high BM25 mass). Pure
dialogue / personal-memory corpora score real hits at conf≈0.5–0.6 with low
mass — the coding bars return `SILENCE below_mass` / `below_gate` even when
SQL would answer in one query. Pick a profile **before** the first recall:

| Ledger shape | Profile | Do once per shell |
|---|---|---|
| This repo's ADLC / code sessions (default) | `coding` | nothing — shipped defaults |
| Chat, LoCoMo-like, personal dialogue, low-mass haystack | `chat` | export gates + prefer chat weights (below) |
| Multi-hop synthesis / WHY over chat | `multi-hop` | same gates as chat, different weights |

```bash
# chat / dialogue corpus — required before HUNT on that ledger
eval "$(python3 "$ROOT/scripts/calibrate-recall.py" --profile chat --print-env 2>&1 >/dev/null | grep '^export')"
W=$(python3 "$ROOT/scripts/calibrate-recall.py" --profile chat --print-cli 2>/dev/null)
# then: rekal --weights "$W" "<q>" | python3 "$ROOT/scripts/recall-route.py"
```

Signals you are on a chat ledger: questions about people/preferences/dates in
conversation; prior `SILENCE` with `below_mass` while `ILIKE` finds the turn;
industry-bench / imported dialogue, not commit-linked code intent.

If the first route returns `SILENCE` with `below_mass` / `below_gate` and the
ask is chat-shaped, **switch to the chat profile and re-query once** before
abstaining — that is a wrong bar, not absence. Sticky `--apply` only when this
repo *is* a chat corpus (see `references/calibrate.md`).

## Silence is a machine event

Do not invent thresholds. Pipe recall through the route script — it gates on
absolute `confidence` (and `mass`), not max-normalized `score`. `INJECT` wins
over a non-empty knowledge block; `KNOWLEDGE` is the fallback when the episode
gate fails. No `INJECT` / `KNOWLEDGE` → stay silent on memory — **except** the
chat-profile retry above when `below_mass` / `below_gate` on dialogue. Near-misses
are noise.

## Dispatch — one Read or one script, then stop

| The question is… | Do |
|---|---|
| Present prose / convention | `rekal "<q>" \| python3 "$ROOT/scripts/recall-route.py"` → on `KNOWLEDGE`, Read `path`@`lines`, **stop** |
| Pointed past episode (coding) | same pipeline → on `INJECT`, `Read references/hunt.md` and drill; on `KNOWLEDGE`/`SILENCE`, stop |
| Pointed past episode (chat / dialogue) | **chat profile first** (above); then hunt.md — reformulate / multi-lookup, time-axis + SQL enumerate; do not stop on coding-bar SILENCE |
| Temporal / ordering / complete-set past | same pipeline; then hunt.md §2b time-axis + enumerate — SQL over `ts`, count, page |
| Judgment over past facts ("would X enjoy", "does X's shop…") | hunt.md §2d — gather premises from the ledger, then infer; silence only when premises are missing |
| Premise smells wrong (subject/event/time mismatch) | hunt.md §2c — verify against the turn; false premise → say the ledger doesn't hold it, never correct-and-answer |
| Why / decision arc | `Read references/why.md` · gate gather with `scripts/why-trail-gate.py` |
| Analytical patterns | `Read references/mine.md` |
| File / line / commit archaeology | `Read references/provenance.md` |
| Breadth / shape | `bash "$ROOT/scripts/map-fresh.sh"` then `Read references/map.md` |
| Rules / libraries / census | `Read references/analytics.md` |
| Publish `docs/wiki/` | `bash "$ROOT/scripts/wiki-branch-gate.sh"` then `Read references/wiki.md` |
| Recall wrong / calibrate local profile | `Read references/calibrate.md` · `scripts/calibrate-recall.py` |
| Flags, SQL, PATH, schema | `Read references/reference.md` |

One question, one substrate, one profile. Cite session / turn / commit with every memory claim.
