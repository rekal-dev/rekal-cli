# DATA-RUN — agent runbook: profile the corpus, run RekalBench, fill the paper

You are running on the operator's machine, inside a repo that has real Rekal
history (`.rekal/data.db` with hundreds of sessions). Your job: produce the
measured values for `docs/research/paper/rekal-paper.typ`, write the raw
outputs into a run directory, replace the `tbd[…]` placeholders the run can
fill, and recompile the PDF. Aggregates only — no session content leaves the
machine or enters the paper.

## 0. Preconditions (verify before anything)

```bash
rekal version                  # must include the summary role + summary_turn_index
                               # (build from main if older)
rekal index                    # REQUIRED: re-tags legacy compaction summaries;
                               # without this, summary_turn_index is absent and
                               # the rung-3 summary column will be empty
which jq rg python3            # all required
export BENCH_LLM="claude -p"   # any stdin→stdout paraphrase command
BENCH=<path-to-rekal-cli>/scripts/bench
RUN=docs/research/runs/$(date +%Y-%m-%d)   # in the rekal-cli repo
mkdir -p $RUN
```

Sanity check the summary harvest before running anything long:

```bash
rekal query --index "SELECT role, COUNT(*) AS n FROM turns_ft GROUP BY role"
# expect a nonzero 'summary' row on a corpus with long sessions
```

## 1. Corpus card (fills §5.1 and the abstract's N values)

```bash
$BENCH/corpus_card.sh > $RUN/corpus-card.json
```

## 2. Labels → queries (leakage-controlled)

```bash
$BENCH/mine_labels.sh $RUN                 # labels-t{1,2,3}.jsonl
python3 $BENCH/gen_queries.py $RUN         # queries.jsonl + skipped.jsonl
```

Record: label counts per task, skipped count (leakage filter), dev/test
sizes. These go in the run manifest and the paper's task table (`tab-tasks`
n column).

## 3. Rung 1 — retrievability (fills Table 3, `tab-rung1`)

```bash
python3 $BENCH/run_rung1.py $RUN \
  --transcripts ~/.claude/projects/<this-repo's-dir>   # for B1 grep-rank
python3 $BENCH/score.py $RUN > $RUN/rung1.md
```

Rules (RHO discipline, non-negotiable):
- Report the TEST split only. The dev split may be used to tune weights;
  if you tune, record before/after weights in the manifest.
- Report bootstrap CIs exactly as score.py emits them.
- B1 grep-rank is the non-agentic DCI proxy — do not present it as DCI's
  best case (that is rungs 2–4).

## 4. Rung 3 proxy — drill strategies (fills `tab-drill`)

```bash
python3 $BENCH/run_rung3.py $RUN > $RUN/rung3.md
```

Use the POOLED table's three columns (tokens / coverage / coverage per 1k);
they map one-to-one onto `tab-drill`'s two rows (window, summary-first).
Also record: n, with-summary n, and the per-task tables (T1 vs T2 contrast
is the paper's stated hypothesis — check whether summary-first wins broad
T1 queries while the window wins pointed T2 lookups, and say which way it
went in the caption).

Honesty boundary: this is context-assembly cost + gold-term coverage, NOT
answer quality. It must never be reported as rung 2.

## 4b. Optional: L3-gate wiki experiment (fills `tab-wiki`)

Only if time permits; requires the rekal-wiki skill (installed by `rekal
init` on a current binary).

1. Generate K=10 topic pages with the rekal-wiki skill, **as a dynamic
   workflow with a memorable name** (one subagent per topic) — the name is
   how the ledger finds the run afterwards.
2. Generation cost, from the ledger itself (after the wiki PR merges and
   the checkpoint lands):
   ```bash
   rekal query --index "SELECT session_id, turn_count, tool_call_count
     FROM session_facets WHERE workflow_name = '<run name>'"
   ```
   Record turns and tool calls per page; take tokens-ingested per topic
   from the harness's own usage accounting if available.
3. Cache payoff: pick 20 broad queries (T1/T3-style) whose gold sessions
   are cited by some generated page. For each, measure both paths with the
   rung-3 proxy method (tokens = bytes/4, gold-term coverage):
   - page path: read only the topic page (+ index.md)
   - recall path: `rekal "<q>"` + the winning drill from `rung3.md`
4. Maintenance price: re-run the staleness check weekly; record pages
   invalidated per week of new sessions.

Honesty: same boundary as the drill proxy — cost and coverage, not answer
quality. If pages age fast or lose on coverage, that IS the result: the
cached L3 layer doesn't get built (roadmap R11).

## 5. Write results into the paper

In `docs/research/paper/rekal-paper.typ`, replace only the `tbd[…]` values
this run measured:

| Paper location | Source |
|---|---|
| Abstract: N sessions / N turns, MRR vs baseline | corpus-card.json, rung1.md |
| §1 corpus parenthetical | corpus-card.json |
| `tab-tasks` n column (T1–T3 only; T4/T5 stay tbd) | wc -l labels-*.jsonl / queries.jsonl |
| §4 label-precision value | only if you hand-audit 50 T1 pairs; else leave tbd |
| §5.1 corpus card | corpus-card.json |
| Table 3 (`tab-rung1`) B1/B3/B4/B5 rows | rung1.md test split |
| `tab-drill` both rows | rung3.md pooled table |
| `tab-wiki` (only if step 4b ran) | ledger lineage query + page-vs-drill proxy |
| §6.3 scale/freshness, Table 4, rung 4 | NOT this run — leave tbd |

Then:

```bash
grep -c 'tbd\[' rekal-paper.typ    # record remaining count in the manifest
cd docs/research/paper
python3 -c "import typst; typst.compile('rekal-paper.typ', output='rekal-paper.pdf')"
```

Do NOT delete the DRAFT banner or the Results-section placeholder caveat —
they stay until every tbd is gone (rungs 2 and 4 are not in this run).

## 6. Manifest + commit

Write `$RUN/manifest.json`: date, rekal version, corpus-card summary, model
ids (BENCH_LLM), weights used (and any dev-split tuning), label/query/skip
counts, dev/test sizes, remaining-tbd count. This is the regression baseline
for future engine changes.

Commit together: `$RUN/` (corpus-card.json, labels meta counts — NOT the
label/query content files if they contain session text; check before
adding), the edited `.typ`, and the recompiled `.pdf`. Suggested message:
`paper: fill rung-1 + drill-proxy numbers from <date> corpus run`.

Privacy check before push: `git diff --cached` must contain aggregate
numbers only — no turn content, no commit messages from private repos, no
paths outside this repo.
