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
  package. First runs `maybeRefreshStaleSkill` (init.go): the agent enters here
  after loading the skill, so a skill left behind by a binary upgrade
  (version-pinned marker mismatch) is refreshed in place — best-effort, bench-
  gated, touches only the gitignored `.claude/skills/`.
  **Default output is the seed digest** (`digest.go`); `--json` gives
  raw structured results. The ranking engine itself lives in `search/`.
  Auto-widening recall: `deriveFramings` derives a bounded set (≤`maxFramings`)
  of deterministic reformulations of the query (keyword-only, clause splits,
  temporal variant — general linguistics, no corpus tuning) and RRF-fuses their
  result lists (`fuseFramings`, k=60, conf=max-per-session) into one seed. The
  original query is always framing v0, so fusion only adds/reorders hits; a
  query that yields no reformulation runs as a single search (byte-identical).
  Also the L1 recall-graph seam (`docs/design/recall-graph.md`): reads each
  seed's reach hint from `session_reach` (`attachReach`) **before** spooling
  this call's own surfaced edges (`logRecallEdges` → `graph.Append`), so a
  session never inflates its own count.
- `digest.go`: in-binary port of the old route.py — turns a `search.Output` into
  the seed digest (INJECT/KNOWLEDGE/SILENCE + per-seed `conf=`), byte-identical
  (golden-tested). Super-low env-overridable floor (`REKAL_HUNT_*`),
  recommendation not decision; SILENCE exits 1. Per-seed `reachHint` suffix
  (`[reached N×· "query"]`) is empty for unreached seeds, so cold-store output
  stays byte-identical.
- `view.go`: in-binary port of the old view.py — `viewSession` (drill →
  readable turns) and `viewRows` (SQL → TSV). The default query output;
  `--json` gives raw. Session view is golden-tested byte-identical
- `find.go`: `rekal find "<term>" [role]` — complete, time-ordered enumeration
  sweep over `turns` (port of find.py, diff-identical)
- `knowledge_index.go`: Knowledge-layer build/refresh — chunk the repo's
  tracked prose files at HEAD into `index.db` (`knowledge_chunks`), diffing
  stored git blob SHAs against `git ls-tree -r HEAD` so only changed files
  re-chunk; commit-SHA watermark (`knowledge_head_sha`) makes the steady
  state one rev-parse. Called by `index` (full) and recall (incremental,
  best-effort). See `docs/design/knowledge-layer.md`
- `checkpoint.go`: Capture session after commit — **no-op while a rebase is in
  progress** (`gitx.RebaseInProgress`, which reads git's `rebase-merge`/
  `rebase-apply` state dirs; `GIT_REFLOG_ACTION` is empty in the post-commit
  environment and cannot be used). git fires post-commit for every replayed
  commit, so a ten-commit rebase would run ten captures, and any whose transcript
  grew mid-rebase would link the live session to commits it never produced —
  false `checkpoint_sessions` edges, the commit↔session ground truth the
  benchmark labels itself from. `--amend`/cherry-pick are deliberately not
  covered: those are real authoring moments. A re-captured transcript
  **appends** to its existing session instead of storing the conversation again:
  a live conversation has different content at every commit, so keying dedup on
  content made each commit a brand-new session (measured 2.07× amplification and
  duplicate seeds crowding out recall). The transcript→session mapping lives in
  `checkpoint_state.session_id` — **local-only, never wired**; `sessions.session_hash`
  keeps its content-hash meaning because `local_import.go` dedups by comparing
  `session.ContentHash` against it, and overloading it would silently re-import
  every session this repo already captured. Appends only turns/tool-calls past
  `db.SessionExtent`, gated by `turnsExtend`: the parsed transcript must be the
  stored turns plus more, else capture falls back to a new session, so a
  rewritten or truncated transcript can never splice unrelated turns into
  history. Subagent parent lookup prefers the same mapping, falling back to the
  trunk's content hash for trunks captured before it. Old stores gain the column
  by migration and pay one extra session row at the boundary, which the index
  supersession pass then collapses. Also drains the L1 recall-graph
  spool into `data.db.recall_edges` (`drainRecallSpool`) while the data.db
  writer is held — even when no new session is captured (an agent may recall/
  drill inside an already-checkpointed session), refreshing `session_reach` via
  `db.RefreshSessionReach` on that path so the graph never stalls; otherwise the
  incremental index refresh rebuilds `session_reach`
