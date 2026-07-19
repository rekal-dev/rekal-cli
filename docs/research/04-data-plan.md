# Data plan: building the benchmark corpus from the local store

The operator's machine already holds the asset every memory paper has to
manufacture: hundreds of real AI coding sessions (one repo alone: ~500
sessions / ~63k turns / ~48k tool calls) with commits linking them to
verified outcomes. This doc is the runnable path from that store to
RekalBench inputs. Everything runs locally; only aggregate numbers ever
leave the machine.

## 0. Which data lives where (matters for labels)

- **Per-repo `data.db`** (source of truth, in each initialized repo's
  `.rekal/`): sessions, turns, tool_calls, **checkpoints**,
  **checkpoint_sessions**, files_touched. T1/T3/T5 labels need
  `checkpoints` — they are **per-repo**. Run label extraction inside each
  rich repo.
- **Per-repo `index.db`** (+ optional cross-repo import): turns_ft,
  session_facets, files_index, file_cooccurrence, tool_calls_index. Rung-1
  retrieval runs here. `rekal index --include-all` folds the whole machine's
  sessions in (origin-labeled, index-only) — used for the **scale sweep**,
  not for labels.
- **Raw JSONL** (`~/.claude/projects/*`): the corpus baseline B1 greps.
  Same underlying material — that is what makes the comparison fair.

## 1. Pick benchmark repos

Choose the 2–3 repos with the deepest history (the ~500-session repo first).
In each:

```bash
rekal version              # ensure a binary with the unified rekal skill (analytics census in references/)
rekal index                # fresh index from data.db
```

## 2. Corpus card (per repo + machine-wide)

```bash
# per-repo card
rekal query "SELECT count(*) FROM sessions"
rekal query "SELECT count(*) FROM turns"
rekal query "SELECT count(*) FROM tool_calls"
rekal query "SELECT count(*) FROM checkpoints"
rekal query "SELECT count(*) FROM checkpoint_sessions"
rekal query "SELECT role, count(*) FROM turns GROUP BY role ORDER BY 2 DESC"
rekal query "SELECT min(captured_at), max(captured_at) FROM sessions"
rekal query "SELECT git_branch, count(*) FROM checkpoints GROUP BY git_branch ORDER BY 2 DESC LIMIT 20"

# machine-wide (after: rekal index --include-all)
rekal query --index "SELECT coalesce(origin,'this repo') AS src, count(*) FROM session_facets GROUP BY src ORDER BY 2 DESC"
```

Record into `docs/research/runs/<date>-corpus-card.md`.

## 3. Mine labels

### T1 — commit → sessions (the backbone; expect hundreds–thousands)

```bash
rekal query "SELECT cs.checkpoint_id, c.git_sha, c.git_branch, cs.session_id \
  FROM checkpoint_sessions cs JOIN checkpoints c ON c.id = cs.checkpoint_id \
  ORDER BY c.ts"
```

Commit message + files (query material, *not* in the index — the built-in
leakage control) come from git:

```bash
git show -s --format='%H%x09%s' <git_sha>
git show --name-only --format= <git_sha>
```

### T2 — steering turns (decision moments)

```bash
rekal query "SELECT t.session_id, t.turn_index, t.content \
  FROM turns t WHERE t.role = 'human_steering' AND length(t.content) > 80"
```

Context window for paraphrase generation (hold the turn itself out of quoted
material):

```bash
rekal query --session <id> --offset <turn_index-3> --limit 7
```

### T3 — abandoned-branch sessions (dead-end set)

```bash
# branches whose tips never reached the default branch: candidates
rekal query "SELECT DISTINCT c.git_branch FROM checkpoints c \
  WHERE c.git_branch NOT IN ('main','master')"
# then per branch, in git:  git merge-base --is-ancestor <tip> origin/main || echo ABANDONED
# sessions of abandoned branches:
rekal query "SELECT cs.session_id, c.git_branch FROM checkpoint_sessions cs \
  JOIN checkpoints c ON c.id = cs.checkpoint_id WHERE c.git_branch = '<b>'"
```

### T4 — linked session pairs (multi-hop)

```bash
rekal query --index "SELECT a.session_id AS s1, b.session_id AS s2, count(*) AS shared \
  FROM files_index a JOIN files_index b \
    ON a.file_path = b.file_path AND a.session_id < b.session_id \
  GROUP BY s1, s2 HAVING shared >= 3 ORDER BY shared DESC LIMIT 200"
# plus lineage pairs:
rekal query --index "SELECT parent_session_id, session_id FROM session_facets \
  WHERE parent_session_id IS NOT NULL"
```

### T5 — drift candidates (small, part-manual)

```bash
rekal query --index "SELECT f1.session_id AS earlier, f2.session_id AS later, f1.file_path \
  FROM files_index f1 JOIN files_index f2 ON f1.file_path = f2.file_path \
  JOIN session_facets s1 ON s1.session_id = f1.session_id \
  JOIN session_facets s2 ON s2.session_id = f2.session_id \
  WHERE s2.captured_at > s1.captured_at + INTERVAL 14 DAY LIMIT 500"
# then filter to pairs where both sessions contain human_steering turns; confirm ~30 by hand
```

## 4. Generate queries (any LLM CLI; record model id + prompt)

Per label: paraphrase into a natural developer question; enforce the n-gram
overlap filter (03 §T2); target ~500 T1, ~300 T2, ~50 T3, ~100 T4, ~30 T5.
Store as JSONL: `{task, query, gold_session_ids, gold_turn_index?, meta}`.

## 5. Run systems

- **B3/B4/B5 (Rekal + ablations):** set `.rekal/config.json` weights, then
  `rekal -n 10 "<query>"` and parse the JSON (`session_id`, `score`,
  `snippet_turn_index`). No reindex between weight settings.
- **B1 (grep/DCI):** agent (same model as B6) with `rg`/`jq`/`sed` over
  `~/.claude/projects/<project-dir>/`, fixed turn budget, published prompt.
- **B2 (static notes):** one-time LLM distillation of the corpus to ≤8k
  tokens; answering agent gets only that file.
- **Scale sweep:** re-run rung 1 at date-cut subsets (10/25/50/100%) — for
  Rekal, rebuild index from a date-filtered data.db copy; for B1, a
  date-filtered transcript dir. Time everything.

## 6. Report

Everything goes to `docs/research/runs/<date>/`: corpus card, label stats +
audit precision, per-task metrics tables with CIs, the accuracy-vs-tokens
curve, scale-sweep chart, and the run manifest (models, prompts, weights,
rekal version). That folder is the regression baseline for every future
engine change.

## 7. Privacy line (absolute)

Raw sessions, labels, queries, and snippets never leave the machine. What
gets published: aggregate counts, metric tables, charts, prompts, and the
harness code. The corpus card contains no content, only numbers. (Sessions
already pass through `scrub` at capture; treat benchmark artifacts with the
same rule anyway.)

## 8. Effort estimate

- Corpus card + T1/T2 label mining: ~1 hour (SQL above is copy-paste).
- Query generation: ~1–2 hours of LLM calls (~1k cheap calls), one review
  pass over samples.
- Rung 1, B1–B5: an afternoon (B5/B3/B4 are trivial loops; B1 is the slow
  one).
- Rung 2–3 (200-query subset): a day including judge runs.
- Rung 4 (10–20 A/B tasks): the expensive week; do last, only if rungs 1–3
  hold up.
