# Search benchmarking — data-driven recall quality

**Status:** design (2026-07). Not yet implemented.

## Problem

Recall quality is unmeasured. The layer weights (BM25 0.35 / LSA 0.10 /
nomic 0.55, steering boost, subagent discount) are folklore — plausible, never
validated against a real corpus. We just made them configurable
(`.rekal/config.json`), which created a tuning loop with **no metric to tune
against**. The skill's guidance to agents ("search first, drill down
progressively") is likewise untested against how agents actually behave.

Quality of recall is the product. It has to become a number.

## The insight: the ledger is already a labeled dataset

Search benchmarks normally die on annotation cost. Rekal doesn't have that
problem, because its own data model contains query→relevance pairs:

> **Every checkpoint links a commit to the sessions that produced it.**

- **Query:** the commit subject line (`git show -s --format=%s <git_sha>`) — a
  human-written, one-line summary of the intent.
- **Ground truth:** the sessions joined via `checkpoint_sessions` — the
  conversations that actually produced that commit.

No annotation, no leakage (commit messages are not indexed in `turns_ft`),
and it scales with the corpus: a machine with years of history is a large
eval set for free. A developer's "massive amount of data on other machines"
is exactly the input this needs.

A second free source — use a held-out `human_steering` turn as the query and
its session as truth — is deferred: the turn text is *in* the index, so BM25
trivially finds itself unless the turn is excluded at index time (a hold-out
rebuild). Phase 2.

## `rekal bench`

A local, offline command. Reads the existing index; writes nothing.

```
rekal bench                # evaluate current config on this repo's corpus
rekal bench --sweep        # grid-search layer weights, report the best mixes
rekal bench -n 200         # cap the query count (default: all usable checkpoints)
```

1. **Build the query set** — every checkpoint whose commit subject is usable
   (non-empty, not a merge/revert boilerplate subject, ≥ 3 words) paired with
   its linked session IDs. Skip checkpoints whose sessions aren't in the
   index.
2. **Run recall** for each query through the real search path (same code as
   `rekal "query"` — no parallel implementation to drift) under one or more
   weight configurations.
3. **Score** — for each query, the rank of the first ground-truth session:
   - **MRR** (mean reciprocal rank) — the headline number
   - **Recall@1 / @5 / @10** — did truth appear in the top k
4. **Report** per configuration, JSON to stdout (agent first) with a human
   table on stderr:
   - `bm25-only`, `2-way (bm25+lsa)`, `3-way current config` — so each
     layer's marginal contribution is visible, answering "is nomic worth its
     cost on *this* corpus?"
   - `--sweep`: coarse grid over the layer simplex (e.g. steps of 0.1) plus
     steering boost {1.0, 1.3, 1.6} — feasible because weights are query-time
     (no reindex per configuration; one pass computes per-layer scores once
     and re-mixes them). Report the top mixes with their MRR, and print the
     `config.json` weights block ready to paste.

The output makes weight changes falsifiable: change config, run bench, the
number moves or it doesn't.

### Soul check

Local-only (reads the local index, never the network). Agent-first (JSON).
Zero config (labels come from the ledger). Transparent (the metric and the
query set are inspectable — `--dump-queries` prints the pairs). It adds one
command, no new storage.

## Data-driven skill and agent ergonomics

The same dogfooding move works for *usage*: Rekal captures sessions in which
agents ran `rekal` itself — `tool_calls` rows whose `cmd_prefix` starts with
`rekal `. Mining them (a bench subreport, or plain `rekal query` SQL) yields
the real query distribution:

- **Zero-result rate** and the queries that produced it → the strongest
  signal for skill fixes (query phrasing guidance, when to suggest
  `--include-all`) and for product fixes (fallback to semantic-only when BM25
  finds nothing).
- **Drill-down behavior** — do agents use `snippet_turn_index` + `--offset`
  windows as the skill teaches, or reach straight for `--full`? If the
  latter, the skill's progressive-loading section isn't landing; fix the
  wording or the defaults.
- **Filter usage** — which filters agents actually pass, informing what stays
  on the CLI surface.

Skill edits then cite measurements, not taste.

## Candidate signals the bench should adjudicate

Ideas that are plausible but must earn their place with an MRR delta before
shipping as defaults:

- **Recency weighting** — today a two-year-old session ranks equal to
  yesterday's at equal textual relevance. A gentle exponential decay
  (configurable half-life in `weights`) is the classic missing signal.
- **Steering boost value** — 1.3 is a guess; the sweep measures it.
- **Origin discount** — should cross-repo (`origin`-labeled) hits rank below
  same-repo hits at equal relevance? Probably; measure it.

## Implementation sketch

1. `search`: expose a bench-friendly entry that returns per-layer scores
   before mixing (the sweep re-mixes without re-searching).
2. `bench` command: query-set builder (checkpoints + `git show`), runner,
   MRR/Recall@k scoring, JSON + table output, `--sweep`, `--dump-queries`.
3. Usage-mining subreport over `tool_calls` (`cmd_prefix LIKE 'rekal %'`).
4. Docs: `docs/spec/command/bench.md`, README one-liner, CLAUDE.md.
5. Phase 2: hold-out steering-turn eval (requires an index rebuild that
   excludes sampled turns); recency/origin signals behind config, adopted as
   defaults only on measured wins.