- `push.go`: Push data to remote branch (wire encode/commit lives in `transport/`)
- `sync.go`: Sync team context (wire decode/import lives in `transport/`).
  `indexSessionFrame` dedups an arriving session by **keeping the longest**: one
  conversation spanning several commits links to several checkpoints and rides in
  each one's frame, so the repeat would violate `session_facets`' primary key and
  abort the whole import. Frames from one export carry identical turns, but
  frames from *different pushes* do not — the author kept talking between them —
  so skipping the repeat would strand the reader on a truncated conversation;
  a longer arrival replaces the indexed rows. Could not arise before capture
  learned to append, when every checkpoint carried a distinct session
- `init.go`: Bootstrap Rekal in a git repo — store, hooks, orphan branch,
  skill (tip + scripts + references), and one marker-tagged CLAUDE.md sentence
  (the whole DX: init, done; `clean` removes the line, refresh replaces it in
  place). `installSkill` pins the binary `Version` into each installed skill
  dir (`.claude/skills/<name>/.rekal-version`); `maybeRefreshStaleSkill` (called
  from recall) reads it and re-installs the skill when it's behind the running
  binary, so an upgrade reaches the repo without a manual re-init. Re-running
  `rekal init` on an already-initialized repo calls `refreshManaged` (skill +
  hooks + CLAUDE.md line + agent rules + gitignore); the recall-time
  auto-refresh deliberately does **only** the skill (gitignored), never the
  tracked/side-effectful assets. `installAgentInstructions` covers the
  non-Claude agents: it detects which are installed on the machine (home-dir
  probe — `~/.codex`, `~/.local/share/opencode`, `~/.cursor`, `~/.gemini`,
  `~/.copilot`, `~/.kiro`) and writes the marker-tagged `rekalAgentLine` into
  the file each reads — `AGENTS.md` (Codex/OpenCode/Cursor, written once),
  `GEMINI.md` (Gemini), `.github/copilot-instructions.md` (Copilot),
  `.kiro/steering/rekal.md` (Kiro) — via the generalized
  `ensureManagedLine`. Created-if-missing, replace-in-place on refresh, user
  content preserved; a file Rekal **newly creates** is gitignored (per-machine
  detection → local-only), while a file the user already tracked stays tracked
  with only the marker line injected
- `clean.go`: Remove Rekal setup — completely, no residue. `removeManagedLines`
  strips the marker line from CLAUDE.md and every detected-agent file (AGENTS.md
  / GEMINI.md / .github/copilot-instructions.md / .kiro/steering/rekal.md),
  deleting a file that was ours and pruning emptied `.github` / `.kiro/steering`
  / `.kiro`
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
  summary boost, subagent discount, facet boost, plus the opt-in
  `recency_boost` and `reach_boost` additive layers — recency over
  `session_facets.captured_at` (default 0.15, inert when candidates share a
  timestamp), reach over the L1 `session_reach` graph (default 0.2,
  self-activating — byte-identical until the graph has edges); both
  ranking-only, never the silence gate, `0` disables — applied at query time,
  no reindex), and
  the `embedding` section (OpenAI-compatible HTTP backend: endpoint/model/
  api_key with `$VAR` expansion and `api_key_env`; a Cohere Embed model under
  the `openai` provider auto-sends `input_type`)
