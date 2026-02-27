# Rekal

> **Beta** — Works with Claude Code. More agents coming.

## Two problems

### Intent has no ledger

Code has git. Every line, every change, every author — recorded forever.

But the reasoning behind the code has nothing. The conversations where a developer and an AI explored a problem, debated approaches, rejected alternatives, arrived at a decision — those vanish the moment the session ends.

The code says *what*. The intent says *why*. The *why* has no permanent record.

### Agents can't remember

An AI agent starts every session blank. It reads the code. It does not know why the code looks the way it does. It does not know what was tried and rejected last week. It does not know that the team already explored and abandoned the approach it is about to suggest.

Humans have institutional memory. Agents have none.

## What Rekal does

Rekal hooks into git and captures your AI session context at every commit. That context becomes a permanent, immutable, shared part of your project history — distributed through git, not through a separate service. When your agent starts a new session, it recalls the precise prior context for the problem it is working on. It knows why the code looks the way it does.

## What makes Rekal different

Rekal is built on beliefs. Those beliefs guide every decision. When a choice conflicts with a belief, the choice loses. That is the difference.

- **Immutable.** The record cannot be edited or deleted. Append-only is what makes the ledger trustworthy.
- **Intent lives next to the code.** Not in a separate system. Not behind someone else's service. In git, next to the code it explains.
- **Thin on the wire, rich on the machine.** Git is the transport and every byte costs. Indexes, embeddings, search — all computed locally.
- **Secure by design.** The data never leaves git and the local machine. No servers. No APIs. No telemetry.
- **Simple.** Single binary. Everything embedded. Nothing to install, nothing to configure, nothing to break.
- **Transparent.** The user sees everything that was created and can remove all of it. No sticky tape.
- **Agent first.** The agent is the consumer. Output format, query interface, context loading — all favor the agent.

The full version: [SOUL.md](SOUL.md).

## Install and uninstall

Install:

```bash
curl -fsSL https://raw.githubusercontent.com/rekal-dev/rekal-cli/main/scripts/install.sh | bash
```

Default location: `~/.local/bin`. Override with `--target <dir>`.

Uninstall:

```bash
rm ~/.local/bin/rekal
```

If you installed to a custom directory, remove the binary from there instead.

## Quick start

Requirements: Git, macOS or Linux.

### Set up

```bash
cd your-project
rekal init
```

`rekal init` creates the following on your system:

- `.rekal/` directory containing `data.db` (shared truth) and `index.db` (local search index)
- A `post-commit` and `pre-push` git hook (marked `# managed by rekal`)
- A Claude Code skill at `.claude/skills/rekal/SKILL.md`
- An orphan branch `rekal/<your-email>` for transport
- Appends `.rekal/` to your `.gitignore`

### Tear down

```bash
rekal clean
```

`rekal clean` removes everything `init` created:

- Deletes the `.rekal/` directory and all its contents
- Removes the git hooks (only the ones marked `# managed by rekal`)

No residue. If you want to start over, run `clean` then `init`.

### Verify

```bash
rekal version
```

When a newer release is available, the CLI prints an update notice after each command.

## How it works

```mermaid
flowchart LR
    subgraph capture ["Capture"]
        A["AI Session"] -->|"rekal checkpoint<br/>(post-commit)"| B[("data.db<br/>append-only")]
    end

    subgraph transport ["Transport"]
        B -->|"rekal push"| C["Wire Format<br/>zstd + varint interning"]
        C -->|"git push<br/>rekal/&lt;email&gt;"| D[("Remote<br/>orphan branch")]
    end

    subgraph index ["Index"]
        B -->|"rekal index"| E[("index.db<br/>local-only")]
        D -->|"rekal sync"| E
        E --- F["BM25 FTS"]
        E --- G["LSA Embeddings"]
        E --- N["Nomic Deep Embeddings"]
        E --- H["Co-occurrence"]
        E --- I["Facets"]
    end

    subgraph query ["Query"]
        J["rekal 'keyword'"] -->|"hybrid search"| E
        E -->|"scored JSON"| K["Agent"]
        K -->|"rekal query<br/>--session &lt;id&gt;"| B
        B -->|"full conversation"| K
    end

    style capture fill:#fff5f5,stroke:#e94560,color:#333
    style transport fill:#f0fdf4,stroke:#22c55e,color:#333
    style index fill:#f0f4ff,stroke:#3b82f6,color:#333
    style query fill:#faf5ff,stroke:#a855f7,color:#333
```

