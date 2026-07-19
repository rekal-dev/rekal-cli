# Rekal CLI — Development Process

This document describes the development logistics, testing, and CI/CD for the Rekal CLI.

---

## 1. Logistics

### 1.1 Repository layout

```
rekal-cli/
├── cmd/rekal/                 # Entrypoint and CLI
│   ├── main.go
│   └── cli/                   # Cobra commands and helpers
│       ├── root.go            # Root command (recall) + registration
│       ├── errors.go          # SilentError pattern
│       ├── preconditions.go   # Shared checks (git root, init)
│       ├── init.go            # rekal init
│       ├── clean.go           # rekal clean
│       ├── checkpoint.go      # rekal checkpoint (stub)
│       ├── push.go            # rekal push (stub)
│       ├── index_cmd.go       # rekal index (stub)
│       ├── log.go             # rekal log (stub)
│       ├── query.go           # rekal query (stub)
│       ├── sync.go            # rekal sync (stub)
│       ├── version.go         # Version constant
│       ├── versioncheck/      # Auto-update notification
│       ├── db/                # DuckDB backend (open, close, schema)
│       ├── lsa/               # LSA vector embeddings
│       ├── nomic/             # Nomic deep semantic embeddings (CGO + llama.cpp)
│       └── integration_test/  # Integration tests (//go:build integration)
├── docs/                      # Dev docs and specs
│   ├── DEVELOPMENT.md         # This file
│   └── spec/                  # Command specifications
│       ├── preconditions.md
│       └── command/           # One file per command
├── scripts/                   # Install script, git hooks
├── .github/workflows/         # CI, Lint, License, Release
├── go.mod, go.sum
├── mise.toml                  # Tool versions and tasks
├── .golangci.yaml             # Linter config
└── CLAUDE.md                  # Agent development guide
```

### 1.2 Tooling

