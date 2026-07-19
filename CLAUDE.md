# Rekal CLI

## Soul

Before making any design decision, read `SOUL.md`. It defines the two problems Rekal exists to solve and the seven beliefs that guide every choice. If a decision conflicts with the soul, the decision is wrong.

When working on a problem, consult Rekal's own memories first:

```bash
rekal "<describe the problem>"
```

The prior context for what you're working on may already exist.

## Standing Rules

- Keep this file up to date. Any change to commands, packages, files, or behavior must be reflected here. Update `--help` text when command behavior changes. Update `docs/spec/command/` when a command spec changes. Stale docs are worse than no docs.
- Consult `SOUL.md` before design decisions. Consult `rekal` before starting work on a problem.

## Architecture

Single binary. Everything embedded — CLI, database engine, embedding model, compression dictionary.

- CLI: Cobra (`github.com/spf13/cobra`)
- Storage: DuckDB via `github.com/marcboeker/go-duckdb` (`database/sql` interface)
- Compression: zstd via `github.com/klauspost/compress` with preset dictionary
- IDs: ULID via `github.com/oklog/ulid/v2`
- Embeddings: LSA (gonum) + Nomic (platform-specific builds)
- Build: mise, go modules
- Lint: golangci-lint v2 (2.8.0)
- Language: Go 1.25.6

Two databases in `.rekal/`:
- `data.db` — immutable source of truth. Append-only. Pushed to git.
- `index.db` — local derived index. Rebuilt from data.db. Never pushed.

This split is a direct consequence of the soul: thin on the wire, rich on the machine.

The `.rekal/` store lives in the repository's **main worktree**; every linked
git worktree resolves to that one shared store via `gitx.MainWorktreeRoot`
(a no-op for non-worktree repos, so existing installs need no migration). All
store-path helpers (`db.StoreDir`, `cli.RekalDir`, and the `Open*`/`*Path`
functions) funnel through it; git-state helpers (`HeadSHA`/`CurrentBranch`) and
session discovery keep using the invoking worktree.

## Key Directories

### Commands (`cmd/rekal/`)

- `main.go`: Entry point

### Core CLI (`cmd/rekal/cli/`)

- `root.go`: Root command (recall is the default) + command registration
- `recall.go`: Recall command orchestration — open/migrate/auto-rebuild the
  index DB, refresh the knowledge layer (watermark-gated), call the `search`
  package, marshal JSON. The ranking engine itself lives in `search/`.
- `knowledge_index.go`: Knowledge-layer build/refresh — chunk the repo's
  tracked prose files at HEAD into `index.db` (`knowledge_chunks`), diffing
  stored git blob SHAs against `git ls-tree -r HEAD` so only changed files
  re-chunk; commit-SHA watermark (`knowledge_head_sha`) makes the steady
  state one rev-parse. Called by `index` (full) and recall (incremental,
  best-effort). See `docs/design/knowledge-layer.md`
- `checkpoint.go`: Capture session after commit
- `push.go`: Push data to remote branch (wire encode/commit lives in `transport/`)
- `sync.go`: Sync team context (wire decode/import lives in `transport/`)
- `init.go`: Bootstrap Rekal in a git repo — store, hooks, orphan branch,
  skill (tip + scripts + references), and one marker-tagged CLAUDE.md sentence (the whole DX:
  init, done; `clean` removes the line, refresh replaces it in place)
- `clean.go`: Remove Rekal setup — completely, no residue
- `index_cmd.go`: Rebuild index DB from data DB (structural: FTS/facets/LSA/
  knowledge chunks). Deep-semantic session + knowledge vectors are deferred
  to background `rekal embed` after the atomic rename. Also carries the
  cross-repo local-import flags (`--include-all`/`--include`/`--no-local`),
  which set a persistent preference and rebuild
- `embed_cmd.go`: `rekal embed` — fill missing semantic vectors in budgeted
  bites (session + knowledge), releasing the DuckDB write lock between
  passes so recall can interleave. Spawned by `index`/`sync`; safe to run
  by hand. Lock: `.rekal/embed.lock`; log when background: `.rekal/embed.log`