The flow: commit, capture, push, sync, recall.

1. **Commit.** You commit code as usual. The post-commit hook runs `rekal checkpoint`, which snapshots your active AI session into `data.db`. Append-only — nothing is ever modified or deleted.

2. **Push.** `rekal push` encodes your data into a compact wire format (zstd compression, string interning) and writes it to your personal orphan branch `rekal/<your-email>`. Your git history stays clean — rekal data lives on a separate branch.

3. **Sync.** `rekal sync` pulls your teammates' orphan branches and imports their session data into your local index.

4. **Recall.** `rekal "query"` runs a three-signal hybrid search (BM25 full-text, LSA embeddings, Nomic deep semantic embeddings) and returns scored results. The agent picks what matters.

### Two databases

Rekal keeps two local DuckDB databases. The split is deliberate.

- **data.db** — The shared truth. Append-only. Contains sessions, turns, tool calls, checkpoints, files touched. This is what gets encoded and pushed through git. `rekal query` reads from here.

- **index.db** — Local intelligence. Full-text indexes, vector embeddings, file co-occurrence graphs. Never synced. Rebuilt anytime with `rekal index`. This is what powers `rekal "query"` search.

Thin on the wire, rich on the machine.

### Orphan branches

Rekal data lives on git orphan branches named `rekal/<email>`. These branches have no common ancestor with your code branches — they do not appear in your project history, do not affect merges, and do not clutter your working tree. Standard git push and fetch move the data.

## Using Rekal with your agent

Rekal is agent-first. The agent is the primary consumer.

When you run `rekal init`, it installs a Claude Code skill that teaches the agent how to use Rekal. The agent learns to search for prior context before modifying files, load sessions progressively to stay within token budgets, and use the structured output to make informed decisions.

The agent workflow:

```bash
# Agent touches src/billing/ — first, recall prior context
rekal --file src/billing/ "discount logic"

# Agent finds a relevant session, drills in
rekal query --session 01JNQX... --limit 5

# Agent loads more detail if needed
rekal query --session 01JNQX... --full
```

The agent controls how much context it loads. Lightweight search first, full sessions only when needed.

## Commands reference

| Command | Description |
|---------|-------------|
| `rekal init` | Initialize Rekal in the current git repository |
| `rekal clean` | Remove Rekal setup from this repository |
| `rekal version` | Print the CLI version |
| `rekal checkpoint` | Capture the current session after a commit |
| `rekal push [--force]` | Push Rekal data to the remote branch |
| `rekal sync [--self]` | Sync team context from remote rekal branches |
| `rekal index` | Rebuild the index DB from the data DB |
| `rekal log [--limit N]` | Show recent checkpoints |
| `rekal [filters...] [query]` | Hybrid search over sessions |
| `rekal query --session <id> [--full]` | Drill into a session |
| `rekal query "<sql>" [--index]` | Run raw SQL against the data or index DB |

Full details: [docs/spec/command/](docs/spec/command/).

## Benchmarks

Measured on two real repositories. All times in seconds, wall clock, macOS/arm64.

### Dataset size

| Metric | 165 sessions | 57 sessions |
|--------|-------------|------------|
| Turns | 14,019 | 3,929 |
| data.db | 13 MB | 7.3 MB |
| index.db | 18 MB | 10 MB |

### Operation timing

| Operation | 165 sessions | 57 sessions |
|-----------|-------------|------------|
| init (cold) | 4.60s | 0.98s |
| checkpoint (cold) | 0.50s | 2.66s |
| checkpoint (incremental) | 0.51s | 0.23s |
| index | 0.85s | 0.61s |
| push | 0.18s | 1.93s |
| sync | 2.06s | 1.78s |
| search "authentication" | 0.15s | 0.13s |
| search "database migration" | 0.17s | 0.14s |
| search "error handling" | 0.16s | 0.13s |
| query | 0.14s | 0.10s |
| log | 0.14s | 0.10s |
| clean | 0.13s | 0.10s |

Search stays under 200ms at 14k turns.

## Development

```bash
git clone https://github.com/rekal-dev/rekal-cli.git rekal-cli
cd rekal-cli
mise install
```

See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for the full development guide.

## Getting help

```bash
rekal --help
rekal <command> --help
```

Issues: [github.com/rekal-dev/rekal-cli/issues](https://github.com/rekal-dev/rekal-cli/issues)

## License

Apache-2.0 — see [LICENSE](LICENSE).
