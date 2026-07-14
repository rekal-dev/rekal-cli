# Paper restructure: working backwards from the position

Two drafts exist. "The Commit Is the Label" (committed, `paper/`) is the
substrate paper: four problems answered by git, a self-labeling benchmark,
a retrievability result. "Memory Is Routing, Not Ranking" (local,
adlc-memory) is the behavior paper: three modes, a router, answer-sufficiency,
the capture-gap finding. Both are true; neither is attractive alone. The
substrate paper proves the ledger is findable without showing why that
matters. The routing draft leads with a critique of a metric — insider
baseball — and buries its own strongest capability in the conclusion.

This document works backwards from the position to one flagship paper that
absorbs both, plus the facet/MRR data, into a single attractive claim.

## 1. The position

> **Rekal is git-bound memory.** The boundary of the memory system is the
> boundary of the repository and its version control. Inside that binding,
> the hard guarantees of memory — ground truth, freshness, verification,
> containment — are inherited from git, not rebuilt in software. On top of
> the bound ledger, a tool + skill + router combination closes the gap
> between *findable* and *answered*: a router classifies each developer
> question and dispatches it to the mode that can answer it — structural
> map for breadth, confidence-gated episodic recall for pointed lookups,
> decision synthesis for rationale — each returning an answer within a
> token budget, not a ranked list.

The reader's one-sentence takeaway, which every section must serve:

> *Bind memory to git and route questions to modes, and an agent answers
> "why is this code like this?" from its team's recorded intent in a few
> hundred tokens — verifiably, on your own repo, because the benchmark
> labels itself.*

## 2. Why the current drafts underperform

Diagnosis, so the restructure fixes causes and not symptoms:

1. **Negative thesis.** "Not Ranking" attacks MRR. Readers remember
   capabilities. The demonstrated capability — memory that reconstructs
   *why a system became what it is* — is the conclusion's last line
   instead of the title's first idea.
2. **Taxonomy-forward.** Three modes read as a classification scheme. The
   attractive object is the *router*: one tool, any question, an answer
   within budget. The modes are its internals.
3. **Token efficiency scattered.** The ~1,500-token drill (paper 1), the
   382–980-token routed answers vs ~3k synthesis vs a 66k-turn history
   (draft 2) — the economics appear as asides. Answer-sufficiency **per
   token** is the spine: it is the soul's own definition of memory
   (context assembly under a token budget) and the axis on which no
   compiled-store competitor can follow.
4. **Substrate divorced from payoff.** Paper 1 has guarantees without
   answers; draft 2 compresses the guarantees into citations. "Git-bound"
   fuses them: the binding is *why* labels, freshness, verification, and
   containment are free — and why the modes can exist at all (the map is
   SHA-watermarked, episodes are commit-anchored, synthesis pulls diffs
   from git on demand).

## 3. The restructured paper

Working title candidates (positive, capability-first):

- *Git-Bound Memory: Routing Developer Questions to Structure, Episode,
  and Synthesis under a Token Budget*
- *Why the Code Is the Way It Is: Git-Bound Memory for Coding Agents*
- *The Answer, Not the Rank: Git-Bound Memory with a Question Router*

### Skeleton, with every existing asset slotted

**§1 The question an agent actually asks.** Open with the worked example
(paper 1 §2, kept — it is the best writing in either draft) but re-aim it:
the Friday agent doesn't need a ranked list, it needs the Monday answer
inside its context budget. State the three question kinds immediately as
*observed*, not asserted: breadth / pointed / rationale, with the
wild-query kind distribution (gap: §5) as evidence the taxonomy is real.

**§2 Git-bound: the guarantees are inherited.** Paper 1's four-problem
table compressed to one section (~1.5 columns). Annotation, staleness,
self-confirmation, contamination — each one paragraph, each ending with
the mechanism (checkpoint link, rebuild+diff, merge gate, review egress).
No empirical defense here; these are by-construction and say so. This is
the substrate paper absorbed as background — cite the tech report for the
long form.

**§3 The engine is a seed supplier.** One column. Ledger (`data.db`),
disposable index, hybrid retrieval + the facet layer. The MRR program
lands here in exactly two results: (a) the 37×/57× floor — raw
transcripts are not memory, parsing is what makes history searchable;
(b) facets — the one imported ranking mechanism with a significant
held-out MRR gain, and only on diverse-tooling corpora. Then the pivot
sentence the whole paper turns on: *ranking is solved-enough as seed
supply; single-shot answer-sufficiency is 0.07–0.20 — findable is not
answered.* The full MRR tables (B1–B5, T1/T2, facet ablation) move to an
appendix / the public run manifest.

