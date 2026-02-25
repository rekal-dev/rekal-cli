# Rekal CLI

This repo contains the CLI for Rekal — gives your agent precise memory.

## Architecture

- CLI built with `github.com/spf13/cobra`
- Storage: DuckDB via `github.com/marcboeker/go-duckdb` (`database/sql` interface)
- Build tool: mise, go modules
- Linting: golangci-lint v2

## Key Directories

### Commands (`cmd/rekal/`)
- `main.go`: Entry point
- `cli/`: Core CLI package
  - `root.go`: Root command (recall) + command registration
  - `errors.go`: SilentError pattern for clean error output
  - `preconditions.go`: Shared checks (git repo, init done, index exists)
  - `init.go`: Bootstrap Rekal in a git repo
  - `clean.go`: Remove Rekal setup (local only)
  - `checkpoint.go`: Capture session after commit
  - `push.go`: Push data to remote branch
  - `index_cmd.go`: Rebuild index DB
  - `log.go`: Show recent checkpoints
  - `query.go`: Raw SQL access
  - `sync.go`: Sync team context
  - `version.go`: Version constant (set via ldflags)
  - `versioncheck/`: Auto-update notification
  - `db/`: DuckDB backend (open, close, schema)
  - `integration_test/`: Integration tests (`//go:build integration`)

### Specifications (`docs/spec/`)
- `preconditions.md`: Shared checks for all commands
- `command/`: One file per command with full behavior spec

### Documentation (`docs/`)
- `DEVELOPMENT.md`: Dev process, testing, CI/CD

## Tech Stack

- Language: Go 1.25.x
- Build tool: mise, go modules
- Linting: golangci-lint v2
- Storage: DuckDB
- CLI framework: Cobra

## Development

### Running Tests
```bash
mise run test              # Unit tests only
mise run test:integration  # Integration tests only
mise run test:ci           # All tests (unit + integration) with race detection
```

Integration tests use the `//go:build integration` build tag and are located in `cmd/rekal/cli/integration_test/`.

### Linting and Formatting
```bash
mise run fmt           # Format code (gofmt)
mise run lint          # Lint check (gofmt + golangci-lint)
```

### Before Every Commit (REQUIRED)

**CI will fail if you skip these steps:**

```bash
mise run fmt           # Format code
mise run lint          # Lint check
mise run test:ci       # Run all tests with race detection
```

Or combined: `mise run fmt && mise run lint && mise run test:ci`

### Building
```bash
mise run build         # Build binary with version from git tag
```

### Test Organization

**Unit tests** (`_test.go` next to source, same package):
- `errors_test.go` — SilentError type behavior
- `preconditions_test.go` — git root and init checks
- `version_test.go` — version constant
- `db/db_test.go` — DuckDB connectivity and schema

**Integration tests** (`integration_test/`, `//go:build integration`):
- `commands_test.go` — full command flows (init, clean, stubs) using `TestEnv`
- Uses `TestEnv` pattern: creates isolated temp git repos per test
- Tests public API only (separate package `integration`)

### Test Parallelization

**Always use `t.Parallel()` in unit tests.** Integration tests that use `os.Chdir` cannot be parallelized.

```go
func TestFeature_Foo(t *testing.T) {
    t.Parallel()
    // ...
}
```

## Code Patterns

### Error Handling

The CLI uses the SilentError pattern to avoid duplicated error output.

**How it works:**
- `root.go` sets `SilenceErrors: true` globally — Cobra never prints errors
- `Run()` in `root.go` prints errors to stderr, unless the error is a `SilentError`
- Commands return `NewSilentError(err)` when they've already printed a custom message

**When to use `SilentError`:**
Use `NewSilentError()` when you print a user-friendly message before returning:

```go
if err := EnsureGitRoot(); err != nil {
    cmd.SilenceUsage = true
    fmt.Fprintln(cmd.ErrOrStderr(), err)
    return NewSilentError(err)
}
```

**When NOT to use `SilentError`:**
For normal errors where the default message is fine, return the error directly.

### Shared Preconditions

All commands except `init` and `clean` must call both:
1. `EnsureGitRoot()` — verifies we are inside a git repo
2. `EnsureInitDone(gitRoot)` — verifies `.rekal/` exists with expected layout

`init` calls only `EnsureGitRoot()`. `clean` calls only `EnsureGitRoot()`.

### DuckDB Backend

Two databases in `.rekal/`:
- `data.db` — source of truth (sessions, checkpoints, files_touched, checkpoint_sessions)
- `index.db` — derived index (turns_ft, tool_calls_index, files_index, session_facets, file_cooccurrence)

Use the `db` package to open connections:
```go
dataDB, err := db.OpenData(gitRoot)
defer dataDB.Close()
```

### Command Structure

Follow this pattern for new commands:

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

## Go Code Style

- Write lint-compliant Go code on the first attempt
- Follow standard Go idioms: proper error handling, no unused variables/imports
- Handle all errors explicitly
- Reference `.golangci.yaml` for enabled linters

## Release Process

1. Ensure main is green (CI, Lint, License Check)
2. Create and push a version tag:
   ```bash
   git tag v0.x.y
   git push origin v0.x.y
   ```
3. Release workflow validates then publishes via GoReleaser
