# The one paper: working backwards from the position

Decided: **one paper overall for Rekal.** It absorbs the substrate paper
("The Commit Is the Label", `paper/`), the full MRR/mechanism program
(the consolidated multi-corpus data pack: 8-corpus matrix, mechanism
sweeps, the SPM facet port), and the three-mode routing/sufficiency
results. Structure: **ladder body, waterfall ending** — each measurement
exposes the gap the next component closes, and the paper lands on
coverage-at-cost.

Corpora are referred to by workload class only, never by name: **K**
(knowledge/docs, n_test=212) and **P** (production-build, n_test=456)
are the two statistically valid corpora; six small corpora (code,
docs/architecture, notes, build-scripts, two mixed) are directional and
OVERFIT-flagged. The paper reports aggregates only; no corpus content or
identity leaves the operator's machine.

## 1. The position

> **Rekal is git-bound memory.** The boundary of the memory system is the
> boundary of the repository and its version control; inside that binding,
> ground truth, freshness, verification, and containment are inherited
> from git, not rebuilt in software. On the bound ledger, a tool + skill +
> router combination closes the gap between *findable* and *answered*:
> retrieval supplies seeds (solved, characterized, honestly bounded), a
> structural map answers breadth, decision synthesis answers
> rationale/multi-hop, a confidence gate keeps bad episodes out of
> context, and gated routing delivers full coverage of the question kinds
> at low expected cost.

Reader takeaway: *bind memory to git, route questions to modes, and an
agent answers "why is this code like this?" from its team's recorded
intent in a few hundred tokens — verifiably, on your own repo, because
the benchmark labels itself.*

## 2. The one honest sentence about retrieval

Not "MRR solves 50%" — the defensible form is:

> **Seed quality is a necessary half of every answer and a sufficient
> answer for one kind.** Every mode consumes seeds (the gate ranks
> episodes, the synthesis gather starts from retrieval, the map's
> clusters are seeded), so the seed stage is load-bearing for all three;
> alone it answers only pointed questions (single-shot sufficiency
> 0.07–0.20 overall).

The under-gathered-synthesis ablation (weak gather → INSUFFICIENT;
~30-turn decision-scoped gather → full arc) is the direct proof that bad
seeding starves even the strongest mode.

Two metrics, stated as a division of labor, not a hedge: **MRR is the
stage metric** (self-labeling makes it free and large-n on the valid
corpora, paired bootstrap CIs, mechanism ablations) — used where it is
the right instrument, seed supply. **Answer-sufficiency is the system
metric** (costs a judge) — used for the answer stage. "We measure each
stage with the cheapest honest instrument."

## 3. The ladder (paper body)

