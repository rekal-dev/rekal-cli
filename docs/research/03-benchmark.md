# RekalBench v0 — benchmark specification

A reproducible, fully-local benchmark for **repo-grounded intent recall**.
No established benchmark exists for this domain (LoCoMo/LongMemEval are
dialogue-persona memory; BRIGHT/BEIR are document QA) — RekalBench is itself
part of the contribution.

Design principles: self-labeled ground truth (no annotation budget), every
baseline runnable by anyone, all data stays on the operator's machine, and
the harness reports the honesty caveats (label precision, leakage controls)
alongside the headline numbers.

## 1. Corpus

Built from a real machine's session store via `rekal index --include-all`
(see `04-data-plan.md`). Published as a **corpus card** (aggregate stats
only, no content): #repos, #sessions, #turns, #tool-calls, #steering turns,
#commits with linked sessions, date range, actor mix, per-repo distribution.

A benchmark run is keyed to a corpus snapshot (content hashes make this
stable); anyone can rebuild their own corpus and run the same harness.

## 2. Tasks

All ground truth is mined from structure Rekal already records. Target of a
query is always a session id (rung 1) or a source turn (rung 2).

### T1 — Provenance (commit → producing session)
- **Label:** `checkpoint_sessions` rows: commit `c` ↔ session set `S(c)`.
- **Query:** generated from the commit — message + changed file paths —
  paraphrased by an LLM into a natural question ("how did we implement the
  retry backoff for webhook delivery?").
- **Leakage control:** commit messages are *not* indexed content (turns_ft
  indexes session text), so lexical overlap is incidental, not circular.
  Paraphrase further de-correlates. Verify by measuring exact-string overlap
  between query and target snippet; report the distribution.
- **Metric:** MRR, Recall@{1,5,10} over sessions.

### T2 — Decision recall (question → steering turn)
- **Label:** a `human_steering` turn `t` in session `s` (the moments
  decisions actually got made).
- **Query:** LLM paraphrase of the *situation* the steering resolved,
  generated from ±3 turns of context **with the steering turn itself held
  out of the generation prompt's quoted text** (paraphrase-only, no verbatim
  spans; enforce by n-gram overlap filter ≤ 0.3 Jaccard on 4-grams).
- **Metric:** session-level MRR/Recall@5; turn-level hit if
  `snippet_turn_index` lands within ±2 of `t` (tests snippet quality, not
  just session ranking).