- `local_import.go`: Cross-repo local session import — folds this machine's
  other local agent sessions (all registered adapters — Claude/Cursor/Codex/Gemini/OpenCode/Copilot/Kiro)
  into the index. **Index-only, never `data.db`**, so imported sessions are
  structurally unpushable to the team; origin-labeled, deduped by content hash
- `log.go`: Show recent checkpoints
- `query.go`: Raw SQL access (explicit `--sql "<stmt>"`; bare positional is
  shorthand; `--sql`/positional/`--session` mutually exclusive) + session drill.
  **Default output is agent-readable text** (`view.go`: readable turns / TSV
  rows); `--json` gives raw JSON/NDJSON. `--help` carries the full queryable
  schema (all tables + columns; FTS-internal/state tables noted as ignore). A
  successful `--session` drill spools an L1 recall-graph drill edge
  (`logDrillEdge` → `graph.Append`) — the strong "this memory was used" signal.
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
  stays local (see `docs/design/merged-only-sharing.md`). An **already-exported**
  checkpoint is shareable unconditionally: it passed the gate once, and a commit
  rebased/squashed away after capture leaves an orphaned `git_sha` that can never
  re-prove (empty cumulative diff → the squash probe fails closed), so
  re-running the gate in `push --re-export` would silently delete already-shared
  conversations from the branch. Export also withholds **superseded** sessions
  (`db.SupersededSessionIDs`) so one conversation is never shipped several times;
  it fails loud rather than shipping the duplicates it exists to prevent
- `gitx/`: Thin git-plumbing helpers (rev-parse, show, hash-object, config,
  `rekal/<email>` branch name, `DefaultBranch`/`IsAncestor`/`IsSquashMergedInto`/
  `BranchTip` for the merged-only export gate, `MainWorktreeRoot` for the
  worktree-shared store) shared by the command and transport layers
- `graph/`: L1 recall citation-graph spool — the lock-free write-ahead buffer
  (`.rekal/recall-log.ndjson`) that keeps query-time edge capture off data.db's
  writer. `Append` (best-effort, no-op under `session.BenchEnv`) on recall/drill;
  `Drain` (rename-to-tmp, partial-line-tolerant) at checkpoint. Transient
  buffer, not the store — the permanent record is `data.db.recall_edges`. See
  `docs/design/recall-graph.md`
- `knowledge/`: Prose-file chunker for the knowledge layer — markdown/plain
  text into heading-anchored sections (breadcrumb trails, 1-indexed line
  ranges, content hashes). Pure functions, no git/DB
