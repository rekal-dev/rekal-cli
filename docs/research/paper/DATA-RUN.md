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
- B1's id-space join is automatic (`sidmap.json`); check
  `sidmap-report.json` coverage and record it in the manifest. Below 100%,
  B1's numbers are a lower bound — say so wherever they're reported.

## 3b. Optional: weight tuning (dev-tuned, test-validated)

```bash
python3 $BENCH/tune_weights.py $RUN        # tune-verdict.md: SHIP/REJECT
```

The sweep sees the dev split only; the verdict compares the dev winner
against the incumbent on TEST with a paired-bootstrap CI. A tuning result
is valid only against the index state it ran on — after any reindex or
re-tag (e.g. the summary re-tag), delete `tune-results.jsonl` and re-run
before trusting a SHIP. Record weights, verdict, and CI in the manifest.

## 3c. Optional: embedding-model substitution (fills `tab-embed`, P9)

Requires a running OpenAI-compatible embedding endpoint (Ollama, vLLM,
TEI) serving a stronger code-tuned model. Write its config to a file:

```bash
cat > $RUN/embed-sub.json <<'EOF'
{"endpoint": "http://localhost:11434/v1/embeddings",
 "model": "<substituted-model-id>"}
EOF
python3 $BENCH/run_rung1.py $RUN --systems b4x,b5x \
  --embedding-config $RUN/embed-sub.json
python3 $BENCH/score.py $RUN            # b4x/b5x tables appended
```

Notes:
- The swap costs one `rekal index` each way (the harness restores config
  and reindexes back automatically); budget the time on a large corpus.
- Compare b4x against b4 and b5x against b5 from the SAME run (same index
  state, same splits). Record the substituted model id in the manifest.
- P9's registered prediction: B4′ improves materially, B5′ barely — if
  the hybrid jumps instead, that's the finding, and the shipped default
  model/weights change.

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

## 4c. Optional: cross-repo wiki A/B (explicit — read before running)

This step folds the operator's OTHER repos' sessions into this repo's
index and lets wiki pages cite them. It is doubly gated: the import is an
explicit preference, and foreign evidence is used only because this step
says so. Everything stays index-only and unpushable; the ONLY place
foreign knowledge can become committed text is the wiki PR itself — which
is exactly what this step measures.

1. Enable and verify the import:
   ```bash
   rekal index --include-all        # or --include <specific project dirs>
   rekal query --index "SELECT origin, count(*) FROM session_facets
     GROUP BY origin"               # NULL = own repo; repo:/... = foreign
   ```
2. Pick the K'=5 topics from step 4b most likely to span repos. Generate
   each topic page TWICE, in separate workflow runs with distinct
   `workflow_name`s:
   - mode A: own-repo evidence only (ignore origin-labeled recall hits)
   - mode B: cross-repo mode per the rekal-wiki skill (foreign citations
     labeled with origins; PR body declares the origin list)
3. Record: per-page foreign-citation count (mode B), and re-run the step-4b
   broad-query coverage measurement against both page sets — the coverage
   delta is what foreign evidence bought. Fills the `tab-wiki` cross-repo
   row (`n pages with foreign citations / coverage delta`).
4. Privacy: the mode-B pages quote nothing verbatim from foreign sessions
   (the skill's rule); before any push, read the pages once specifically
   checking for foreign content that shouldn't be visible to this repo's
   audience. If in doubt, keep mode-B pages on their branch unmerged —
   the measurement works from the branch; the merge is the operator's call.

## 5. Write results into the paper

The paper is organized around pre-registered predictions **P1–P9** (§6 of
the paper); every results table carries a `Verdict:` slot. Fill values AND
verdicts — a verdict is `holds` / `fails` (P7 may also be `cache not
built`). Never reframe a failed prediction; report it.

In `docs/research/paper/rekal-paper.typ`, replace only the `tbd[…]` values
this run measured:

| Paper location | Prediction | Source |
|---|---|---|
| Abstract: N sessions / N turns, MRR vs baseline, k× tokens | — | corpus-card.json, rung1.md |
| `tab-tasks` n column (T1–T3 only; T4/T5 stay tbd) | — | wc -l labels-*.jsonl / queries.jsonl |
| §5 corpus card sentence | — | corpus-card.json |
| `tab-rung1` rows + verdict | P1, P2 | rung1.md test split |
| `tab-embed` rows + verdict (only if step 3c ran) | P9 | rung1.md b4/b4x/b5/b5x test split |
| `tab-drill` rows + verdict | P3 | rung3.md pooled table (per-task split for the verdict) |
| `tab-wiki` rows + verdict (only if step 4b ran) | P7 | ledger lineage query + page-vs-drill proxy |
| `tab-wiki` cross-repo row (only if 4c ran) | P7 | origin counts + mode A/B coverage delta |
| Label-precision value (§5 + P8 verdict) | P8 | 50-pair T1 hand audit — do this one; it's cheap |
| `tab-rung23` (judged), scale/freshness curves, rung 4 | P4, P5, P6 | NOT this run — leave tbd |

Optional but high-value: replace the §2 worked example's illustrative JSON
with a real (redacted) recall result from this corpus — pick a T2 label
whose steering turn is quotable, run the query, paste the actual output,
and delete the "(Illustrative…)" caveat sentence.

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
