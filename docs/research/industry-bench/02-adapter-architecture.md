# Adapter Architecture — Plugging Rekal into General Memory Benchmarks

The binding technical design. Core rule (decision #2 in the
[README](README.md)): **Rekal core is frozen**; everything here lives in
`scripts/industry-bench/` and touches Rekal only through the CLI and
`.rekal/config.json`.

## The insight the whole adapter rests on

Rekal's ledger anchors sessions to git commits, but nothing requires those
commits to contain real code. A benchmark conversation corpus becomes a
**synthetic git history**: one repo per benchmark *user/conversation*, one
commit per benchmark *session*, with the session transcript ingested by the
production `rekal checkpoint` pipeline exactly as a real coding session
would be. Everything downstream — hybrid recall, facets, knowledge layer,
the confidence gate, tokens-per-question accounting — then runs stock.

## <a name="ingestion"></a>1. Ingestion: the synthetic-history generator (`sh-gen`)

### Contract

```
sh-gen --dataset <longmemeval-s|longmemeval|locomo|msc> \
       --input <dataset file/dir> \
       --out <workdir> \
       [--limit N] [--seed 42]
```

For each benchmark conversation (a "haystack" of sessions), `sh-gen` produces
under `<workdir>/<conversation-id>/`:

1. **A git repo** (`git init`; default branch `main`; `rekal init` run once).
2. **One commit per benchmark session**, in chronological order:
   - The commit writes/updates one marker file per session
     (`sessions/<session-date>-<n>.md`, containing the session's date header
     only) so `files_touched` is non-empty and diffs are cheap.
   - Commit timestamps are **backdated to the benchmark session's date**
     via `GIT_AUTHOR_DATE` / `GIT_COMMITTER_DATE`. Temporal-reasoning
     questions depend on this: the ledger's time axis must reproduce the
     benchmark's.
   - Commit message: `session <date> (<benchmark-session-id>)`.