- `config.go`: Two-tier config, both gitignored/local-only (never committed,
  pushed, or synced) — local `.rekal/config.json` deep-merges over global
  `~/.config/rekal/config.json` (path honors `$REKAL_CONFIG_HOME` then
  `$XDG_CONFIG_HOME`), precedence local → global → built-in defaults. Merge is
  per-key: `embedding` inherits wholesale, `weights` field-by-field,
  `local_import` not inherited (per-repo), `scoring_lineage` **global-only**
  (machine diagnostic switch; ignored from local, never written to the repo
  file; default off — observe-only NDJSON of recall score lineage + stage
  timings to stderr or a lumberjack-rotated local path; envelope
  `ts`/`v`/`run_id`/`event` joins query→candidate→result). `readConfig` is
  local-only (the write path — the `--include*` flags read-modify-write it,
  so global values are never baked in); `readMergedConfig` is the
  consumption view (recall weights, index embedding, scoring lineage). Holds
  the cross-repo `local_import` preference,
  the recall-tuning `weights` (BM25/LSA/nomic layer mix, steering boost,
  summary boost, subagent discount, facet boost — applied at query time, no
  reindex), and
  the `embedding` section (OpenAI-compatible HTTP backend: endpoint/model/
  api_key with `$VAR` expansion and `api_key_env`; a Cohere Embed model under
  the `openai` provider auto-sends `input_type`)
- `local_import.go`: Cross-repo local session import — folds this machine's
  other Claude Code sessions (all repos + shell, from `~/.claude/projects/*`)
  into the index. **Index-only, never `data.db`**, so imported sessions are
  structurally unpushable to the team; origin-labeled, deduped by content hash
- `log.go`: Show recent checkpoints
- `query.go`: Raw SQL access
- `version.go`: Version constant (set via ldflags)
- `errors.go`: SilentError pattern for clean error output
- `preconditions.go`: Shared checks — `RequireInitializedRepo` (git repo +
  init done) plus the individual `EnsureGitRoot`/`EnsureInitDone` helpers

### Packages (`cmd/rekal/cli/`)

- `codec/`: Binary wire format — frame encoding/decoding, body, dictionary, preset zstd dictionary
- `transport/`: Git-side sync — encode checkpoints to the orphan-branch wire
  format and decode them back (`export`/`import`/remote-sync glue, orphan-branch
  commit). Sits above `codec`, `db`, and `gitx`; called by `push`/`sync`/`init`.
  `export` applies the **merged-only gate** (`filterMerged`): checkpoints reach
  the wire only when their `git_sha` is an ancestor of the default branch or
  their branch landed as a patch-equivalent squash commit — unmerged work
  stays local (see `docs/design/merged-only-sharing.md`)
- `gitx/`: Thin git-plumbing helpers (rev-parse, show, hash-object, config,
  `rekal/<email>` branch name, `DefaultBranch`/`IsAncestor`/`IsSquashMergedInto`/
  `BranchTip` for the merged-only export gate, `MainWorktreeRoot` for the
  worktree-shared store) shared by the command and transport layers
- `knowledge/`: Prose-file chunker for the knowledge layer — markdown/plain
  text into heading-anchored sections (breadcrumb trails, 1-indexed line
  ranges, content hashes). Pure functions, no git/DB