- `search/`: Recall ranking engine — hybrid BM25 + LSA + Nomic scoring plus
  the additive facet layer (BM25 over per-session tool paths + command
  prefixes + steering text; `weights.facet_boost`, default 0.3, `0` =
  byte-identical pre-facet ranking; fails soft without a facet FTS index),
  the additive **recency** and **reach** layers (`weights.recency_boost`
  default 0.15 / `reach_boost` default 0.2; recency = min-max over
  `session_facets.captured_at`, reach = max-normalized L1 `session_reach.reach_count`
  via `loadCapturedAt`/`loadReachCounts`; additive before the subagent discount
  like facet, ranking-only — excluded from `absoluteConfidence`. Each
  self-inerts until it has signal — recency contributes 0 when candidates share
  a timestamp, reach is byte-identical on a cold store and fails soft on an
  index without the reach table; `0` disables the lookup),
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
- `session/`: AI-session parsing — extract turns, tool calls, deduplicate. One
  `Adapter` per agent (`adapter.go` registry): `claude`, `cursor`, `codex`,
  `gemini`, `opencode`, `copilot`, `kiro` — each `Discover`s that agent's
  session files for the repo and `Parse`s them into a `SessionPayload`. `kiro.go`
  (`KiroAdapter`) reads **both** Kiro CLI and IDE sessions (Parse dispatches on
  the ref extension). CLI: `$KIRO_HOME/sessions/cli/<id>.json` (metadata:
  `session_id`/`cwd`/`title`/`created_at` — `cwd` gives the exact repo match) +
  `<id>.jsonl` (v3 event log
  `{"kind":"Prompt"|"AssistantMessage","data":{"content":[{"kind":"text","data":"…"}]}}`).
  IDE: the Code-OSS global storage (`$KIRO_IDE_STORAGE` override; else
  macOS `~/Library/Application Support/Kiro`, Linux `~/.config/Kiro`, Windows
  `%APPDATA%/Kiro`) at `User/globalStorage/kiro.kiroagent/workspace-sessions/<ws>/`
  — `sessions.json` index (`workspaceDirectory` repo match, `dateCreated`,
  `hidden`) + `<sessionId>.json` (`{history:[{message:{role,content}}]}`,
  content string or `[{type,text}]`). Both schemas verified against the
  community readers (prabhugr/kiro-cli-history, pajaydev/kiro-history) since
  Kiro's is unpublished; tool blocks have no documented shape so they're
  best-effort/fail-soft. `$KIRO_HOME` defaults to `~/.kiro`.
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
  exists). `PurgeSupersededSessionsFromIndex` collapses re-captures written
  before capture keyed on ref identity: a session whose turns are a strict
  prefix of a longer one's is dropped from the **index only** — data.db keeps
  every copy, since the ledger is append-only and those rows are already on the
  wire. Grouped by source + author + parent + turn 0 so different agents that
  open alike are never merged; runs before facets/reach so neither is computed
  for a session about to be dropped. `knowledge.go` holds the knowledge layer's tables
  (`knowledge_chunks` + `knowledge_embeddings`, created on demand by
  `EnsureKnowledgeSchema` so old index DBs upgrade in place), the guarded
  `CreateKnowledgeFTSIndex`, and the chunk-vector helpers (missing-vectors
  join for budgeted convergence, content-hash-keyed store/query, orphan
  pruning). `reach.go` holds the L1 recall citation graph: the permanent
  append-only `recall_edges` (data.db, **local-only — never wired**, like
  `checkpoint_state`) via `InsertRecallEdges`, and the derived `session_reach`
  aggregate (index.db, created on demand by `EnsureReachSchema`) via
  `PopulateSessionReach` (from `recall_edges` in full + incremental index) and
  `LoadReach` (the hot-path hint read). `embedcache.go` is the
  content-hash-keyed embedding cache
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
  fix for the concurrent-recall model-load crash.
  `embed.c`'s `MAX_TOKENS` (8192, the model's trained window) is the hard
  truncation point: a session is embedded as one concatenated document, so
  anything past the cap is **discarded, not blurred** — and decisions land late
  in a conversation. It was 2048, which meant the neural layer never saw the
  end of any real session. **`MAX_TOKENS` and `ModelName` move together**:
  `session_embeddings` and `embed_cache` are both keyed by model, so bumping
  the window without bumping the name leaves old vectors matching and the
  change is a silent no-op on every existing store (and mixes two vector
  spaces in one column if anything does re-embed). `ModelName` is duplicated
  across the cgo/nocgo build-tag files — change both, and add the retired id to
  `supersededNomicModels` (recall.go) so an index still carrying it warns
  "run `rekal embed`" instead of the misleading "no embedding config is set"
