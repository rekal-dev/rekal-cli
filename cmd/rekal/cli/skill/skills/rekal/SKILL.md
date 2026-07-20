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
     the ledger has content — go there.)
2. If now — **code** or **prose**?
   - **Code** (path/symbol, present tense) → **Tree**. Grep; do not recall.
   - **Prose** → **Knowledge**. Never invent an episode when HEAD prose answers.

grep for code that is · knowledge for prose that is · ledger for the why that was.

## Route output — recommendation, you judge

**Pipe every skill rekal** — never read raw JSON:
- recall → `python3 "$ROOT/scripts/route.py"`
- `query --session` / SQL → `python3 "$ROOT/scripts/view.py"` (pipe with `2>&1`
  so engine errors surface — an error is not an empty set)
- every mention of a term → `python3 "$ROOT/scripts/find.py" "<term>" [role]`

`init` also puts `rekal-route` / `rekal-view` / `rekal-find` on PATH
(`~/.local/bin`) — same scripts, no `$ROOT` boilerplate: `rekal "<q>" | rekal-route`.

`route.py` labels are recommendations, biased toward more data than decision:
only empty / near-zero absolute `confidence` is machine-silenced (never
max-normalized `score` — junk tops out near 1.0 too). Substrates are
inclusive — line 1 is the primary label; `INJECT` may carry a trailing
`KNOWLEDGE` line for a mixed question. You judge from confidence + content.

- `INJECT top=… gap=… N seeds` + rows `sid conf=… t<n> "snippet"` — seed
  context in rank order. Weigh `conf=`; drill `sid` at `t<n>` for detail;
  reformulate / multi-search when the window isn't enough. A lexically thin
  dialogue hit still injects.
- `KNOWLEDGE path=score …` — judge the distribution: clear leader → Read its
  `path` at HEAD; flat cluster → stay silent on prose. Near-zero scores are
  omitted, not judged.
- `SILENCE reason=…` — nothing on either substrate. Say so; don't pad with
  near-misses. You may still reformulate.

## Dispatch — route, then act

| The question is… | Do |
|---|---|
| Present prose / convention | `rekal "<q>" \| python3 "$ROOT/scripts/route.py"` → on `KNOWLEDGE` (alone or after `INJECT`), Read the clear leader's `path`@`lines` |
| Past episode / why / tried / rejected | same → on `INJECT`, `Read references/ledger.md`; drill with `rekal query --session … \| python3 "$ROOT/scripts/view.py"` |
| All / every / how many mentions of a thing | `python3 "$ROOT/scripts/find.py" "<term>"` — complete sweep in time order; then drill and judge (class-mapping, set size) |
| Temporal, analytical, decision-arc, provenance | `Read references/ledger.md` — SQL via `rekal query … \| python3 "$ROOT/scripts/view.py"`; don't rank a set |
| Breadth / structure | `bash "$ROOT/scripts/map.sh" fresh` then `Read references/map.md` |
| Publish `docs/wiki/` | `bash "$ROOT/scripts/wiki-gate.sh"` then `Read references/wiki.md` |
| Flags, SQL, PATH, schema | `Read references/reference.md` |

The route returns data; you decide the move. Cite session / turn / commit with
every memory claim.

## Judgment — agent, not the script

- **Only what the ledger holds.** Do not invent or pad. If the record is thin,
  say so — or stay silent.
- **A partial set is a wrong answer when the question asks for the set.**
  Enumerate (SQL, page until empty). Ranked recall is for pointed questions,
  not "all / which / how many / every beat of an arc."
- **Keep the speaker's precision.** Relative time and attribution stay as the
  record states them; don't round, force, or tidy away ambiguity.
- **A false premise has no answer.** When the question asserts something the
  record contradicts or never says, say that — never fabricate the asserted
  fact, and never silently answer a corrected question nobody asked.
- **Drill the hit before concluding absence.** A recalled seed you haven't
  drilled outranks any amount of tree-grepping; grep never answers a ledger
  question, and an empty stub greps forever.

## Semantic warming

A `SEMANTIC warming` note means the deep-semantic daemon is still loading;
those results are keyword + LSA only. If the answer matters, re-run the same
recall with exponential backoff (2s, 4s, 8s) until the note is gone; after
about three tries proceed — the keyword layer stands on its own.
