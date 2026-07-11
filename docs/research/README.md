# Rekal Memory Research

**Goal:** position Rekal as the natural, *better* memory tool for AI coding
agents — with data, not adjectives — and adjust the product where the
literature shows a better design.

This folder works backwards from that goal. Read in order:

| Doc | Question it answers |
|---|---|
| [01-positioning.md](01-positioning.md) | What exactly are we claiming, and what evidence would prove it? (the evidence ladder) |
| [02-literature-map.md](02-literature-map.md) | What do the 17 papers say, and is each one support, threat, or something to steal? |
| [03-benchmark.md](03-benchmark.md) | RekalBench — tasks, metrics, baselines, protocol. What we run to get the numbers. |
| [04-data-plan.md](04-data-plan.md) | How to get the data — building the benchmark corpus from the massive local session store, with exact SQL. |
| [05-roadmap.md](05-roadmap.md) | Product adjustments derived from the literature ("make the best memory tool"). |
| [00-sources.md](00-sources.md) | The verified source list (all 17 papers, arXiv links). |

## The working-backwards chain in one paragraph

The end state is a public claim — *"an agent with Rekal answers 'why is this
code like this / what did we already try' correctly, at a fraction of the
tokens, with zero maintenance"* — backed by a reproducible benchmark. To make
that claim we need four rungs of evidence (retrievability → answer quality →
token efficiency → agent-in-the-loop A/B), each defined in `01`. To produce
those numbers we need a benchmark whose ground truth is **self-labeled** —
Rekal's own `checkpoint_sessions` table links every commit to the sessions
that produced it, which is free, abundant, objective supervision (`03`). To
build that benchmark we need a corpus — the operator's local store (hundreds
of sessions, tens of thousands of turns across repos) folded in via
`rekal index --include-all`, extracted with the SQL in `04`. And the same
literature that frames the evaluation also tells us where to improve the
product, which is `05`.

## The three findings that shape everything here

1. **Freshness of derived structure is the field's #1 unsolved problem**
   (deferred by LLM-Wiki, AdaMem, MRAgent, SAG alike). Rekal solves it
   *structurally*: `data.db` is an append-only source of truth; `index.db` is
   disposable and rebuilt, never migrated. Derived structure that can be
   thrown away cannot go stale. This is the positioning centerpiece.
2. **Unbounded grep does not scale** (RISE: at 1M docs, direct-corpus-
   interaction accuracy falls to 60% with wall-clock failures at ~$1.10/query;
   a bounded BM25-constructed workspace holds 78% at $0.28). Rekal is exactly
   a bounded-interaction-space constructor for intent history: hybrid search
   bounds, `rekal query` drills. This answers the strongest objection
   ("agents can already grep the transcripts").
3. **Memory reuse without external verification poisons itself** (EDV's
   self-confirmation trap). Rekal's merged-only export gate is an *external*
   verification signal no dialogue-memory system has: team memory admits only
   experience whose code actually landed on main.

## Status

- [x] Literature read (8 core deep, 9 supporting at abstract depth; all links
      verified via live search July 2026)
- [x] Positioning + evidence ladder defined
- [x] Benchmark spec (RekalBench v0)
- [x] Data-extraction plan keyed to the operator's local store
- [ ] Run corpus extraction on the operator's machine (`04-data-plan.md` §2)
- [ ] Run RekalBench v0 rungs 1–2, publish corpus card + numbers
- [ ] Rung 3 (token efficiency) and rung 4 (agent A/B)
