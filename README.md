# Rekal

> **Status: Pre-release** — Actively working towards beta. Core scaffolding is in place; most commands are stubs being implemented milestone by milestone. Expect breaking changes.

Every meaningful software change begins with a conversation — a developer explores a problem, debates approaches, hits dead ends, and lands on a solution. Then they commit 12 lines of code. The conversation disappears. The code remains, stripped of all the reasoning that produced it.

**Rekal captures that conversation and makes it a permanent, queryable part of your project's history.** It hooks into git, stores AI session context in a version-controlled database, and lets any developer — or any AI agent — retrieve the *why* behind every line of code.

Rekal is not a tool for humans to browse history. **Rekal gives your AI agent precise memory — the exact context it needs for the file it is currently working on.** The agent starts every session knowing why the code looks the way it does.

**What makes Rekal different:**

- **Team-shared memory** — `rekal push` and `rekal sync` share session context across your entire team through git. Every developer's AI agent benefits from every other developer's prior sessions.
- **Immutable by design** — Session snapshots are append-only. Content-hash deduplication means two developers always write to disjoint rows — merge conflicts are structurally impossible.
- **Signal, not bulk** — A 2-10 MB session file becomes a ~10 KB payload. Only conversation turns, tool sequences, and actor metadata are stored. A full year of team context is ~200 KB on the remote.
- **Git-native** — No external infrastructure. Rekal data lives on standard git branches, syncs through your existing remote, and uses git's object store for point-in-time recovery.

## Table of Contents

- [How It Works](#how-it-works)
- [Quick Start](#quick-start)
- [Commands Reference](#commands-reference)
- [Typical Workflow](#typical-workflow)
- [Architecture](#architecture)
- [Development](#development)
- [Getting Help](#getting-help)
- [License](#license)

## How It Works

```
  You code with an AI agent          Rekal captures the session
  ─────────────────────────          ──────────────────────────
  prompt → response → commit   ───►  conversation, tool calls,
                                     reasoning — linked to the commit
```

When you commit, Rekal automatically snapshots your active AI session into a local database. `rekal push` shares it with your team on a per-user orphan branch — your git history stays clean.

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

When a newer release is available, the CLI prints an update notice after each command.

## Commands Reference

| Command | Description | Status |
|---------|-------------|--------|
| `rekal init` | Initialize Rekal in the current git repository | Implemented |
| `rekal clean` | Remove Rekal setup from this repository (local only) | Implemented |
| `rekal version` | Print the CLI version | Implemented |
| `rekal checkpoint` | Capture the current session after a commit | Stub |
| `rekal push` | Push Rekal data to the remote branch | Stub |
| `rekal sync` | Sync team context from remote rekal branches | Stub |
| `rekal index` | Rebuild the index DB from the data DB | Stub |
| `rekal log [--limit N]` | Show recent checkpoints | Stub |
| `rekal query "<sql>" [--index]` | Run raw SQL against the data or index DB | Stub |
| `rekal [filters...] [query]` | Recall — search sessions by content, file, or commit | Stub |

### Recall Filters (root command)

Recall is the primary interface — especially for agents. `rekal <query>` is the root invocation; there is no separate `search` subcommand.

| Flag | Description |
|------|-------------|
| `--file <regex>` | Filter by file path (regex, git-root-relative) |
| `--commit <sha>` | Filter by git commit SHA |
| `--checkpoint <ref>` | Query as of a checkpoint ref on the rekal branch |
| `--author <email>` | Filter by author email |
| `--actor <human\|agent>` | Filter by actor type |
| `-n`, `--limit <n>` | Max results (0 = no limit) |

### Examples

```bash
rekal init                              # Set up Rekal in your repo
rekal checkpoint                        # Capture current session
rekal push                              # Share context with the team
rekal sync                              # Pull team context
rekal log                               # Show recent checkpoints
rekal "JWT expiry"                      # Recall sessions mentioning JWT
rekal --file src/auth/ "token refresh"  # Recall with file filter
rekal --actor agent "migration"         # Show only agent-initiated sessions
rekal query "SELECT * FROM sessions LIMIT 5"
rekal clean                             # Remove Rekal from this repo
```

## Typical Workflow

```bash
# 1. Enable Rekal in your project
rekal init

# 2. Work normally — write code with your AI agent, commit as usual.
#    Rekal hooks into post-commit to capture sessions automatically.

# 3. Share your session context
rekal push

# 4. Pull your team's context
rekal sync

# 5. Your agent recalls prior decisions on the files it touches
rekal --file src/billing/ "why discount logic"
```

## Architecture

Rekal uses two local databases with distinct roles:

- **Data DB** (`.rekal/data.db`) — Append-only shared truth. Session snapshots and checkpoint links. Pushed to git as a SQL dump via `rekal push`.
- **Index DB** (`.rekal/index.db`) — Local-only search intelligence. Full-text indexes, vector embeddings, file co-occurrence graphs. Never synced. Rebuild anytime with `rekal index`.

The data DB can be recovered from any point in time using git:

```bash
git show rekal/alice@example.com~10:rekal_dump.sql | duckdb .rekal/data.db
```

## Development

Uses [mise](https://mise.jdx.dev/) for tools and tasks.

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

## Getting Help

```bash
rekal --help              # General help
rekal <command> --help    # Command-specific help
```

- **Issues:** [github.com/rekal-dev/cli/issues](https://github.com/rekal-dev/cli/issues)

## License

Apache-2.0 — see [LICENSE](LICENSE).
