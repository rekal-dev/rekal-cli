# Rekal Memory Research

**Goal:** position Rekal as the natural, *better* memory tool for AI coding
agents — with data, not adjectives — and adjust the product where the
literature shows a better design.

**Current state:** the program converged on one flagship paper — *"Why Git
Is the Memory Solution for the Agentic Development Lifecycle"*
(`paper/rekal-paper.typ`) — whose structure is derived in
`07-paper-restructure.md` (git-bound memory; two problems solved separately
— seed supply, answer assembly — then combined by a gated router; measured
by answer-sufficiency per token). The docs below are the working-backwards
chain that produced it; where a doc and the paper disagree, the paper wins.

| Doc | Question it answers |
|---|---|
| [01-positioning.md](01-positioning.md) | What exactly are we claiming, and what evidence would prove it? (the evidence ladder) |
| [02-literature-map.md](02-literature-map.md) | What do the papers say, and is each one support, threat, or something to steal? |
| [03-benchmark.md](03-benchmark.md) | RekalBench — tasks, metrics, baselines, protocol. |
| [04-data-plan.md](04-data-plan.md) | How to build the benchmark corpus from the local session store, with exact SQL. |
| [05-roadmap.md](05-roadmap.md) | Product adjustments derived from the literature. |
| [06-eval-strategy.md](06-eval-strategy.md) | Multi-repo-at-scale evaluation + the Rekal-usage/effectiveness pillar; the one-snapshot rule. |
| [07-paper-restructure.md](07-paper-restructure.md) | The flagship paper's working-backwards structure, evidence-gap table, and landing order (facets → gate recalibration → frozen snapshot). |
| [00-sources.md](00-sources.md) | The verified source list (arXiv links). |
| [RUN.md](RUN.md) | The full multi-repo run sequence and the paper's data pack. |
| [runs/](runs/) | Committed aggregate run records: `single-corpus/` (rung 1) and `consolidated/` (multi-corpus matrix, mechanism sweeps, facet, sufficiency — status TRANSCRIBED_PENDING_VERIFICATION). |
| [paper/](paper/) | The flagship paper (Typst source + PDF) and its provenance README. |

## The chain in one paragraph

The claim is the paper's title, argued then measured: git already runs the
ADLC's code, and — bound, not bolted on — it runs its memory too. Ground
truth is self-labeled (`checkpoint_sessions` links every commit to its
producing sessions), which makes the retrieval program free to run and
replicate. That program is now *closed*: honest grep floors, a mechanism
graveyard with two RHO-disciplined survivors (per-corpus tuning; the SPM
facet term — shipped default `facet_boost=0.3`), and the lesson that gains
come from orthogonal evidence layers, not rank polish. Above retrieval the
same lesson becomes the system: real questions split into broad / pointed /
why; a gated router (shipped as the single `rekal` skill: MAP / MINE / HUNT / WHY
workflows) answers each kind at 382–980 tokens per question, and decision
synthesis reconstructs the "why" arcs single-shot retrieval fragments. The
binding constraint that remains is capture.

## The three findings that shape everything here

1. **Freshness of derived structure is the field's #1 unsolved problem**
   (deferred by LLM-Wiki, AdaMem, MRAgent, SAG alike). Rekal solves it
   *structurally*: `data.db` is an append-only source of truth; `index.db`
   is disposable and rebuilt; the structural map is a SHA-watermarked
   function of the tree, refreshed by diff. Derived structure that can be
   thrown away cannot go stale.
2. **Findable is not answered.** Single-shot episodic recall answers
   0.07–0.20 of real developer questions (answer-sufficiency, blind judge);
   ungated episode injection *degrades* a good structural answer
   (0.63→0.29), and the confidence gate recovers most of the loss. Routing
   by question kind — map for breadth, gated episodes for pointed,
   synthesis for why — is where the value above ranking lives.
3. **Memory reuse without external verification poisons itself** (EDV's
   self-confirmation trap). Rekal's merged-only export gate is an *external*
   verification signal no dialogue-memory system has, and synthesis output
   carries turn+commit pointers for the same reason.

## Status

- [x] Literature read; positioning + evidence ladder defined
- [x] RekalBench spec + label miners (T1–T5); self-labeling proven
- [x] Rung 1 run (single corpus) — committed manifest
- [x] Multi-corpus retrieval matrix + mechanism sweeps + facet port (consolidated record; anonymized aggregates committed)
- [x] Mode/sufficiency runs (n=12–15, two corpora) — directional
- [x] Engine aligned to the paper: facet layer shipped (default 0.3); router shipped as the single `rekal` skill (MAP/MINE/HUNT/WHY, gated)
- [ ] Verify the transcribed consolidated record against the operator copy (flip its status field)
- [ ] Gate recalibration on the facet-enabled engine → freeze the snapshot all paper numbers cite
- [ ] Sufficiency at scale: n ≫ 15, second judge + agreement, T4 mined gold for synthesis
- [ ] Wild-query kind distribution → the expected-cost figure (waterfall's last row)