- `search/`: Recall ranking engine — hybrid BM25 + LSA + Nomic scoring plus
  the additive facet layer (BM25 over per-session tool paths + command
  prefixes + steering text; `weights.facet_boost`, default 0.3, `0` =
  byte-identical pre-facet ranking; fails soft without a facet FTS index),
  with configurable weights (`weights.go`; query-time only), signal weighting
  (steering-turn boost, compaction-summary boost, subagent down-weight),
  conversation grouping
  (see `docs/agent-metadata.md`), snippet extraction, the LSA
  query-projection cache (`projection.go`), absolute `confidence` + raw
  BM25 `mass` for silence gates (`confidence.go`; ranking still uses
  max-normalized `score`), thin-query rejection (empty/whitespace/single-char
  → empty hybrid, no knowledge), the top-level `semantic`
  `{status:"warming",retryable:true}` field (present only while the nomic daemon
  loads the model — recall degraded to keyword+LSA; the agent re-runs with
  backoff for full quality, taught by `SKILL.md`), the per-result
  `summary_turn_index` pointer (latest compaction-summary turn — pointer,
  never the 10-17KB payload; drill with `--role summary`), the
  `--explain` enrichments (per-layer normalized scores + query-time
  related-session joins over `files_index`; default output unchanged
  without the flag), and optional scoring-lineage NDJSON (`lineage.go` —
  global `scoring_lineage` config only; default off; schema v3: per-layer
  raw/norm/contrib + stage `timings_ms` + candidate/returned `confidence`/`mass`
  + `result.knowledge` file hits with winning-chunk bm25/semantic;
  `result.semantic{used,backend,model}` names the real embedder —
  `http`|`embedded` + model id — distinct from the historical layer key
  `nomic` in weights/timings/skipped; observe-only, ranking unchanged), and
  the **knowledge layer** (`knowledge.go` — hybrid
  BM25 + chunk-vector cosine over prose-file chunks at HEAD, blended with the
  `layers2` keyword/semantic split, query vector shared from the session
  semantic pass; chunks scored / files returned as pointers with
  anchor + lines + `sessions` provenance edge; separate additive `knowledge`
  block above `results`, never merged with session ranking; fails soft
  without a knowledge FTS index, and to keyword-only without chunk vectors —
  `docs/design/knowledge-layer.md`)
- `session/`: Claude Code `.jsonl` parsing — extract turns, tool calls, deduplicate.
  `SkipCapture` refuses RekalBench/harness sessions (`REKAL_BENCH` /
  `REKAL_SKIP_CHECKPOINT`, bench cwd, gen_queries prompt fingerprints) so
  synthetic fixtures are never checkpointed or locally imported into a
  real store.
  Turn roles: `human`, `human_steering` (queue-operation captures), `assistant`,
  `summary` (isCompactSummary compaction distillations; rows written before the
  role existed stay `human` in append-only data.db and are reclassified by
  content fingerprint in the derived views, scoped to source='claude' sessions
  so other agent types are untouched — `db.SummaryFingerprint`).
  `local.go` enumerates/resolves project session dirs under `~/.claude/projects/*`
  for the cross-repo local import
- `scrub/`: Redact secrets, anonymize file paths, and guarantee valid UTF-8 (`SanitizeText`) before any DB insert — sessions (`checkpoint` / cross-repo import after parse) and knowledge chunks (`knowledge.ChunkFile` + `db.InsertKnowledgeChunks`). DuckDB rejects invalid-UTF-8 VARCHAR binds, so this is the last-line guard against `could not bind parameter` (prose `.txt` dumps with binary/truncated runes used to abort the whole knowledge-layer transaction).
- `db/`: DuckDB backend — open, close, schema, insert helpers, index
  population (incl. `PopulateFacetText` — per-session facet documents from
  the index's own tables, full + incremental — and the guarded
  `CreateFacetFTSIndex`, built by `index`/`sync` only when facet material
  exists). `knowledge.go` holds the knowledge layer's tables
  (`knowledge_chunks` + `knowledge_embeddings`, created on demand by
  `EnsureKnowledgeSchema` so old index DBs upgrade in place), the guarded
  `CreateKnowledgeFTSIndex`, and the chunk-vector helpers (missing-vectors
  join for budgeted convergence, content-hash-keyed store/query, orphan
  pruning). `embedcache.go` is the content-hash-keyed embedding cache
  (`.rekal/embed-cache.db`, vectors only): rebuilds embed only unseen content;
  a model switch invalidates by key construction
- `embedhttp/`: HTTP embedding client — batched, hard-timeboxed so the
  post-commit hook can never stall; selected over the embedded nomic model
  via config. Two providers: `openai` (default; any OpenAI-compatible
  `/embeddings` server — vLLM/Ollama/TEI, or a gateway; a Cohere Embed model
  here auto-sends `input_type` and client-caps each input at 2048 chars so
  Portkey/Bedrock gateways that 400 before server-side truncate still work)
  and `bedrock` (Amazon Bedrock runtime, Cohere Embed models, bearer API key,
  no SigV4 — asymmetry via Cohere `input_type` not text prefixes)