### T3 — Dead-end awareness (proposal → abandoned attempt)
- **Label:** sessions whose `git_branch` never reached the default branch
  (the merged-only gate's negative side — knowledge unique to Rekal).
- **Query:** "have we tried / should we do X?" where X paraphrases the
  abandoned branch's cumulative intent (from its human turns).
- **Metric:** Recall@5 of the abandoned-branch session; secondary: judged
  whether the returned snippet conveys *that it was abandoned*.

### T4 — Multi-hop synthesis (cross-session)
- **Label:** session pairs (s1, s2) linked by strong `file_cooccurrence` or
  lineage (`parent_session_id`), where answering requires both.
- **Query:** LLM-generated question whose answer needs facts from both
  (validated by the generator: answerable from {s1,s2}, not from either
  alone).
- **Metric:** both-in-top-k (k=10); rung-2 answer accuracy is the primary
  signal here. This is the MRAgent-motivated task: one-shot top-k should
  visibly struggle; skills-driven multi-step recall should not.

### T5 — Decision drift (latest decision wins)
- **Label:** decision pairs where a later session touching the same files
  reverses/supersedes an earlier one (candidate-mined by SQL: same file set,
  ≥2 sessions apart in time, steering turns in both; human-confirmed on a
  small sample — this is the only task needing light manual curation).
- **Query:** "what is our current approach to X?"
- **Metric:** top hit is the *later* decision; reversal mentioned in answer
  (rung 2). From BeliefShift; small N is fine (20–50 pairs).

## 3. Systems under test

| ID | System | How it runs |
|---|---|---|
| B0 | No memory | answering agent sees the question (+ repo code if the task allows) only — the floor |
| B1 | **DCI/grep** over raw `~/.claude/projects` JSONL | agent with rg/sed/jq in the transcripts dir; strong prompt (GrepSeek-informed), generous turn budget. The must-beat |
| B2 | Static notes | a MEMORY.md/CLAUDE.md distilled once from the corpus by an LLM (token-capped at 8k) — the folk practice |
| B3 | BM25-only | Rekal with weights `{bm25:1, lsa:0, nomic:0}` — ablation via `.rekal/config.json`, no code changes |
| B4 | Vector-only | weights `{bm25:0, lsa:0, nomic:1}` — ablation |
| B5 | **Rekal full** | default hybrid weights + steering boost + subagent down-weight |
| B6 | Rekal + skills | B5 driven by the skill playbooks (active reconstruction) — rungs 2–4 only |

Weight ablations B3/B4 are free: query-time weights mean no reindex between
systems. B1 must be run honestly (see 02 §9–10): good prompt, same model,
same turn budget as B6.

## 4. Metrics

- **Rung 1:** MRR, Recall@{1,5,10}, nDCG@10; per-task and pooled;
  bootstrap CIs (1k resamples over queries).
- **Rung 2:** answer accuracy by LLM judge (rubric: matches source turn's
  content; judge sees the gold turn), with a 50-sample human agreement check.
- **Rung 3:** context tokens loaded until first correct answer; wall-clock;
  $-cost at list prices. Report the full trade-off curve (accuracy vs tokens),
  RISE-style.
- **Rung 3 (judge-free drill-strategy proxy, runnable today):** on queries
  where the gold session reaches recall's top-k, compare the two drills an
  agent can make — a 5-turn raw window around the matched turn vs the single
  turn `summary_turn_index` points at — on tokens ingested (bytes/4) and
  gold-term coverage (fraction of the label's distinctive content words
  present in the drilled text). Paired: summary column only where a
  compaction summary exists. This measures context-assembly cost, not answer
  quality — never present it as rung 2. Implemented:
  `scripts/bench/run_rung3.py`.
- **L3 gate (wiki experiment):** both sides of the materialized-browsing-
  layer trade. Generation cost: turns/tool-calls per topic page from the
  ledger's own lineage (`SELECT ... FROM session_facets WHERE workflow_name
  = '<run>'` — the wiki playbook fans out one subagent per topic, and Rekal
  captures each transcript with `parent_session_id`/`workflow_name`), plus
  tokens ingested from harness accounting. Cache payoff: on broad (T1/T3-
  style) queries, answering from the page vs recall+drill, same
  tokens/coverage proxy as the drill-strategy rows. Maintenance price:
  regeneration diff rate — pages invalidated per week of new sessions (the
  B2 static-notes failure mode made continuous). Cross-repo contribution:
  A/B the generation mode (own-repo evidence vs explicit cross-repo mode)
  and report pages-with-foreign-citations plus the broad-query coverage
  delta — the value of machine-wide memory measured at its reviewed-egress
  gate. A cached in-index L3 layer is built only if this experiment says so
  (roadmap R11).
- **Scale sweep (C4/RISE reproduction):** rung-1 metrics and B1 latency at
  corpus subsets {10%, 25%, 50%, 100%} by capture date — where does grep
  degrade, where does Rekal hold?
- **Freshness:** rung-1 metrics bucketed by target-session age; index
  rebuild wall-clock vs corpus size.
- **Label quality:** precision of T1 labels on a 50-pair human audit
  (a commit's linked session actually discusses the commit's change).

## 5. Protocol

1. Freeze a corpus snapshot (record `index_state`, counts, content-hash
   roll-up).
2. Mine labels (SQL in `04`), generate queries (paraphrase model ≠ judge
   model ≠ answering model where feasible; record all model ids + prompts in
   the run manifest).
3. Split: 10% dev (prompt/weight tuning allowed), 90% test (one shot —
   RHO discipline: no peeking, changes accepted only on dev).
4. Run rung 1 for B1–B5 (B0 not applicable). Publish.
5. Run the judge-free rung-3 drill-strategy proxy (`run_rung3.py`) on the
   full query set — no extra models needed; fills the paper's drill-strategy
   table (`tab-drill`).
6. Run rung 2–3 (judged) on a 200-query stratified subset for B0–B6.
7. L3-gate wiki experiment (optional, after rung 1): generate K topic pages
   via the rekal-wiki workflow on a named `workflow_name`, pull generation
   cost from the ledger lineage, re-answer a broad-query subset from pages
   vs recall+drill with the rung-3 proxy, and record the regeneration diff
   rate over the following weeks. Fills the paper's `tab-wiki`.
8. Rung 4: 10–20 real tasks in the richest repo, A/B (B0 vs B1 vs B6);
   measure steering count, dead-end re-proposals, time-to-done.
9. Every run emits a manifest (corpus card, model ids, prompts, weights,
   metrics, CIs) into `docs/research/runs/` — the regression baseline for
   all future engine changes (RHO's incumbent-vs-candidate rule).

## 6. Harness (v0 = scripts, not product code)

- `scripts/bench/` (implemented; see its README for the runnable flow):
  `corpus_card.sh` (aggregate stats), `mine_labels.sh` (T1–T3 gold),
  `gen_queries.py` (LLM paraphrase + leakage filter + dev split),
  `run_rung1.py` (B5 hybrid, B3/B4 weight ablations, B1 grep-rank),
  `score.py` (MRR/Recall@k/nDCG with bootstrap CIs), `run_rung3.py`
  (judge-free drill-strategy proxy). T4/T5 miners and the judged rungs
  remain future work.
- No new `rekal` commands required for rung 1. Judged rung 3's token
  accounting needs the runner to count tokens it feeds the answering model —
  harness concern, not product.
- Product hooks that would help later (`05-roadmap.md`): a `--bench` JSON
  echo of per-layer scores, and opt-in query→drill logging (LRAT labels).

## 7. Known threats to validity (stated in any publication)

- **Label noise** (T1): mitigated by audit + reporting precision.
- **Query-generation leakage:** paraphrase + n-gram filter + report overlap.
- **Judge bias:** distinct models for generate/answer/judge; human agreement
  sample.
- **Single-corpus generalization:** harness is public; invite replication;
  run on ≥2 of the operator's repos and report per-repo variance.
- **B1 under-tuning:** publish B1's prompt; accept community-suggested
  improvements; re-run.
