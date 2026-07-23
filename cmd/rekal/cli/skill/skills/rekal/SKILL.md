---
name: rekal
description: >
  Use in a repo with Rekal initialized (.rekal/ exists). Rekal is memory of
  prior AI sessions — who changed what, why, and when. Before spending a token,
  decide WHERE the answer lives: TREE / KNOWLEDGE / LEDGER / MAP. Route to one
  substrate, act, and stay silent when memory is not the tool. Rekal's commands
  return compact agent-readable text by default; the judgment is yours.
---

# Rekal — which substrate answers this?

Most wasted effort is the wrong substrate. Decide before you grep or recall.

| Substrate | Holds | Tense | Reach it with | Answers |
|---|---|---|---|---|
| Tree | current code | now | grep / read | what does X do, where is it |
| Knowledge | current prose | now | `rekal "<q>"` → Read HEAD | convention / what we know |
| Ledger | session intent | past | `rekal "<q>"`, drill `rekal query --session` | why, tried, rejected |
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

## Commands — text by default, `--json` for machines

Rekal's read commands print compact agent-readable text; add `--json` only when
a program needs to parse it.

- `rekal "<q>"` — recall. Prints a **seed digest**: line 1 is the verdict
  (`INJECT` / `KNOWLEDGE` / `SILENCE`), then per-seed `sid conf=… t<n> "snippet"`.
- `rekal "<q>" --also "<framing2>" --also "<framing3>"` — widen a weak recall:
  each `--also` is another phrasing of the *same* question, RRF-fused into one
  seed. Use when one phrasing wasn't enough before concluding.
- `rekal find "<term>" [role]` — every ledger mention of a term, complete and in
  time order (the "all / every / how many" sweep). A partial list is a wrong
  answer to a set question — this is the set.
- `rekal when <YYYY-MM-DD> "<phrase>"` — resolve a relative date ("last
  Saturday") from a mention's date; honest window for vague phrases.
- `rekal query --session <sid|ulid> [--offset N --limit 5 --role …]` — drill a
  session into readable turns. `rekal query --sql "SELECT …"` for analytical /
  complete-set SQL (see `references/reference.md` for the full schema; `ts` is a
  TIMESTAMP — use `BETWEEN`, not `LIKE`).

`INJECT`/`SILENCE` are **recommendations**, biased toward more data than
decision: only empty / near-zero absolute `confidence` is machine-silenced
(never max-normalized score — junk tops out near 1.0 too). Substrates are
inclusive — `INJECT` may carry a trailing `KNOWLEDGE` line. **You** judge from
`conf=` + content; a lexically thin dialogue hit still injects. On `KNOWLEDGE
path=score …` judge the distribution: clear leader → Read its `path` at HEAD;
flat cluster → stay silent on prose.

## Dispatch — route, then act

| The question is… | Do |
|---|---|
| Present prose / convention | `rekal "<q>"` → on `KNOWLEDGE`, Read the clear leader's `path`@`lines` |
| Past episode / why / tried / rejected | `rekal "<q>"` → on `INJECT`, `Read references/ledger.md`; drill `rekal query --session <sid> --offset <t-2> --limit 5` |
| Weak recall, one phrasing wasn't enough | `rekal "<q>" --also "<f2>" --also "<f3>"` — RRF-fuse framings |
| All / every / how many mentions of a thing | `rekal find "<term>"` — complete sweep; then drill and judge (class-mapping, set size) |
| Relative "when" (last Saturday, a month ago) | `rekal when <anchor-date> "<phrase>"` |
| Temporal, analytical, decision-arc, provenance | `Read references/ledger.md` — SQL via `rekal query --sql "…"`; don't rank a set |
| Breadth / structure | `bash scripts/map.sh fresh` then `Read references/map.md` |
| Publish `docs/wiki/` | `bash scripts/wiki-gate.sh` then `Read references/wiki.md` |
| Flags, SQL, PATH, schema | `Read references/reference.md` |

The command returns data; you decide the move. Cite session / turn / commit with
every memory claim.

## Judgment — agent, not the command

- **Only what the ledger holds.** Do not invent or pad. If the record is thin,
  say so — or stay silent.
- **A partial set is a wrong answer when the question asks for the set.** Use
  `rekal find` / SQL and page until empty. Ranked recall is for pointed
  questions, not "all / which / how many / every beat of an arc."
- **Keep the speaker's precision.** Relative time and attribution stay as the
  record states them; don't round, force, or tidy away ambiguity.
- **A false premise has no answer.** When the question asserts something the
  record contradicts or never says, say that — never fabricate the asserted
  fact, and never silently answer a corrected question nobody asked.
- **Drill the hit before concluding absence.** A recalled seed you haven't
  drilled outranks any amount of tree-grepping; grep never answers a ledger
  question, and an empty stub greps forever.

## Semantic warming

A `SEMANTIC warming` line means the deep-semantic daemon is still loading; those
results are keyword + LSA only. If the answer matters, re-run the same recall
with exponential backoff (2s, 4s, 8s) until it's gone; after ~three tries
proceed — the keyword layer stands on its own.