- `lsa/`: Latent Semantic Analysis embeddings
- `nomic/`: Nomic-embed-text deep semantic embeddings (platform build tags).
  Model loading is isolated in a **single-flight daemon** (`daemon.lock` flock,
  one per store) that loads the model **before** opening its socket — so a
  connectable socket means "ready", and a native model-load crash kills the
  disposable daemon, never the caller. `NewClient(gitRoot, wait)`: recall passes
  `wait=false` (degrade to keyword/LSA now, daemon warms for next call);
  `rekal embed` passes `wait=true` (block for the model, bounded). Cache
  extraction is flock-serialized; spawns are cooldown-rate-limited. This is the
  fix for the concurrent-recall model-load crash
- `skill/`: One Claude Code skill, redesigned from `SOUL.md`'s "The skill"
  tenets around three homes — **function → script, knowledge → rich prose on
  demand, judgment → reasoning**. `skills/rekal/` embeds `SKILL.md` (thin
  route: 4-substrate triage — tree/knowledge/ledger/map — boundary, silence,
  dispatch; trusts reasoning, no profiles), `scripts/` (deterministic data for
  judgment — `route.py` recall gate: INJECT/KNOWLEDGE/SILENCE + digest.
  The episode gate is absolute `confidence` (saturating BM25 — corpus-invariant
  by construction, so a fixed floor is permitted; SOUL.md bans only tuned
  constants that decide). Raw `mass` and the knowledge `score` are reported
  verbatim for the agent to judge — no tuned `low_mass`/`KNOWLEDGE_MIN` bucket
  (both drift per corpus). `map.sh` fresh|watermark; `wiki-gate.sh`), and
  `references/`
  (rich, on demand — `ledger.md` is the one page on reasoning over the past:
  recall/widen/depth-judgment, time-axis, enumeration, whose-fact/premise,
  analytical SQL, decision arcs, provenance; plus `map.md`, `wiki.md`,
  `reference.md`). No corpus profiles or benchmark tuning ship — chat corpora
  route to the ledger by the general boundary (only-record-is-conversation) +
  the reported `mass` signal; the calibrate/profile tooling lives on
  `research/industry-bench`. `init` installs the whole tree (scripts 0755) and
  purges legacy `rekal-*` companion dirs; `clean` removes current + legacy.
  Adding a module = adding a file under `skills/rekal/`; tip must name it.
  Topology diagrams: `docs/design/skill-router.md`.
- `versioncheck/`: Auto-update notification
- `integration_test/`: Integration tests (`//go:build integration`)

### Docs (`docs/`)

- `DEVELOPMENT.md`: Dev process, testing, CI/CD
- `git-transportation.md`: Git transport layer design
- `design/skill-router.md`: Unified skill — substrate triage, progressive
  disclosure, executable gate scripts (mermaid)
- `db/`: Database schema and design
- `research/`: Memory-research program — positioning claim + evidence ladder,
  18-paper literature map, RekalBench spec (self-labeled repo-grounded intent
  recall), local-corpus data plan, literature-derived product roadmap,
  the multi-repo-at-scale + Rekal-usage/effectiveness eval strategy
  (`06-eval-strategy.md`), the working-backwards flagship-paper restructure —
  git-bound memory, tool + skill + router, answer-sufficiency per token
  (`07-paper-restructure.md`), and `paper/` (Typst + PDF + LaTeX of the
  flagship "Why Git Is the Memory Solution for the Agentic Development
  Lifecycle", accepted as [arXiv:2607.14390](https://arxiv.org/abs/2607.14390);
  supersedes v1 "The Commit Is the Label" in git history;
  single-corpus values from `runs/single-corpus/manifest.json`, multi-corpus
  and sufficiency values from `runs/consolidated/manifest.json` —
  anonymized by workload class). The
  runnable harness lives in `scripts/bench/` (corpus card, T1–T3 + T4
  multi-hop label mining, T5 candidate miner, query generation with leakage
  filter + multi-hop validation, system runner incl. weight ablations +
  grep-rank baseline with UUID→ULID sidmap join, scorer with T4 both@10,
  dev-tuned/test-validated weight tuner, `usage_mine.py` for observational
  Rekal-usage/effectiveness signals, `mine_wild.py` for real in-the-wild
  recall — replaying actual queries against the sessions agents drilled into,
  with cross-repo drills flagged — and `run_rung2.py` for LLM-judged answer
  quality with distinct answer/judge models, prompts committed under
  `scripts/bench/prompts/`). The full multi-repo run sequence + the paper's
  data pack is `docs/research/RUN.md`