Abstract states the destination first (git-bound + gated routing + the
waterfall's last row); the body climbs:

**Rung 1 — Raw transcripts are not memory.** Two honest steps now, not
one: (a) grep over raw JSONL is a near-zero floor (0.005 MRR, the 37×/57×
result); (b) the stronger B1′ floor — term-frequency grep over the
*parsed* turns index, all 8 corpora — and the hybrid still dominates
everywhere (K: 0.021 → 0.225, ~11×; P: 0.028 → 0.159, ~6×). Parsing is
most of the miracle; ranking still beats grep on parsed turns on every
corpus. This kills the strawman-baseline critique preemptively.

**Rung 2 — The seed stage, closed.** The full mechanism study, presented
as a completed characterization, not a leaderboard:

- 8 corpora, 4 workload types; 2 statistically valid (K, P), 6 small and
  OVERFIT-flagged — reported directionally, flags in the table.
- The shipped formula (weighted BM25*+LSA+neural, per-role boosts,
  subagent discount, opt-in facet term; defaults 0.35/0.10/0.55, steer
  1.3, summary 1.15, subagent 0.7, facet 0).
- **The mechanism graveyard:** RRF, temporal decay, lexical dilution,
  z-normalization (even z-native re-tuned: +0.024, CI crosses 0), and
  embedder substitution (n.s. on hybrid) — all tested under the RHO
  incumbent-vs-candidate discipline, all rejected. The negative results
  are armor: every "did you try X" is pre-answered with a CI.
- **The two survivors:** per-corpus layer-mix + steering-boost tuning
  (significant on P +0.032 and the small code corpus +0.134), and the
  **SPM facet term** — the facet idea from SPM (arXiv 2607.09493) ported
  to a retrieval term: deterministic per-session facet doc (distinct
  tool paths + command prefixes + steering text, no LLM, index-time),
  BM25 over it as a 4th hybrid term. The first and only imported
  mechanism to survive end-to-end: the tuner picks facet_boost=0.3 on
  both valid corpora, marginal significant on both (K +0.110
  [+0.056,+0.173]; P +0.053 [+0.012,+0.107]). Ships off by default;
  auto-tuning selects it.
- **Generalization rules** (testable hypotheses keyed on measurable
  corpus characteristics): tool-diversity predicts facet lift (monotone:
  0.065 → +0.110 vs 0.028 → +0.053); commit-noise/prose predicts
  hybrid>BM25 (docs corpora +0.352*, code corpora tie); steering density
  (bimodal — only interactive corpora have steering turns) predicts
  steering-boost headroom; embedder and z-norm near-irrelevant because
  the engine already max-normalizes. This is the no-free-lunch verdict
  (ANMS) operationalized into predictors.

**The bridge sentence** (end of rung 2, the paper's hinge): *every pure
re-ranking mechanism failed; the only retrieval gain came from adding an
orthogonal evidence layer — "what tools did X use" lives in tool-call
metadata that conversational turns never mention. Gains come from
coverage of question kinds, not rank polish. The rest of the paper is
that lesson applied above retrieval.*

**Rung 3 — Findable is not answered.** Answer-sufficiency exposes what
rank cannot see: single-shot episodic sufficiency 0.07–0.20; breadth
questions have no single episode to rank.

**Rung 4 — The structural map** answers breadth: LLM-authored,
SHA-watermarked, regenerated by diffing changed clusters (inherits the
staleness answer — never a stale snapshot).

**Rung 5 — Decision synthesis** answers rationale/multi-hop: decision-
scoped gather (~30 turns / ~2.1k tokens) → one synthesis call → the arc
(design → rejected alternatives → forcing constraint → final rationale),
every claim carrying turn/commit pointers (inherits the
self-confirmation answer). Gather ablation as the mechanism proof.
Multi-hop is T4: self-labeled gold exists at n=92–104 pairs in each of
the three high-file-cooccurrence corpora — synthesis gets mined gold,
not just judged questions.

**Rung 6 — The gate.** Ungated episodes poison the map (0.63 → 0.29);
the top1–top2 gap signal suppresses 11/12 bad injections. Naive
context-stuffing is harmful; knowing when to stay silent is part of the
system.

**Rung 7 — Gated routing: full coverage at low cost.** The architecture
section: router classifies kind → dispatches mode → gate controls
injection → cost follows the question, not the corpus.

## 4. The waterfall (the paper's last table)

| Stage added | Kinds covered | Sufficiency | Cost/question |
|---|---|---|---|
| grep, raw JSONL | none | ~0 | — |
| grep, parsed turns (B1′) | weak pointed | low | unbounded scan |
| hybrid + 2 validated levers (incl. facet) | pointed; seeds for all modes | 0.17–0.50 pointed | ~1.5k drill |
| + structural map | breadth | 0.50–0.92 broad | amortized (SHA-diff regen) |
| + decision synthesis | rationale / multi-hop | 0.83 | ~2–3k |
| **gated routing over all** | **all kinds** | **0.43 / 0.60 overall** | **~400–1,000** |

With the wild-query kind distribution mined (`mine_wild.py` +
classifier), the last row upgrades to **expected cost**:
`E[tokens] = Σ p(kind) · cost(mode)` — a number requiring both the
routing architecture and real usage data; no compiled-store system can
produce it.

Baselines framing: B0 (no memory) and B2 (LLM-distilled MEMORY.md ≤8k
tokens — the folk practice) bracket the system for practitioners; B2 is
the comparison working engineers actually ask about. B6 (engine driven
by skill playbooks) is the shipped configuration.

## 5. Guarantees section (compressed substrate)

One section (~1.5 columns), early (§2 of the paper): the four
git-inherited answers — annotation (checkpoint link → self-labeling
benchmark), staleness (rebuild+diff; the map's SHA watermark is this
answer applied to mode structure), self-confirmation (merge gate;
synthesis evidence pointers are this answer applied to generated text),
contamination (review-only egress). By construction; cite the substrate
tech report for the long form. The committed paper is superseded for
numbers (one frozen snapshot) but stands as the long-form reference.

## 6. Landing order (hard dependency, unchanged)

1. Land the facet implementation branch (RHO-gated: done — significant
   on both valid corpora).
2. Recalibrate the confidence gate on the facet-enabled score
   distribution (top1–top2 gap thresholds shift when a 4th term enters
   the mix).
3. Freeze one snapshot; every number in the paper comes from it.

## 7. Evidence gaps before submission (updated)

| Gap | Status after data pack | Cost |
|---|---|---|
| Multi-corpus MRR at scale | **CLOSED** — 8 corpora, 2 valid, mechanism sweeps, CIs | — |
| Honest grep floor | **CLOSED** — B1′ parsed-turn grep, all corpora | — |
| Facet/mechanism ablations | **CLOSED** — graveyard + 2 survivors, RHO-disciplined | — |
| Synthesis gold at n | **PARTIAL** — T4 pairs mined (92–104 × 3 corpora); run them | small |
| Sufficiency n per corpus (now 12–15) | open — the flagship claim needs it | judging cost |
| ≥2 judge models + agreement | open | small |
| Wild-query kind distribution | open — unlocks expected-cost row and §1's taxonomy evidence | small, data exists |
| Gate recalibration post-facet | open — landing order §6 | small |
| Breadth gold miner | open — no T-family mines breadth; judged-only unless invented (doc-anchored questions?) | flag in paper's honesty budget |
| 50-pair label audit (06 §5) | open | small |

## 8. Title candidates

- *Git-Bound Memory: Routing Developer Questions to Structure, Episode,
  and Synthesis under a Token Budget*
- *Why the Code Is the Way It Is: Git-Bound Memory for Coding Agents*
- *The Answer, Not the Rank: Git-Bound Memory with a Question Router*

## 9. What this decides for the product

- **Rises:** confidence gate in recall output (engine; recalibrate after
  facets); decision-scoped gather primitive + synthesis skill with
  mandatory evidence pointers; structural map as SHA-watermarked
  regenerable artifact; router in the skill layer first (SHIPPED: one
  `rekal` skill with internal gated routing — MAP/MINE/HUNT/WHY); facet layer
  (SHIPPED: default facet_boost=0.3, guarded index, explicit 0 =
  byte-identical baseline); capture-gap measurement (verbalization rate).
- **Falls:** further ranking-mechanism work — the graveyard says the
  seed stage is closed; MRR becomes the regression diagnostic.
- **Unchanged:** the git binding (§5) — already built, already the moat.
