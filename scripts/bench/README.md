# RekalBench harness (v0 — rung 1)

Implements the retrievability rung of `docs/research/03-benchmark.md`:
self-labeled queries from your own corpus, run against Rekal (full hybrid +
single-signal ablations) and a grep-rank baseline over raw transcripts, then
scored (MRR / Recall@k / nDCG with bootstrap CIs). Everything runs locally;
nothing but aggregate numbers is meant to leave the machine.

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
```

Notes
- Weight ablations rewrite `.rekal/config.json` temporarily and restore it
  (query-time weights: no reindex between systems).
- B1 grep-rank is the *non-agentic* DCI proxy for rung 1: sessions ranked by
  rg term-hit counts over raw JSONL. The agentic grep baseline belongs to
  rungs 2–3 (see 03-benchmark.md §3) — do not present grep-rank as DCI's
  best case.
- `--explain` per-layer scores (`rekal --explain`) make the per-signal
  analysis in §6.1 of the paper directly observable.
- Splits: `gen_queries.py` tags 10% of queries `"split":"dev"`; tune on dev
  only, report test.