- `skill/`: One Claude Code skill, **scriptless** for retrieval/navigation —
  those moved into the binary as commands (`docs/design/skill-into-command.md`).
  `skills/rekal/` embeds `SKILL.md` (thin route: 4-substrate triage —
  tree/knowledge/ledger/map — boundary, silence, dispatch, judgment; trusts
  reasoning, no profiles; plus the **ledger workflow gate** — a ledger question
  classifies the requested *answer type* and loads exactly one specialist
  workflow, followed by a shared `### Final answer check` contract), `scripts/`
  (only the workflow gates now — `map.sh` fresh|watermark, `wiki-gate.sh`), and
  `references/` (rich, on demand — `ledger.md` is the one page on reasoning over
  the past: recall/widen/depth-judgment, time-axis, enumeration, whose-fact/
  premise, analytical SQL, decision arcs, provenance; plus `map.md`, `wiki.md`,
  `reference.md`, and `references/workflows/` — the five answer-type specialists
  the gate routes to: `duration.md`, `complete-set.md`, `event-time.md`,
  `inference.md`, `point-fact.md`, each a concentrated evidence contract, not
  truth. Ordered/exclusive routing + shipped content are hash-pinned in
  `skill_test.go`). The
  agent uses the commands directly — `rekal "<q>"` (seed digest, `digest.go`,
  auto-widened via `deriveFramings` + RRF), `rekal find`, `rekal query
  --session`/`--sql` (readable text via `view.go`); all default to compact text,
  `--json` for raw (SOUL: agent-first output is text, JSON on opt-in). No corpus
  profiles or benchmark tuning ship. `init` installs the tree (scripts 0755) and
  purges legacy `rekal-*` companion dirs; `clean` removes current + legacy. The
  route/gate scripts (route/view/find/seek/when.py) were removed once the
  commands proved byte-identical (golden/diff-tested); the property harness
  `scripts/skill-permtest.py` now drives `rekal "<q>"` directly.
  This skill ships **only** in the binary. The Claude Code plugin (`plugin/`,
  listed by the repo-root `.claude-plugin/marketplace.json`) is **setup-only**:
  two skills — `/rekal:install` (once per machine) and `/rekal:init` (once per
  repository, the recurring one, and deliberately silent on a repo that merely
  lacks `.rekal/`) — plus `bin/rekal-install`, a byte-identical vendored copy of
  `scripts/install.sh` that puts the installer on the Bash `PATH` instead of
  piping a live URL into a shell (`TestPlugin_VendoredInstaller` pins the copy).
  It carries no recall material — no `skills/rekal`, `references/`, or
  `scripts/`. The split is the point: a plugin tracks `main` while an installed
  binary does not, so shipping the recall skill in both would load two copies at
  once and let the newer one describe flags the user's binary lacks. One owner,
  no divergence — pinned by `TestPlugin_SetupOnly` in `skill_test.go`. The
  plugin versions on its own line (`plugin/.claude-plugin/plugin.json`, not the
  binary's `v0.2.x`) — **bump it whenever the setup skill changes**, or
  self-hosted-marketplace users stay pinned; `claude plugin validate --strict`
  (what the submission pipeline runs) rejects a missing version.
  Topology diagrams: `docs/design/skill-router.md`.
  Distribution + the community-marketplace submission path:
  `docs/design/plugin-distribution.md`.
- `versioncheck/`: Auto-update notification
- `integration_test/`: Integration tests (`//go:build integration`)

### Docs (`docs/`)

- `DEVELOPMENT.md`: Dev process, testing, CI/CD
- `usage.md`: Operational guide the README links out to — two databases,
  orphan branches & merged-work-only sharing, worktrees, the full agent
  command surface + skill, cross-repo recall
- `configuration.md`: `.rekal/config.json` reference — ranking weights,
  embedding backends, API-key handling (the deep content the README's
  Configuration section links to)
- `git-transportation.md`: Git transport layer design
- `design/skill-router.md`: Unified skill — substrate triage, progressive
  disclosure, executable gate scripts (mermaid)
- `design/skill-into-command.md`: Interface-lock design for folding the skill
  scripts into the binary — default-uplift principle (agent form by default,
  `--json` raw), command boundaries (retrieval/navigation/lifecycle, one verb
  each), the skill↔command seam, and the byte-identical / no-performance-change
  migration contract
- `design/recall-graph.md`: L1 recall citation graph — query-time edge capture
  (recall + drill), the lock-free spool → checkpoint drain → `recall_edges`
  (permanent, local-only) → `session_reach` (derived) path, the display-only
  `[reached N×]` hint, and the non-goals (source attribution, authority
  ranking, team-sharing)
- `design/plugin-distribution.md`: Claude Code plugin packaging — why the plugin
  is setup-only (the binary stays the single owner of the recall skill; a plugin
  tracks `main`, an installed binary does not), why the binary cannot ship
  inside it, self-hosted marketplace vs. the community marketplace's
  form-not-PR submission path, and the non-goals
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
- `spec/command/`: One file per command — checkpoint, clean, embed, find, index, init, log, push, query, recall, sync

### Community files (repo root)

- `CONTRIBUTING.md`: What a contribution has to satisfy — SOUL.md's eight
  questions as the review bar, the explicit decline list (server, telemetry,
  required API key, any mutation/forgetting path over `data.db`, corpus-tuned
  constants in skill scripts, a rule where a function would do), the
  fmt/lint/test:ci gate, the doc-sync rule (CLAUDE.md + `--help` +
  `docs/spec/command/` in the same change), test placement, commit convention
  (`type(scope): what changed`), and the binding benchmark honesty rules for
  `scripts/bench/` + `scripts/industry-bench/`
- `SECURITY.md`: Private reporting via GitHub advisories, and the threat model
  that follows from the architecture — in scope is anything putting material on
  the wire that should stay local (redaction/path-anonymization failure, a
  `filterMerged` bypass, cross-repo import reaching `data.db`, `recall_edges`/
  `checkpoint_state`/config on the wire), credential handling, untrusted wire
  frames and transcripts, and local integrity (append-only violations,
  `.rekal/` state permissions). Out of scope: recall quality, `query --sql`,
  what an agent does with what it recalls. Also the secret-in-the-ledger
  remedy: rotate first, branch surgery second, report the redaction gap
- `CODE_OF_CONDUCT.md`: Contributor Covenant 2.1; reports via GitHub *Report
  content* or a maintainer DM in Discord

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

The DuckDB FTS extension is embedded so recall works offline, on the same
three platforms as nomic and the release matrix (linux/amd64, linux/arm64,
darwin/arm64). The linux/amd64 and darwin/arm64 blobs are committed under
`cmd/rekal/cli/db/extensions/`; the linux/arm64 blob is gitignored and
downloaded at build time by `scripts/fetch-fts-extension.sh` (a `mise run
build` dependency, also run in the release workflow), versioned off
go-duckdb's own DuckDB pin. Building **directly** with `go build` on
linux/arm64 needs `mise run fetch-extensions` first; otherwise the
`//go:embed` fails on the missing blob. On any other platform,
`db.LoadFTSExtension` falls back to a one-time network download.

**Cloud agents / fresh containers:** the cold-start is build → init → sync →
verify, and it has three traps — llama.cpp HEAD won't link (pin tag `b8157`),
the nomic model ships as a git-LFS pointer (`git lfs pull` or recall silently
drops to BM25+LSA), and the installed `.claude/skills/rekal/` copy goes stale
after a rebuild. A **released** binary (versioned via ldflags) self-heals: the
first recall after an upgrade sees the pinned-version mismatch and refreshes the
skill in place. A **dev** rebuild keeps `Version="dev"`, so the marker still
matches and the auto-refresh won't fire — re-run `rekal init` to refresh it
(data is untouched). Follow
[`docs/cloud-agent-setup.md`](docs/cloud-agent-setup.md) — it has the exact
steps, the data-sync sequence (`rekal init` + `rekal sync`), the
semantic-layer verification, and the no-mise dev-loop fallbacks. Don't repeat
the setup mistakes, and never judge recall quality before the verify step.

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

Rekal memory is active here — before non-trivial work, use the `rekal` skill and route: grep the tree for present-tense code, `rekal` knowledge for present prose at HEAD, the ledger (`rekal` + gates) for past intent, the map for structure. <!-- managed by rekal -->
