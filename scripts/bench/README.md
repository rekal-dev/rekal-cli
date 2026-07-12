# RekalBench harness (v0 — rung 1, plus a rung-3 proxy)

Implements the retrievability rung of `docs/research/03-benchmark.md`:
self-labeled queries from your own corpus, run against Rekal (full hybrid +
single-signal ablations) and a grep-rank baseline over raw transcripts, then
scored (MRR / Recall@k / nDCG with bootstrap CIs). Everything runs locally;
nothing but aggregate numbers is meant to leave the machine.

`run_rung3.py` adds a tokens-to-context PROXY for rung 3: after recall lands
on the gold session, it compares the raw-window drill against the
summary-first drill (via the per-result `summary_turn_index` pointer) on
tokens ingested vs gold-term coverage. It is a cost/coverage measurement,
not LLM-judged answer quality — rung 2 remains future work; do not present
its numbers as rung 2.

Requirements: an initialized rekal repo with history, `jq`, `rg`, `python3`
(stdlib only), and — for query generation — any LLM CLI that reads a prompt
on stdin and prints a paraphrase to stdout (e.g. `claude -p`).

## Run

```bash
cd <your-repo-with-rekal-history>
BENCH=/path/to/rekal-cli/scripts/bench
OUT=./bench-out && mkdir -p $OUT

# 0. corpus card (aggregate stats only)
$BENCH/corpus_card.sh > $OUT/corpus-card.json

# 1. mine gold labels (T1 provenance, T2 decision recall, T3 dead-ends)
$BENCH/mine_labels.sh $OUT          # writes labels-t{1,2,3}.jsonl

# 2. generate natural-language queries with leakage control
export BENCH_LLM="claude -p"        # any stdin→stdout paraphrase command
python3 $BENCH/gen_queries.py $OUT  # writes queries.jsonl (+ skipped.jsonl)

# 3. run systems (B1 grep-rank, B3 bm25-only, B4 neural-only, B5 hybrid)
python3 $BENCH/run_rung1.py $OUT \
  --transcripts ~/.claude/projects/<this-repo's-dir>   # for B1

# 4. score
python3 $BENCH/score.py $OUT        # markdown tables + per-task breakdown

# 5. (optional) rung-3 proxy: drill-strategy token/coverage comparison
python3 $BENCH/run_rung3.py $OUT    # writes rung3.jsonl + markdown table

# 6. (optional) weight tuning: dev-split grid search, test-validated verdict
python3 $BENCH/tune_weights.py $OUT # writes tune-results.jsonl + tune-verdict.md

# 7. (optional) embedding-model substitution — B4'/B5', prediction P9
python3 $BENCH/run_rung1.py $OUT --systems b4x,b5x \
  --embedding-config embed-sub.json  # {"endpoint": ..., "model": ...}

# 8. T4 multi-hop: mine session pairs, generate both-needed questions, score
$BENCH/mine_t4.sh $OUT              # labels-t4.jsonl (own-repo pairs)
python3 $BENCH/gen_queries.py $OUT # now also emits t4 (validates multi-hop)
python3 $BENCH/run_rung1.py $OUT   # ranks; then score.py prints a T4 both@10 table

# 9. (opportunistic) T5 decision-drift candidates — MANUAL confirm before use
$BENCH/mine_t5.sh $OUT             # labels-t5-candidates.jsonl

# 10. Rekal usage & effectiveness (observational, no LLM)
python3 $BENCH/usage_mine.py $OUT  # usage.md + usage.json
```

Multi-repo (06-eval-strategy.md): the miners are per-repo (gold needs a
checkpoint ledger). Run steps 1, 8, 9 in each labeled repo and concatenate
the `labels-*.jsonl`; run `rekal index --include-all` once so run_rung1 scores
against the full machine-wide index as the haystack. `usage_mine.py` already
spans the whole indexed corpus.

Notes
- Weight ablations rewrite `.rekal/config.json` temporarily and restore it
  (query-time weights: no reindex between systems).
- B1 grep-rank is the *non-agentic* DCI proxy for rung 1: sessions ranked by
  rg term-hit counts over raw JSONL. The agentic grep baseline belongs to
  rungs 2–3 (see 03-benchmark.md §3) — do not present grep-rank as DCI's
  best case.
- B1 id-space join: transcript filenames (harness UUIDs) ≠ gold labels
  (Rekal ULIDs). `run_rung1.py` builds `sidmap.json` by matching a
  distinctive turn substring per gold session into the transcripts dir;
  `sidmap-report.json` records coverage. Unmapped gold sessions deflate b1 —
  treat its numbers as a lower bound below 100% coverage.
- T4 gold is a session *pair*; `score.py` reports `both@10` (both in the top
  ten, the paper's b@10) plus `partial@10`. Query generation runs a second
  LLM check that discards questions answerable from either session alone, so
  only genuine multi-hop pairs survive.
- `usage_mine.py` is observational (a natural experiment): the steering delta
  between rekal-using and non-using sessions is confounded and directional,
  not causal — it sets priors for the interventional A/B (06 §4b).
- `tune_weights.py` enforces the RHO rule: grid search on the dev split
  only, then winner-vs-incumbent on test with a paired-bootstrap CI; SHIP
  only on a test win. Tuning is valid only against the index state it ran
  on — after a reindex/re-tag, delete `tune-results.jsonl` and re-run.
- b4x/b5x swap the embedding backend via the config's `embedding` section
  and reindex each way (the embed cache makes the swap back re-embed
  nothing). Compare against b4/b5 from the same run only, and record the
  substituted model id in the manifest.
- `--explain` per-layer scores (`rekal --explain`) make the per-signal
  analysis in §6.1 of the paper directly observable.
- Splits: `gen_queries.py` tags 10% of queries `"split":"dev"`; tune on dev
  only, report test.