3. **A session file per benchmark session** in **Claude Code JSONL format**,
   placed where the Claude adapter discovers it:
   `~/.claude/projects/<sanitized-repo-path>/<session-id>.jsonl`
   (sanitization rule: normative implementation is
   `cmd/rekal/cli/session/claude.go` — `sessionDirFor`, ~line 127; use the
   repo's real path, sanitized the same way; **do not reimplement by guess,
   port from the Go source and add a round-trip test against
   `rekal checkpoint` actually finding the file**).
   - Turn mapping: benchmark `user` → `human`; benchmark `assistant` →
     `assistant`. No tool calls, no subagents, no `human_steering`, no
     `summary` turns — chat benchmarks have none of these, and adapters must
     not fabricate them (`session/adapter_test.go` enforces `summary` is
     Claude-only; we stay within that).
   - The JSONL entry shapes must match what `ClaudeAdapter.Parse` accepts:
     normative reference is `cmd/rekal/cli/session/claude.go` +
     `session/testdata/` fixtures. Golden-file test required (see DoD in
     [03-workstreams](03-workstreams.md#ws-b)).
4. **The ingestion loop**: for each session in order — write JSONL, write
   marker file, `git add -A && git commit` (hook fires `rekal checkpoint`).
   Set `REKAL_BENCH=1` in the *harness's own* environment when the harness
   itself runs under a coding agent, so the harness session is never
   captured into the corpus (guard: `cmd/rekal/cli/session/bench.go`).
5. **Post-ingest**: `rekal index` (full build: LSA needs the whole corpus),
   then a **verification pass** (session count, turn count, checkpoint links
   — exact SQL in [04-procedures §verify](04-procedures.md#verify)).

### Why through the pipeline, and the sanctioned shortcut

Through-the-pipeline exercises parse/dedup/link code and is immune to schema
drift. Cost: ~1 commit per session; LongMemEval-S ingests in minutes.
For **BEAM 10M-token tiers only**, a `--fast` mode may batch many sessions
per commit (marker files still one-per-session; sessions keep their own
JSONL files and timestamps). Raw `data.db` writes remain forbidden — if even
`--fast` is too slow, that finding goes in the run notes and BEAM is dropped,
not worked around.

### Embedding at scale

Default embedded nomic is fine for LongMemEval-S. For full LongMemEval /
BEAM, point `.rekal/config.json` `embedding.endpoint` at a local vLLM/Ollama
server (same model: `nomic-embed-text-v1.5`) — supported config, no core
change, data still local. Record the choice in the run manifest.

## 2. Mode mapping: Rekal's router onto chat questions

| Rekal substrate (paper #1) | Chat-benchmark analog | Implementation |
|---|---|---|
| Ledger / episodic recall (HUNT) | Single-hop & temporal questions | Stock `rekal "<question>"` → route/hunt gates → drill via `rekal query --session` |
| Decision synthesis (WHY) | Multi-hop & knowledge-update questions | Stock synthesis flow; "the decision arc" ≡ "the fact's update history" |
| Map (structural digest) | Persona/preference questions | **New prompt, same machinery**: digest-of-all-sessions ("persona map") refreshed incrementally, watermarked by last-ingested commit SHA — mirroring `map-fresh.sh`'s freshness discipline |
| Knowledge (prose at HEAD) | *(thin)* marker files carry no prose | Optional: persona map written into the repo as `docs/persona.md` per refresh, making it a real HEAD document the knowledge layer indexes — this is the honest analog of "current truth lives at HEAD" |
| Tree | none | absent; router never dispatches there |
| Silence / abstention gate | **Abstention questions (LongMemEval category 5)** | Stock `recall-route.py` SILENCE verdict → answer "I don't know". This category is where the gate — paper #1's key mechanism — is tested on shared ground. Do not special-case it. |

Router collapse: benchmark questions are classified (by the same cheap
classifier prompt the skill tip uses) into pointed / why / persona /
abstain-if-unfounded. No benchmark question kind maps to tree, so the
4-substrate router runs as a 3+gate router. This is a *configuration* of the
shipped design, not a redesign.

## 3. Gate recalibration — the one tunable, pre-registered

Chat-turn score distributions differ from coding-session distributions, so
the absolute `confidence`/`mass` thresholds in the route/hunt gates (and
optionally the `weights` mix in `.rekal/config.json`) may be recalibrated
**once, on the dev split only**, before any test question is run:

1. Split: LongMemEval's published dev/test if present; else a seeded 20%
   conversation-level holdout (seed in the manifest).
2. Grid over gate thresholds + `weights` (reuse `scripts/bench/tune_weights.py`
   conventions).
3. Freeze the chosen values in `scripts/industry-bench/calibration/<dataset>.json`,
   commit **before** the first test run (pre-registration; see
   [04-procedures §prereg](04-procedures.md#prereg)).

Per-dataset calibration is disclosed in the paper. Per-*question* or
post-hoc adjustment is fabrication; don't.

## <a name="shim"></a>4. The harness shim: Rekal behind add/search verbs

Open harnesses (memorybench, mem0-style APIs) speak roughly:
`add(user_id, messages)` / `search(user_id, query) -> contexts` /
`answer(query, contexts) -> text`.

Shim mapping (one small Python module, `scripts/industry-bench/shim/`):

- `add` → append to the pending session buffer; on session boundary, `sh-gen`
  ingests it (JSONL + commit) into that user's synthetic repo.
- `search` → run the routed recall (mode mapping above) inside the user's
  repo; return retrieved snippets/turns as the harness's context objects,
  **recording token counts** of everything returned.
- `answer` → the harness's own answer model, untouched — Rekal competes on
  retrieval + routing, not on a better answer model.

One pinned answer model and one pinned judge across ALL systems in a
comparison table ([04-procedures §pins](04-procedures.md#pins)).

## 5. Metric alignment

- **Headline**: each benchmark's official metric via its official script
  (LongMemEval QA-judge accuracy per category; LoCoMo's official
  QA/judge protocol).
- **Secondary**: Rekal's blind answer-sufficiency judge (paper #1
  continuity), same rubric, clearly labeled ours.
- **Always**: tokens-per-question — retrieved-context tokens and full
  answer-path tokens, mean and p95, per category. This column is mandatory
  in every table in every run record.
- LoCoMo and LongMemEval numbers are never averaged together
  ([00 §6](00-landscape.md)).

## 6. Package layout (target)

```
scripts/industry-bench/
  README.md            # points here; quickstart
  sh_gen/              # WS-B: synthetic-history generator
  shim/                # WS-E: add/search/answer shim + token accounting
  modes/               # WS-C: persona-map prompt + refresh; question classifier
  calibration/         # WS-D: frozen per-dataset gates/weights (committed pre-test)
  baselines/           # WS-F: mem0/zep/full-context/naive-RAG runners
  scoring/             # WS-E: official-script wrappers + our judge + tables
  runs/                # run manifests + aggregate outputs (committed)
  datasets/            # .gitignore'd; acquisition scripts only are committed
```
