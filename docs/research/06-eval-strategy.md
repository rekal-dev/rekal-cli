# Evaluation strategy: multi-repo at scale + Rekal in use

The single-corpus study in `03-benchmark.md` established retrievability on one
operator's primary repo (T1/T2, the 37× floor over raw grep, the localized
neural gain). This document is the next run's strategy. It changes three
things: **many repos, not one**; **T1–T4, with T5 deprioritized**; and a
second pillar the retrieval study does not have — **does using Rekal actually
help**, measured both observationally (free, at scale) and interventionally
(A/B, small N).

It supersedes, it does not append: when this run lands, its manifest replaces
`runs/2026-07-12/` as the canonical numbers and the paper cites one snapshot.
Never mix a T2 number from the old run with a T4 from the new one.

## 1. Corpus: every repo on the machine, at scale

Fold the whole machine with `rekal index --include-all` and freeze one
snapshot (content-hash roll-up). Repos play two roles:

- **Labeled repos** — those with a `.rekal/` checkpoint ledger (Rekal is
  actually installed). They supply gold: T1 (commit→session), T2 (steering
  turns), T3 (unmerged branches, where topology provides them), T4
  (co-occurrence / lineage pairs). These are also the repos where usage and
  effectiveness (§4) can be measured, because only they have the commit
  anchors.
- **Haystack repos** — imported, index-only, no checkpoints. They cannot be
  gold, but they enlarge the retrieval space as realistic distractors and
  carry the scale sweep.

This is the honest harmonisation of own-repo and machine-wide data: **gold
comes only from checkpoint-bearing repos; every query is scored against the
full machine-wide index as the haystack.** One index, labels from the subset
that has them, a harder and more realistic retrieval task than a single repo
in isolation.

Deliverables: a per-repo corpus card plus an aggregate; **per-repo variance
on every headline metric** — this is the multi-corpus generalization the
single-repo study explicitly lacks, and reporting the spread (not just the
pooled mean) is what answers the "case study until replicated" caveat.

## 2. Tasks: T1–T4, T5 opportunistic

- **T1 provenance, T2 decision recall** — unchanged, mined per labeled repo.
- **T3 dead-ends** — mined where a repo actually has unmerged/abandoned
  branches. Report *which* repos yield T3 and how many; expect zero from
  single-branch repos. T3's presence is a corpus property, not a harness
  setting.
- **T4 multi-hop** — the priority new miner (`scripts/bench/mine_t4.*`):
  session pairs linked by strong `file_cooccurrence` or `parent_session_id`
  lineage, LLM-generated questions the generator validates as answerable from
  *both* sessions and not from either alone. T4 is the most valuable addition:
  it is the task where one-shot top-k retrieval should visibly struggle and
  skills-driven multi-step recall (B6) should win — a stronger result than
  anything in the retrieval-only study.
- **T5 decision drift** — attempt the SQL candidate miner (same file set, ≥2
  sessions apart, steering in both), but treat it as **opportunistic, not a
  deliverable**. It needs a genuine reversal history and per-pair manual
  confirmation. If a repo yields a clean sample, report it; otherwise omit it
  honestly rather than manufacture it. We may not get T5, and that is fine.

## 3. Systems

B1 grep-rank, B3 BM25-only, B4 neural-only, B5 hybrid for retrieval (rung 1),
plus **B6 Rekal + skills** for the usage/effectiveness rungs, where the
active-reconstruction playbooks (search → facets → zoom → lineage → drill) are
the system under test rather than one-shot recall.

## 4. Does Rekal help? Two pillars

Retrievability proves memory is *findable*. This run adds whether it is
*useful*, at two levels of rigor.

### 4a. Usage mining — observational, free, at scale

Every repo where Rekal is installed is already an effectiveness dataset: the
ledger records the sessions where an agent *called* `rekal`. Mine them across
all labeled repos (raw SQL over the corpus, no new labeling):

- **Adoption / frequency** — `rekal` invocations per session and per task;
  how many distinct repos and actors use it at all.
- **Drill-through rate** — after a `rekal` search, does the agent follow with
  `rekal query --session <id>`? The follow-through (and whether it keeps
  reading) is an implicit relevance label (LRAT `12` in the literature map) —
  a free, large-N signal of whether recall returned something worth reading.
- **Steering delta (the natural experiment)** — compare sessions that used
  Rekal against matched sessions that did not (same repo, similar task
  shape), on `human_steering` turn count and on re-proposal of
  known-abandoned approaches. Observational and confounded — agents and tasks
  differ — but directional, at a scale no A/B can reach, and honest about
  being a correlation.
- **Provenance chains followed** — how often an agent walks
  artifact→commit→session→intent in practice (the `rekal-provenance`
  playbook), evidence the pointer structure gets used, not just shipped.

State the confounds plainly: this is a natural experiment, not a controlled
one. Its value is scale and zero cost, and it sets the priors the A/B tests
below confirm on small N.

### 4b. Effectiveness — interventional A/B, small N

On a curated set of real tasks drawn from the richest repos, run the same
agent **with Rekal (B6) vs without (B0) vs grep (B1)**, measuring: human
steering interventions, re-proposals of known-abandoned approaches,
time-to-done, tokens-to-done, and final diff quality. Expensive (needs a human
or scripted rater); scope to ~10–20 tasks per repo across two or three repos.
This is the gold-standard confirmation of whatever 4a's natural experiment
suggests.

## 5. Scale and validity

- **Scale sweep (RISE/C4)** — rung-1 metrics and B1 latency at 10/25/50/100%
  date-cut subsets of the machine-wide corpus; locate the grep-degradation
  crossover on real session data, now across repos.
- **Label audit (P8)** — finally do the 50-pair T1 precision hand-audit on the
  canonical snapshot; the self-labeling benchmark's validity is unestablished
  without it, and it is cheap.
- **RHO discipline** — dev-tune on the 10% split, score test once; if any
  weight or index changes, every number comes from the new snapshot; record
  weights, splits, and model ids in the manifest.
- **Honesty budget** — retrieval is a findability proxy; usage mining is
  observational; T3/T5 depend on corpus topology; single-machine still, until
  others run the public harness on their own stores.

## 6. What lands in the paper

One harmonised manifest supersedes `runs/2026-07-12/`. The retrieval tables
gain per-repo variance and T4; the paper gains an effectiveness section
(4a headline numbers, 4b A/B where available); T5 appears only if a clean
sample exists. The four-problems framing is unchanged — this run strengthens
the *annotation* demonstration (more repos, more tasks) and begins to put
numbers behind the claim that the memory is not just findable but used.
