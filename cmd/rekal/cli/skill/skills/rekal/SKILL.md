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
  One call already **widens itself** — it fuses several deterministic
  reformulations of your query (keyword-only, clause splits, a temporal variant)
  so you get the full picture in one go; still reformulate *by hand* only when
  the answer needs a genuinely different angle the mechanical variants miss.
  A seed may carry `[reached N× drilled M×· "past query"]` before its snippet —
  a **usage** hint. `reached` counts how often the search surfaced it, which is
  the engine quoting itself: on a small store nearly everything is reached, so
  a bare high count means little. `drilled` counts how often an agent opened
  it — that is the load-bearing signal and a good first drill. The echoed query
  is the one that most often surfaced this memory, so it shows how the need is
  usually framed. Neither raises `conf=` — judge relevance from `conf=` +
  content as always. No tag just means newly surfaced, not worse.
- `rekal find "<term>" [role]` — every ledger mention of a term, complete and in
  time order (the "all / every / how many" sweep). A partial list is a wrong
  answer to a set question — this is the set.
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
| Weak recall (one call already fused reformulations) | re-search a genuinely different angle — synonyms, entity/path anchor, a re-split of a multi-hop question |
| All / every / how many mentions of a thing | `rekal find "<term>"` — complete sweep; then drill and judge (class-mapping, set size) |
| Relative "when" (last Saturday, a month ago) | ledger → classify at the workflow gate below (event-time) |
| Temporal, analytical, decision-arc, provenance | `Read references/ledger.md` — SQL via `rekal query --sql "…"`; don't rank a set |
| Breadth / structure | `bash scripts/map.sh fresh` then `Read references/map.md` |
| Publish `docs/wiki/` | `bash scripts/wiki-gate.sh` then `Read references/wiki.md` |
| Flags, SQL, PATH, schema | `Read references/reference.md` |

The command returns data; you decide the move. Cite session / turn / commit with
every memory claim.

## Ledger workflow gate

For a question routed to the ledger, classify the answer type before searching.
Choose the first matching row and read exactly that workflow. Do not blend
several workflows: concentrated guidance is more reliable than a pile of
partially relevant checks.

1. Elapsed time or duration between endpoints → Read `references/workflows/duration.md`
2. A count, set, plural list, repeated events, or ordered history → Read `references/workflows/complete-set.md`
3. A calendar time/date or temporal relation → Read `references/workflows/event-time.md`
4. A qualified prediction, likelihood, possibility, or inference → Read `references/workflows/inference.md`
5. A fact, episode, explanation, provenance, reflection, or other ledger answer → Read `references/workflows/point-fact.md`

Classify by the form of the answer requested, not by incidental words: "Which
events happened before June?" asks for a set, while "When did the event happen?"
asks for event time. The workflow supplies evidence invariants and useful
operations, never truth. The ledger remains authoritative; preserve genuine
ambiguity and reject unsupported premises.

### Final answer check

Before answering, silently compare the candidate answer with the requested
actor, entity, relation, time scope, and answer type.

- Reject another speaker's fact, a nearby semantic slot, an adjacent event, or a
  suggestion or plan mistaken for a completed event.
- When event time is requested, resolve a source-relative expression against the
  historical assertion timestamp. A relative expression in the question is
  anchored to the asker's present. Preserve source precision.
- For a count or set, ensure members were enumerated across the requested scope,
  class-mapped when necessary, and deduplicated.
- Before answering "unknown," make one focused reformulation only when retrieved
  evidence signals that the exact fact may be buried.
- If a check fails, repair evidence gathering rather than weakening the evidence
  standard or satisfying a false premise.

## Judgment — agent, not the command

- **Only what the ledger holds.** Do not invent or pad. If the record is thin,
  say so — or stay silent.
- **A partial set is a wrong answer when the question asks for the set.** Use
  `rekal find` / SQL and page until empty. Ranked recall is for pointed
  questions, not "all / which / how many / every beat of an arc."
- **Keep the record's precision.** Month-only evidence supports a month, not an
  invented day; attribution stays as the record states it; don't fake precision
  the record lacks or tidy away genuine ambiguity. (A resolvable source-relative
  phrase is still converted — see the event-time workflow.)
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