- **Go**: version from `go.mod` (currently 1.25.6). Use the same minor for local and CI.
- **mise**: [mise](https://mise.jdx.dev/) manages Go and golangci-lint and runs tasks. Install mise, then in the repo run `mise install`.
- **golangci-lint**: v2.8.0 (configured in `mise.toml` and `.golangci.yaml`). Used for lint only; formatting is done with `gofmt`.

All commands below assume you are in the repo root unless noted.

### 1.3 One-time setup

```bash
# Install Git LFS (required — the nomic model is stored via LFS)
# macOS: brew install git-lfs
# Linux: apt install git-lfs
git lfs install

git clone https://github.com/rekal-dev/rekal-cli.git rekal-cli
cd rekal-cli
mise install                    # Install Go and golangci-lint per mise.toml
./scripts/install-hooks.sh      # Optional: run test + lint before every git push
```

#### Building llama.cpp (required for nomic embeddings)

The nomic embedding package uses CGO bindings to llama.cpp. On supported platforms (darwin/arm64, linux/amd64), you need to build the native libraries once:

```bash
# Prerequisites: cmake, clang/gcc
# macOS: brew install cmake
# Linux: apt install cmake build-essential

# Clone llama.cpp into .deps/ — PIN the tag CI uses. llama.cpp HEAD has
# relocated/renamed the `common` static lib, which breaks the `-lcommon` link
# in nomic_cgo.go. Tag b8157 builds libcommon.a where the CGO flags expect it.
git clone --depth 1 --branch b8157 https://github.com/ggml-org/llama.cpp .deps/llama.cpp

# Build static libraries
cd .deps/llama.cpp
cmake -B build \
  -DLLAMA_METAL=ON \          # macOS only — omit on Linux
  -DLLAMA_BUILD_TESTS=OFF \
  -DLLAMA_BUILD_EXAMPLES=OFF \
  -DLLAMA_BUILD_SERVER=OFF \
  -DBUILD_SHARED_LIBS=OFF \
  -DCMAKE_BUILD_TYPE=Release
cmake --build build --config Release -j$(nproc) --target llama --target ggml --target common
cd ../..
```

On Linux without GPU, omit `-DLLAMA_METAL=ON`. The nomic package gracefully degrades on unsupported platforms — if the build isn't available, embeddings are skipped and search falls back to BM25 + LSA.

#### Downloading the nomic model

The Q8_0 GGUF model (~134MB gzipped) is embedded in the binary via `//go:embed`. It should already be present at `cmd/rekal/cli/nomic/models/nomic-embed-text-v1.5.Q8_0.gguf.gz`. If missing:

```bash
curl -L -o /tmp/nomic.gguf \
  "https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q8_0.gguf"
gzip -9 /tmp/nomic.gguf
mv /tmp/nomic.gguf.gz cmd/rekal/cli/nomic/models/nomic-embed-text-v1.5.Q8_0.gguf.gz
```

### 1.4 Daily workflow

| What you do | Command / check |
|-------------|------------------|
| Format code | `mise run fmt` |
| Run unit tests | `mise run test` |
| Run integration tests | `mise run test:integration` |
| Run all tests (CI-style) | `mise run test:ci` |
| Coverage report | `mise run test:coverage` (total + `coverage.html` line view) |
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

1. Runs `mise run test:ci` (or `go test -tags=integration -race ./...` if mise is not available).
2. Runs `mise run lint` (or gofmt + golangci-lint if no mise).

If either step fails, the push is aborted. Remove the hook by deleting `.git/hooks/pre-push`.

---

## 2. Testing

### 2.1 Test categories

| Category | Location | Build tag | Command |
|----------|----------|-----------|---------|
| **Unit tests** | Next to source (`*_test.go`) | None | `mise run test` |
| **Integration tests** | `cmd/rekal/cli/integration_test/` | `//go:build integration` | `mise run test:integration` |

### 2.2 Unit tests

- Sit next to the code they test, in `_test.go` files in the same package.
- Package name is the same as the production package (no `_test` suffix).
- Test isolated functions: error types, precondition checks, DB connectivity.
- **Always use `t.Parallel()`** for unit tests.
- Keep tests fast and deterministic; no external network or heavy I/O.

### 2.3 Integration tests

- Located in `cmd/rekal/cli/integration_test/` with `//go:build integration` tag.
- Separate package (`integration`) — tests the public API only.
- Use `TestEnv` pattern: creates isolated temp git repos per test.
- Test full command flows (init, clean, preconditions, stubs).
- Skipped by `go test ./...` (need `-tags=integration`).

### 2.4 Running tests

```bash
mise run test              # Unit tests only (go test ./...)
mise run test:integration  # Integration tests only
mise run test:ci           # All tests + race detection (CI-style)
```

Without mise:

```bash
go test ./...                                              # Unit tests
go test -tags=integration ./cmd/rekal/cli/integration_test/... # Integration
go test -tags=integration -race ./...                      # All + race
```

### 2.5 Race detector

CI and the pre-push hook run tests with the race detector. Run the same locally with `mise run test:ci` before pushing.

---

## 3. CI/CD process

### 3.1 Workflows overview

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| **CI** | Push to `main`, PRs, `workflow_dispatch` | Run all tests (unit + integration) with race detector |
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
- **Job**: `lint` — checkout, setup Go, mise, then `mise run lint` and `golangci-lint-action`.
- Linters enabled in `.golangci.yaml`: govet, errcheck, ineffassign, staticcheck, unused.

### 3.4 License Check workflow

- **File**: `.github/workflows/license-check.yml`
- **Job**: `check-license` — verify LICENSE exists and is Apache-2.0.

### 3.5 Release workflow

- **File**: `.github/workflows/release.yml`
- **Trigger**: Push of a tag matching `v*` (e.g. `v0.0.4`).
- **Gate**: The `validate` job runs first: `mise run test:ci` and `mise run lint`.
- **Release job**: GoReleaser builds Linux/amd64 binary (DuckDB requires CGO; cross-compilation TBD).

### 3.6 Cutting a release

1. Ensure `main` is green (CI, Lint, License Check).
2. Create and push a version tag:
   ```bash
   git tag v0.x.y
   git push origin v0.x.y
   ```
3. The Release workflow runs: validate (test:ci + lint), then release (GoReleaser).

---

## 4. Quick reference

| Task | Command |
|------|---------|
| Install tools | `mise install` |
| Format | `mise run fmt` |
| Unit tests | `mise run test` |
| Integration tests | `mise run test:integration` |
| All tests (CI) | `mise run test:ci` |
| Lint | `mise run lint` |
| Build | `mise run build` |
| Install pre-push hook | `./scripts/install-hooks.sh` |