- `spec/preconditions.md`: Shared checks for all commands
- `spec/command/`: One file per command — checkpoint, clean, index, init, log, push, query, recall, sync

## Development

### Running Tests

```bash
mise run test              # Unit tests only
mise run test:integration  # Integration tests only
mise run test:ci           # All tests (unit + integration) with race detection
mise run test:coverage     # All tests + statement coverage (writes coverage.html)
```

### Linting and Formatting

```bash
mise run fmt           # Format code (gofmt)
mise run lint          # Lint check (gofmt + golangci-lint)
```

### Building

```bash
mise run build         # Build binary with version from git tag
mise run build:all     # Build for all platforms (snapshot)
```

**Cloud agents / fresh containers:** building from a clean clone has two traps —
llama.cpp HEAD won't link (pin tag `b8157`) and the nomic model ships as a
git-LFS pointer (`git lfs pull` or recall silently drops to BM25+LSA). Follow
[`docs/cloud-agent-setup.md`](docs/cloud-agent-setup.md) before building or
judging recall quality; don't repeat the setup mistakes.

### Before Every Commit

```bash
mise run fmt && mise run lint && mise run test:ci
```

### Test Organization

Unit tests (`_test.go` next to source, same package). Always use `t.Parallel()`.

Integration tests (`integration_test/`, `//go:build integration`). Use `TestEnv` pattern — isolated temp git repos per test. Tests public API only. Cannot be parallelized (uses `os.Chdir`).

## Code Patterns

### Error Handling — SilentError

- `root.go` sets `SilenceErrors: true` globally
- Commands return `NewSilentError(err)` when they've already printed a user-friendly message
- For normal errors, return the error directly

```go
if err := EnsureGitRoot(); err != nil {
    cmd.SilenceUsage = true
    fmt.Fprintln(cmd.ErrOrStderr(), err)
    return NewSilentError(err)
}
```

### Shared Preconditions

All commands except `init` and `clean` must call both:
1. `EnsureGitRoot()` — verifies inside a git repo
2. `EnsureInitDone(gitRoot)` — verifies `.rekal/` exists

### Command Structure

```go
func newFooCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "foo",
        Short: "Short description",
        RunE: func(cmd *cobra.Command, args []string) error {
            cmd.SilenceUsage = true
            gitRoot, err := EnsureGitRoot()
            if err != nil {
                fmt.Fprintln(cmd.ErrOrStderr(), err)
                return NewSilentError(err)
            }
            if err := EnsureInitDone(gitRoot); err != nil {
                fmt.Fprintln(cmd.ErrOrStderr(), err)
                return NewSilentError(err)
            }
            // command logic
            return nil
        },
    }
    return cmd
}
```

### CLI Output Voice

From the soul: short sentences, plain words, say what happened, say what to do, stop.

```
rekal: not a git repository (run this inside a project)
rekal: captured 3 sessions, 847 turns
rekal: no sessions match "JWT expiry" in src/auth/
```

No exclamation marks. No emoji. No "oops."

## Go Code Style

- Write lint-compliant Go code on the first attempt
- Follow standard Go idioms: proper error handling, no unused variables/imports
- Handle all errors explicitly
- Reference `.golangci.yaml` for enabled linters (govet, errcheck, ineffassign, staticcheck, unused)

## Release Process

1. Ensure main is green (CI, Lint, License Check)
2. Tag and push:
   ```bash
   git tag v0.x.y
   git push origin v0.x.y
   ```
3. Release workflow validates then publishes via GoReleaser
