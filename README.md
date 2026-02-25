# Rekal CLI

> **Status: Work in Progress** — Early development. Not ready for production use.

Rekal gives your agent precise memory — the exact context it needs for what it's working on. It hooks into git, stores AI session context in a version-controlled database, and lets any developer — or any AI agent — retrieve and understand the *why* behind every line of code.

## Table of Contents

- [Quick Start](#quick-start)
- [Commands Reference](#commands-reference)
- [Development](#development)
- [License](#license)

## Requirements

- Git
- macOS or Linux

## Quick Start

```bash
# Install
curl -fsSL https://raw.githubusercontent.com/rekal-dev/cli/main/scripts/install.sh | bash

# Or with a specific version
REKAL_VERSION=v0.0.4 curl -fsSL https://raw.githubusercontent.com/rekal-dev/cli/main/scripts/install.sh | bash
```

Install location: `~/.local/bin` (override with `REKAL_INSTALL_DIR`).

```bash
# Initialize in a git repo
cd your-project
rekal init

# Check version
rekal version
```

When a newer release is available, the CLI prints an update message after each command.

## Commands Reference

| Command | Description | Status |
|---------|-------------|--------|
| `rekal init` | Initialize Rekal in the current git repository | Implemented |
| `rekal clean` | Remove Rekal setup from this repository (local only) | Implemented |
| `rekal checkpoint` | Capture the current session after a commit | Stub |
| `rekal push` | Push Rekal data to the remote branch | Stub |
| `rekal index` | Rebuild the index DB from the data DB | Stub |
| `rekal log [--limit N]` | Show recent checkpoints | Stub |
| `rekal query "<sql>" [--index]` | Run raw SQL against the Rekal data model | Stub |
| `rekal sync` | Sync team context from remote rekal branches | Stub |
| `rekal [filters...] [query]` | Search/recall (root command) | Stub |
| `rekal version` | Print the version | Implemented |

### Recall Filters (root command)

| Flag | Description |
|------|-------------|
| `--file <regex>` | Filter by file path (regex, git-root-relative) |
| `--commit <sha>` | Filter by git commit SHA |
| `--checkpoint <ref>` | Query as of checkpoint ref on rekal branch |
| `--author <email>` | Filter by author email |
| `--actor <human\|agent>` | Filter by actor type |
| `-n`, `--limit <n>` | Max results (0 = no limit) |

### Examples

```bash
rekal init                          # Set up Rekal
rekal log                           # Show recent checkpoints
rekal "JWT expiry"                  # Search sessions
rekal --file src/auth/ "JWT"        # Search with file filter
rekal query "SELECT * FROM checkpoints LIMIT 5"
rekal clean                         # Remove Rekal setup
```

## Development

Uses [mise](https://mise.jdx.dev/) for tools and tasks (same pattern as [Entire](https://github.com/entireio/cli)).

```bash
git clone https://github.com/rekal-dev/cli.git rekal-cli
cd rekal-cli
mise install          # Install Go, golangci-lint
```

### Common Tasks

```bash
mise run fmt              # Format code
mise run test             # Run unit tests
mise run test:integration # Run integration tests
mise run test:ci          # Run all tests (unit + integration) with race detection
mise run lint             # Run linters
mise run build            # Build rekal binary
```

**Before committing:** `mise run fmt && mise run lint && mise run test:ci`

Install the pre-push hook to run CI checks locally before each push:

```bash
./scripts/install-hooks.sh
```

See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for full development guide.

### Getting Help

```bash
rekal --help              # General help
rekal <command> --help    # Command-specific help
```

## License

Apache-2.0 — see [LICENSE](LICENSE).
