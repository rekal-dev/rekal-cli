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
  index DB, call the `search` package, marshal JSON. The ranking engine itself
  lives in `search/`.
- `checkpoint.go`: Capture session after commit
- `push.go`: Push data to remote branch (wire encode/commit lives in `transport/`)
- `sync.go`: Sync team context (wire decode/import lives in `transport/`)
- `init.go`: Bootstrap Rekal in a git repo
- `clean.go`: Remove Rekal setup — completely, no residue
- `index_cmd.go`: Rebuild index DB from data DB. Also carries the cross-repo
  local-import flags (`--include-all`/`--include`/`--no-local`), which set a
  persistent preference and rebuild
- `config.go`: `.rekal/config.json` (gitignored, never committed) — Rekal's
  durable local config. Holds the cross-repo `local_import` preference, the
  recall-tuning `weights` (BM25/LSA/nomic layer mix, steering boost, subagent
  discount — applied at query time, no reindex), and the `embedding` section
  (OpenAI-compatible HTTP backend: endpoint/model/api_key with `$VAR`
  expansion and `api_key_env`)
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
- `search/`: Recall ranking engine — hybrid BM25 + LSA + Nomic scoring with
  configurable weights (`weights.go`; query-time only), signal weighting
  (steering-turn boost, subagent down-weight), conversation grouping
  (see `docs/agent-metadata.md`), snippet extraction, and the LSA
  query-projection cache (`projection.go`)
- `session/`: Claude Code `.jsonl` parsing — extract turns, tool calls, deduplicate.
  `local.go` enumerates/resolves project session dirs under `~/.claude/projects/*`
  for the cross-repo local import
- `scrub/`: Redact secrets, anonymize file paths, and guarantee valid UTF-8 (`SanitizeText`) in a session payload before any DB insert (runs in `checkpoint` and cross-repo import after parse). DuckDB rejects invalid-UTF-8 VARCHAR binds, so this is the last-line guard against `could not bind parameter`.
- `db/`: DuckDB backend — open, close, schema, insert helpers, index population.
  `embedcache.go` is the content-hash-keyed embedding cache
  (`.rekal/embed-cache.db`, vectors only): rebuilds embed only unseen content;
  a model switch invalidates by key construction
- `embedhttp/`: OpenAI-compatible HTTP embedding client (vLLM/Ollama/TEI) —
  batched, hard-timeboxed so the post-commit hook can never stall; selected
  over the embedded nomic model via config
- `lsa/`: Latent Semantic Analysis embeddings
- `nomic/`: Nomic-embed-text deep semantic embeddings (platform build tags)
- `skill/`: Rekal Skill definition for Claude Code integration
- `versioncheck/`: Auto-update notification
- `integration_test/`: Integration tests (`//go:build integration`)

### Docs (`docs/`)

- `DEVELOPMENT.md`: Dev process, testing, CI/CD
- `git-transportation.md`: Git transport layer design
- `db/`: Database schema and design
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
