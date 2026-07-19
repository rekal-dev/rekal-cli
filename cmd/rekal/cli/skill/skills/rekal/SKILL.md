---
name: rekal
description: >
  Use in a repo with Rekal initialized (.rekal/ exists). Rekal is memory of
  prior AI sessions — who changed what, why, and when. Before spending a token,
  decide WHERE the answer lives: TREE / KNOWLEDGE / LEDGER / MAP. Route to one
  substrate, act, and stay silent when memory is not the tool. Always pipe
  rekal through a script (route.py / view.py) so you ingest compressed data;
  the judgment is yours.
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
| Ledger | session intent | past | `rekal` → `route.py`, drill → `view.py` | why, tried, rejected |
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

## Silence — machine for episodes, yours for knowledge

Pipe recall through `route.py`. Treat its labels as a **recommendation**, not a
final call: the router is intentionally biased toward **more data than
decision**. It keeps a **super-low** floor on absolute `confidence` (saturating
BM25) — only empty / near-zero is machine-silenced — and never gates on
max-normalized `score` (junk tops out near 1.0 too). Substrates are
**inclusive**: an episode and a knowledge hit can both be real for a mixed
question. Line 1 is the primary label (`INJECT` / `KNOWLEDGE` / `SILENCE`);
when both fire you get `INJECT` plus a trailing `KNOWLEDGE` line. `INJECT`
carries `top=`/`gap=` and each seed as `sid conf=… t<n> "snippet"` — **you
judge** from confidence + content. Mass stays inside the script (never a veto).
A lexically thin dialogue hit still injects. No episode signal and no knowledge
→ `SILENCE`.

`KNOWLEDGE path=score …` is reported only above a **super-low** floor (~0.25);
junk marker scores are omitted. Above the floor, **you** judge the distribution:
clear leader (`x.md=0.93 y.md=0.60 …`) → Read `x.md`; flat cluster → stay
silent on prose. On a mixed `INJECT`+`KNOWLEDGE` output, combine both if needed.

**Pipe every skill rekal** — never read raw JSON:
- recall → `python3 "$ROOT/scripts/route.py"`
- `query --session` / SQL → `python3 "$ROOT/scripts/view.py"`

## Dispatch — route, then act

| The question is… | Do |
|---|---|
| Present prose / convention | `rekal "<q>" \| python3 "$ROOT/scripts/route.py"` → on `KNOWLEDGE` (alone or after `INJECT`), judge the `path=score` distribution; Read the clear leader's `path`@`lines` |
| Past episode / why / tried / rejected | same → on `INJECT`, `Read references/ledger.md`; drill with `rekal query --session … \| python3 "$ROOT/scripts/view.py"` |
| Temporal, complete-set, analytical, decision-arc, provenance | `Read references/ledger.md` — SQL via `rekal query … \| python3 "$ROOT/scripts/view.py"`; don't rank a set |
| Breadth / structure | `bash "$ROOT/scripts/map.sh" fresh` then `Read references/map.md` |
| Publish `docs/wiki/` | `bash "$ROOT/scripts/wiki-gate.sh"` then `Read references/wiki.md` |
| Flags, SQL, PATH, schema | `Read references/reference.md` |

The route returns data; you decide the move. Cite session / turn / commit with
every memory claim.

## Judgment — SOUL-aligned (agent, not the script)

Scripts recommend and compress. **You** judge. Keep it general:

- **Only what the ledger holds.** Do not invent, pad, or borrow world knowledge
  when the record is thin — say what is there, or stay silent.
- **Lists and arcs are complete-sets.** "What/which/how many/besides/goals/events"
  → enumerate across *all* sessions (SQL + page until empty). The first two
  hits are not the answer. Stop only when further pages add nothing new.
- **Name the thing the question asked.** Causes → event names; hobbies → each
  distinct activity; job arcs → every beat (left X, got Y, aspires Z) that the
  record actually states.
- **Time stays anchored.** Prefer the relative form the speaker used plus the
  session date ("last week before 2022-10-13"), not a rounded gloss
  ("early October"), unless the record itself gives a calendar day.
- **Attribute carefully.** Who did it, with whom, where — if the drill
  contradicts the question's framing, say so; don't force a fit.

## Semantic warming — retry, don't settle

A `SEMANTIC warming` note means the deep-semantic layer's model daemon is still
loading (first recall after `init`/`sync`). The results you got are keyword +
LSA only — usable, but not the full ranking. If the answer matters, **re-run the
same recall with exponential backoff** (2s, 4s, 8s) until the note is gone; the
daemon warms in a few seconds and the retry carries semantic scoring. If it's
still warming after ~three tries, proceed with what you have — the keyword layer
already stands on its own.
