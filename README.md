# Rekal

[Website](https://rekal.dev) · [GitHub](https://github.com/rekal-dev/rekal-cli) · [Discord](https://discord.gg/eNNabp4b)

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

The flow: commit → capture → push → sync → recall.

### Developer touchpoints

| You do | Rekal does |
|--------|------------|
| `rekal init` (once per repo) | Creates `.rekal/`, installs git hooks, writes agent skill file |
| `git commit` | Hook runs `rekal checkpoint` — snapshots your active AI session into `data.db` (append-only) |
| `git push` | Hook runs `rekal push` — encodes only your unexported data into compact wire format (zstd + string interning) and pushes to your orphan branch `rekal/<email>` |
| `rekal sync` (manual, when you want team context) | Fetches teammates' orphan branches, imports their sessions into your local DB and rebuilds the search index |
| `rekal clean` (if needed) | Removes `.rekal/` and hooks from the repo |

Day-to-day: commit and push as normal. Everything else is automatic.

### Agent touchpoints

| Agent does | Rekal does |
|------------|------------|
| `rekal "auth middleware"` | Runs hybrid search (BM25 + LSA + Nomic), returns scored JSON with `snippet_turn_index` pointing to the best-matching turn |
| `rekal query --session <id> --offset N --limit 5` | Returns a small window of turns around the relevant part of the conversation, with `has_more` for pagination |
| `rekal query --session <id> --role human` | Returns only human turns — cheapest way to understand session intent |
| `rekal query --session <id> --full` | Returns everything: turns, tool calls, files touched — only when the agent needs full detail |
| `rekal --file src/billing/ "discount"` | Scoped search filtered by file path |
| `rekal sync` (optional, at session start) | Pulls team context before the agent starts working |

The agent controls how much context it loads. Search first, drill down progressively, full sessions only when needed.

```bash
# Agent touches src/billing/ — first, recall prior context
rekal --file src/billing/ "discount logic"

# Agent finds a relevant session, drills into the matching turn
rekal query --session 01JNQX... --offset 10 --limit 5

# Agent loads full detail only if needed
rekal query --session 01JNQX... --full
```

### Ad-hoc usage

```bash
# Raw SQL for edge cases
rekal query "SELECT id, user_email, branch FROM sessions ORDER BY captured_at DESC LIMIT 5"

# Rebuild the search index after manual DB changes
rekal index

# View recent checkpoints
rekal log
```

### Two databases

Rekal keeps two local DuckDB databases. The split is deliberate.

- **data.db** — The shared truth. Append-only. Contains sessions, turns, tool calls, checkpoints, files touched — every branch, merged or not. This is the only source `rekal push` encodes from (filtered to merged work — see below). `rekal query` reads from here.

- **index.db** — Local intelligence. Full-text indexes, vector embeddings, file co-occurrence graphs. Never synced. Rebuilt anytime with `rekal index`. This is what powers `rekal "query"` search.

Thin on the wire, rich on the machine.

### Worktrees

Linked git worktrees (`git worktree add`) share **one** `.rekal/` store — the
one in the main checkout. Init once in the main repo; every worktree then reads
and writes the same data, index, and config, so there's no per-worktree
`rekal sync` or reindex. Checkpoints still record the branch and commit of
whichever worktree you committed in. A repo that never uses worktrees is
unaffected — the store is just its own `.rekal/`.

### Orphan branches

Rekal data lives on git orphan branches named `rekal/<email>`. These branches have no common ancestor with your code branches — they do not appear in your project history, do not affect merges, and do not clutter your working tree. Standard git push and fetch move the data.

### What gets shared: merged work only

Your local databases keep **every** branch — full fidelity, nothing gated. The wire is different: `rekal push` shares a session only when its code **landed on the default branch**, detected two ways, both exact:

- its commit is an ancestor of `main` (merge-commit and rebase workflows), or
- its branch's changes landed as a **squash merge** (patch-equivalence detection — no heuristics)

Unmerged work simply waits: it stays local, is re-checked on every push, and ships automatically the moment its branch merges. Abandoned branches never qualify, so a dead-end spike never reaches your teammates. Commit everything for yourself; share only what merged.

### Cross-repo recall (optional)

Your agent's memory can span your whole machine, not just this repo:

```bash
rekal index --include-all            # recall every local Claude Code session (all repos + shell)
rekal index --include /path/to/repo  # just that repo
rekal index --no-local               # back to this repo only
```

Imported sessions live in the **index only** — never in `data.db`, which is the only thing `push` reads — so they are structurally impossible to share. Results are labeled with their origin (`repo:/path`, `shell:/path`). The setting persists across rebuilds.

## Configuration (optional)

Rekal is zero-config by default. When you do want to tune it, there is exactly one file: `.rekal/config.json` — gitignored, local to the machine, never committed.

```json
{
  "local_import": { "all": true },
  "weights": {
    "bm25": 0.35,
    "lsa": 0.10,
    "nomic": 0.55,
    "steering_boost": 1.3,
    "subagent_downweight": 0.7
  },
  "embedding": {
    "endpoint": "$EMBED_ENDPOINT",
    "model": "nomic-embed-text-v1.5",
    "api_key_env": "EMBED_API_KEY",
    "timeout_seconds": 10
  }
}
```

- **`weights`** tunes recall ranking (layer mix, steering-turn boost, subagent discount). Applied at query time — changing them takes effect on the next search, no reindex, any corpus size.
- **`embedding`** switches deep semantic embeddings from the embedded nomic model to any OpenAI-compatible endpoint (vLLM, Ollama, LM Studio, TEI). Requests are batched and hard-timeboxed so a slow server can never stall a commit (embedding is always non-fatal). Pointed at localhost, your data still never leaves the machine; pointed at a cloud API, session text leaves — your call, made explicitly.

### API key: three ways, pick one

| Form | Example | Where the secret lives |
|---|---|---|
| Real string | `"api_key": "sk-abc123"` | In the file (gitignored, this machine only) |
| Env reference | `"api_key": "$MY_KEY"` | In the environment, expanded at run time |
| Env var name | `"api_key_env": "EMBED_API_KEY"` | In the environment, read directly |

Precedence: `api_key_env` wins when set and the variable is non-empty; otherwise `api_key` (after `$VAR` expansion) is used; no key at all just omits the `Authorization` header — the normal case for a localhost server. `endpoint` expands `$VAR` the same way. One edge: a *hardcoded* `api_key` containing a literal `$` would be treated as an env reference — real provider keys never contain `$`, and `api_key_env` is the unambiguous form for anything sensitive.
- Switching embedding model/endpoint requires one `rekal index` to regenerate vectors. A content-hash-keyed cache (`.rekal/embed-cache.db`, vectors only, never text) makes routine rebuilds embed only new sessions — and makes a model switch cost exactly one full pass.

## Commands reference

| Command | Description |
|---------|-------------|
| `rekal init` | Initialize Rekal in the current git repository |
| `rekal clean` | Remove Rekal setup from this repository |
| `rekal version` | Print the CLI version |
| `rekal checkpoint` | Capture the current session after a commit |
| `rekal push [--force] [--re-export]` | Push Rekal data to the remote branch (merged work only) |
| `rekal sync [--self]` | Sync team context from remote rekal branches |
| `rekal index [--include-all\|--include <repo>\|--no-local]` | Rebuild the index DB; optionally fold in cross-repo local sessions |
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
