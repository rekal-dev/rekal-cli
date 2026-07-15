# RUN — the RekalBench data pack for the paper

**Hand this whole file to an agent running on the operator's machine** (the
one with real Rekal history across several repos). It produces one harmonised
data pack that supersedes `runs/single-corpus/` and fills every table in the
paper. Rationale lives in `06-eval-strategy.md`; this is the executable
version. Aggregates only leave the machine — no session content, no commit
messages, no foreign paths.

## 0. Preconditions

```bash
rekal version                      # must have summary role, cross-repo import, sidmap
which jq rg python3
export BENCH_LLM="<model A -p>"     # query generation (paraphrase)
export BENCH_ANSWER_LLM="<model B>"# rung-2 answering — DIFFERENT model
export BENCH_JUDGE_LLM="<model C>"  # rung-2 judging — DIFFERENT again (bias control)
BENCH=<path>/rekal-cli/scripts/bench
RUN=<path>/rekal-cli/docs/research/runs/$(date +%F)
mkdir -p "$RUN"

rekal index --include-all          # ONE snapshot: all repos folded; freeze here
```

Everything below scores against this single index. If you reindex or re-tag,
start over — one run, one snapshot (RHO rule).

## 1. Corpus card  →  §5 + abstract N

```bash
$BENCH/corpus_card.sh > $RUN/corpus-card.json
rekal query --index "SELECT origin, count(*) FROM session_facets GROUP BY origin" \
  > $RUN/repo-breakdown.jsonl        # own (NULL) vs each imported repo
```

## 2. Labels — multi-repo (gold needs a checkpoint ledger)

For **each repo where Rekal is installed**, `cd` into it and:

```bash
$BENCH/mine_labels.sh $RUN/_tmp      # T1/T2/T3
$BENCH/mine_t4.sh     $RUN/_tmp      # T4 multi-hop pairs
$BENCH/mine_t5.sh     $RUN/_tmp      # T5 candidates (manual-confirm; usually empty)
cat $RUN/_tmp/labels-t1.jsonl >> $RUN/labels-t1.jsonl   # concat across repos
cat $RUN/_tmp/labels-t2.jsonl >> $RUN/labels-t2.jsonl
cat $RUN/_tmp/labels-t4.jsonl >> $RUN/labels-t4.jsonl
rm -rf $RUN/_tmp
```

Record per-task label counts and which repos yielded T3/T4.

## 3. Queries (leakage-controlled; T4 multi-hop validated)

```bash
python3 $BENCH/gen_queries.py $RUN   # queries.jsonl (+ skipped.jsonl); tags 10% dev
```

## 4. Rung 1 — retrievability  →  retrieval table (P1, P2, T4 both@10)

```bash
python3 $BENCH/run_rung1.py $RUN \
  --transcripts ~/.claude/projects/<this-repo-dir>   # B1 grep sidmap
python3 $BENCH/score.py $RUN > $RUN/rung1.md
```

Report the TEST split. `rung1.md` carries per-task + pooled MRR/R@k/nDCG with
CIs, and a T4 both@10 line. Check `sidmap-report.json` coverage; note it.

## 5. Real in-the-wild recall  →  §4c (the strongest table)

```bash
python3 $BENCH/mine_wild.py $RUN                       # $RUN/wild/queries.jsonl
python3 $BENCH/run_rung1.py $RUN/wild --systems b5,b3
python3 $BENCH/score.py $RUN/wild > $RUN/wild/rung1.md
```

This grades current recall against the sessions agents **actually drilled
into** after their real queries — no synthetic queries. `wild/wild-meta.json`
also gives the real recall-invocation count and the cross-repo drill count.

## 6. Usage & cross-repo effectiveness  →  §4a, §4d

```bash
python3 $BENCH/usage_mine.py $RUN                      # usage.md + usage.json
# Optional own-vs-machinewide coverage A/B (heavy: two reindexes):
#   rekal index --no-local && run_rung1 on a fixed query subset -> ownrepo-rung1.md
#   rekal index --include-all && run_rung1 same subset          -> allrepo-rung1.md
#   the Recall@k delta is what cross-repo import bought.
```

## 7. Rung 2 — judged answer quality (LLM judge)  →  judged table

```bash
python3 $BENCH/run_rung2.py $RUN \
  --transcripts ~/.claude/projects/<this-repo-dir> > $RUN/rung2.md
```

Distinct answer/judge models; the judge sees the gold turn. Context is
generous by design — tokens are not the metric here (that is rung 3), so let
the answering model see plenty; the question is whether the system surfaced
the answer at all. Reports judged accuracy **per task and pooled**, with the
CORRECT/PARTIAL/WRONG breakdown, for B0 (no memory) / B1 (grep) / B5 (Rekal).
Also hand-check 50 judgements for agreement and record the rate.

## 8. Rung 3 — drill-cost proxy  →  efficiency figure

```bash
python3 $BENCH/run_rung3.py $RUN > $RUN/rung3.md
```

## 9. Optional — weight tuning, P8 audit

```bash
python3 $BENCH/tune_weights.py $RUN > $RUN/tune-verdict.md   # SHIP/REJECT
```

**P8 label audit (do this — it's cheap and it's the benchmark's validity):**
read 50 random T1 `(commit, gold session)` pairs; for each, does the session
actually discuss that commit's change? Record precision in the manifest.

## 10. Manifest + commit

Write `$RUN/manifest.json`: date, rekal version, corpus card, model ids
(gen/answer/judge), label/query/skip counts, dev/test sizes, sidmap coverage,
per-repo variance, P8 precision, weights used. Then:

```bash
git diff --cached   # MUST be aggregates only — no turn content, no foreign paths
```

Commit `$RUN/` (json + `.md` aggregate tables — NOT the `labels-*.jsonl` /
`queries.jsonl` if they contain session text; check first) plus a one-line
summary.

---

## The data pack (screenshot these for the paper)

| File | Paper artifact | What it shows |
|---|---|---|
| `corpus-card.json`, `repo-breakdown.jsonl` | §5 corpus card, abstract N | scale across repos |
| `rung1.md` (test split) | retrieval table | P1 (vs grep), P2 (ablations), per-task, **T4 both@10**, per-repo variance |
| `wild/rung1.md` + `wild/wild-meta.json` | §4c real-recall table | recall vs sessions agents really drilled; real return/drill rates; **cross-repo drills** |
| `usage.md` | §4a effectiveness | adoption, drill-through, steering delta |
| `rung2.md` | judged table | **answer accuracy** per task + pooled, CORRECT/PARTIAL/WRONG, B0/B1/B5 (the LLM-judge result) |
| `rung3.md` | efficiency | drill tokens vs gold-term coverage |
| `tune-verdict.md` | footnote | weights SHIP/REJECT on held-out test |
| `manifest.json` | reproducibility | one canonical run record |

When these exist, send them over and I harmonise the paper to this single run
(retrieval + T4 + wild + usage + cross-repo + judged), retire `single-corpus`,
and finish the site.