**§4 The router and the three modes.** The system section, now the star.
Router (question kind → mode), then per mode: structural map
(SHA-watermarked, regenerated by diffing changed clusters — inherits §2's
staleness answer), gated episodic recall (top1–top2 gap as the confidence
signal; injection only above the bar), decision synthesis
(decision-scoped gather ~30 turns / ~2.1k tokens → one synthesis call →
the arc: original design, rejected alternatives, forcing constraint,
final rationale — every claim carrying turn/commit pointers, inheriting
§2's self-confirmation answer). The skills are named as what they are:
the routing policy is a skill driving engine primitives — the tool +
skill + router combination is the architecture, and it is legible because
each layer is inspectable text.

**§5 Evaluation: answer-sufficiency per token.** Headline metric:
sufficiency (SUFFICIENT/PARTIAL/INSUFFICIENT, blind judge) *with tokens
alongside every number*. The current draft's tables restructured so cost
is a first-class column, because the claim is economic:

- single-shot episodic: 0.07–0.20 sufficiency — the gap, stated once.
- routed: 0.43 / 0.60 overall at ~400–1,000 tokens per question — the
  per-kind floor at the cheap price.
- synthesis: 0.83 on the young corpus, and the rationale ablation (only
  an adequately-gathered synthesis answers the execution-engine "why";
  the 4-term gather starves it) — the capability result, at ~3k tokens.
- the gate ablation: ungated episodes poison the map (0.63 → 0.29);
  gating suppresses 11/12 bad injections — naive context-stuffing is
  *harmful*, knowing when to stay silent is part of the product.

Then the in-the-wild section (06-eval-strategy §4c/4d): real queries,
drill-through as implicit labels, cross-repo drill rate, and the
question-kind distribution of real usage.

**§6 The residual is capture.** Draft 2's §6 kept nearly verbatim — it is
the honest frontier and reads as strength: every non-routing failure was
an answer never verbalized; thin capture is the correct division of labor
(intent in the ledger, content in git, diffs pulled by SHA on demand).

**§7 Related work / §8 Conclusion.** The three departures survive, but
reordered to match the new spine: (i) guarantees inherited from git;
(ii) the router over three modes as the architecture that closes
findable→answered; (iii) answer-sufficiency per token as the measure —
and the last line is the capability: memory can reconstruct why a system
became what it is.

### What happens to the committed paper

"The Commit Is the Label" is not withdrawn or retconned — it becomes the
substrate tech report the flagship cites for the long-form guarantees and
the benchmark construction. Its numbers are superseded per the standing
snapshot rule (06 §0): the flagship cites one frozen post-facet manifest.

## 4. Landing order (hard dependency)

The confidence gate is calibrated on score distributions; the facet layer
changes score composition. Therefore:

1. Land `apple-facets` (RHO-gated: it beat the incumbent held-out — done).
2. Recalibrate the gate (top1–top2 gap thresholds) on the facet-enabled
   engine.
3. Freeze one snapshot; every number in the paper comes from it. Never
   mix pre- and post-facet numbers.

## 5. Evidence gaps before submission

What "we have everything" still needs, in cost order:

| Gap | Why it's needed | Cost |
|---|---|---|
| n per corpus well above 12–15 | current CIs span .20–.67; "directional" won't carry the flagship claim | question mining is cheap; judging is the cost |
| ≥2 judge models, agreement reported | single blind judge is the draft's weakest boundary | small |
| Wild-query kind distribution (`mine_wild.py` + classifier) | converts the breadth/pointed/rationale taxonomy from asserted to measured — §1's opening evidence | small, data already exists |
| ≥3 labeled corpora with per-repo variance (06 §1) | "each mode wins a different kind" needs to survive repo diversity | medium |
| Gate recalibration post-facets | §4 landing order | small |
| 50-pair label audit (06 §5) | benchmark validity, still unestablished | small |

## 6. What this decides for the product

The paper's spine is the roadmap's spine:

- **Rises:** confidence gate in recall output (engine); decision-scoped
  gather primitive + synthesis skill with mandatory evidence pointers;
  structural map as SHA-watermarked regenerable artifact (extends
  rekal-wiki/distill); router in the skill layer first, engine later;
  capture-gap measurement (verbalization rate).
- **Falls:** further ranking-weight work (R1) — facets were the last
  ranking investment the data justifies; MRR becomes a regression
  diagnostic, not a target.
- **Unchanged:** everything in §2 — the git binding is the moat and it is
  already built.
