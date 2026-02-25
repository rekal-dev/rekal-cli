# Rekal CLI — Development Process

This document describes the development logistics, unit testing, and CI/CD for the Rekal CLI. Integration and e2e tests are not yet in scope; only unit tests are documented.

---

## 1. Logistics

### 1.1 Repository layout

```
rekal-cli/
├── cmd/rekal/           # Entrypoint and CLI
│   ├── main.go
│   └── cli/             # Cobra commands and helpers
├── scripts/             # Install script, git hooks
├── .github/workflows/   # CI, Lint, License, Release
├── docs/                # This and other dev docs
├── go.mod, go.sum
├── mise.toml            # Tool versions and tasks
└── .golangci.yaml       # Linter config
```

### 1.2 Tooling

- **Go**: version from `go.mod` (currently 1.22). Use the same minor for local and CI.
- **mise**: [mise](https://mise.jdx.dev/) manages Go and golangci-lint and runs tasks. Install mise, then in the repo run `mise install`.
- **golangci-lint**: v2.0.0 (configured in `mise.toml` and `.golangci.yaml`). Used for lint only; formatting is done with `gofmt`.

All commands below assume you are in the repo root unless noted.

### 1.3 One-time setup

```bash
git clone https://github.com/rekal-dev/cli.git rekal-cli
cd rekal-cli
mise install                    # Install Go and golangci-lint per mise.toml
./scripts/install-hooks.sh      # Optional: run test + lint before every git push
```

### 1.4 Daily workflow

| What you do | Command / check |
|-------------|------------------|
| Format code | `mise run fmt` |
| Run unit tests | `mise run test` or `mise run test:ci` (with race) |
| Run linters | `mise run lint` |
| Build binary | `mise run build` |
| Before push | Pre-push hook runs `test:ci` + `lint` if you ran `install-hooks.sh` |

Recommended before committing: run `mise run fmt`, then `mise run test:ci`, then `mise run lint`. The pre-push hook repeats test:ci and lint so push will fail if they fail.

### 1.5 Pre-push hook (optional)

To run the same checks as CI before every `git push`:

```bash
./scripts/install-hooks.sh
```

This installs `.git/hooks/pre-push`, which:

1. Runs `mise run test:ci` (or `go test -race ./...` if mise is not available).
2. Runs `mise run lint` (or gofmt + golangci-lint if no mise).

If either step fails, the push is aborted. Remove the hook by deleting `.git/hooks/pre-push`.

---

## 2. Unit testing

We currently have **unit tests only**. Integration and e2e tests are not yet in place.

### 2.1 How to run tests

```bash
mise run test        # go test ./...
mise run test:ci    # go test -race ./...  (same as CI)
```

Without mise:

```bash
go test ./...
go test -race ./...
```

### 2.2 Where tests live

- Tests sit next to the code they test, in `_test.go` files in the same package (e.g. `version_test.go` next to `version.go` in `cmd/rekal/cli/`).
- Package name is the same as the production package (no `_test` suffix), so tests can touch unexported symbols when needed.

### 2.3 Conventions

- Use `testing.T` and standard table-driven or single-case tests.
- Prefer `t.Parallel()` for tests that do not share global state.
- Keep tests fast and deterministic; no external network or heavy I/O.

### 2.4 Race detector

CI and the pre-push hook run tests with the race detector (`go test -race ./...`). Run the same locally with `mise run test:ci` before pushing.

---

## 3. CI/CD process

### 3.1 Workflows overview

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| **CI** | Push to `main`, PRs, `workflow_dispatch` | Run unit tests with race detector |
| **Lint** | Push to `main`, PRs, `workflow_dispatch` | gofmt check + golangci-lint |
| **License Check** | Push to `main`, PRs, `workflow_dispatch` | Ensure LICENSE exists and is Apache-2.0 |
| **Release** | Push of tag `v*`, `workflow_dispatch` | Validate then build artifacts and publish release |

All workflows use `ubuntu-latest`. CI and Lint use [jdx/mise-action@v3](https://github.com/jdx/mise-action) so the same tools and tasks as local (mise.toml) run in CI.

### 3.2 CI workflow

- **File**: `.github/workflows/ci.yml`
- **Job**: `test` — checkout, mise install, `mise run test:ci`.
- No release or artifact upload; only validation.

### 3.3 Lint workflow

- **File**: `.github/workflows/lint.yml`
- **Job**: `lint` — checkout (PR head or push SHA), setup Go, mise, then:
  1. `mise run lint` (gofmt -l -s then golangci-lint).
  2. `golangci/golangci-lint-action@v7` with version v2.0.0 for inline feedback on PRs.
- Linters enabled in `.golangci.yaml`: govet, errcheck, ineffassign, staticcheck, unused. Formatting is enforced by the shell step (gofmt), not the linter.

### 3.4 License Check workflow

- **File**: `.github/workflows/license-check.yml`
- **Job**: `check-license` — checkout, verify LICENSE exists and that its first lines indicate Apache License (Apache-2.0).

### 3.5 Release workflow

- **File**: `.github/workflows/release.yml`
- **Trigger**: Push of a tag matching `v*` (e.g. `v0.1.0`), or manual `workflow_dispatch`.
- **Gate**: The `validate` job runs first: `mise run test:ci` and `mise run lint`. The `release` job runs only if `validate` succeeds.
- **Release job**: Full checkout (fetch-depth: 0 for GoReleaser), mise, GitHub App token for the Homebrew tap repo, then `goreleaser release --clean`. Binaries and artifacts are published to the GitHub release for the tag; Homebrew tap is updated when configured.

### 3.6 Cutting a release

1. Ensure `main` is green (CI, Lint, License Check).
2. Create and push a version tag:
   ```bash
   git tag v0.x.y
   git push origin v0.x.y
   ```
3. The Release workflow runs: validate (test:ci + lint), then release (GoReleaser). Fix any failing job before re-tagging if needed.

---

## 4. Quick reference

| Task | Command |
|------|---------|
| Install tools | `mise install` |
| Format | `mise run fmt` |
| Unit tests | `mise run test` |
| Unit tests (race) | `mise run test:ci` |
| Lint | `mise run lint` |
| Build | `mise run build` |
| Install pre-push hook | `./scripts/install-hooks.sh` |
