---
name: rekal
description: |
  Use this skill when working in a repo with Rekal initialized (.rekal/ exists).
  Rekal gives you memory of prior AI sessions — who changed what, why, and when.
  Start with `rekal "keyword"` to search, then drill into sessions with
  `rekal query --session <id>`. Run `rekal <command> --help` for full details.
---

# Rekal — Session Memory

Rekal captures AI coding sessions (conversation turns, tool calls, file changes) and stores them in a local DuckDB database. Use it to understand prior context before modifying code.

## Binary

If `rekal` is not on PATH, run `export PATH="$HOME/.local/bin:$PATH"` first.
The presence of this skill file means the binary is installed.

## When to Use

- Before modifying a file — check what prior sessions touched it
- When you need context about why code looks the way it does
- When the user asks about prior session history
- When working on files that were recently changed by AI agents

## Workflow

### 1. Search — find relevant sessions

```bash
rekal "JWT expiry"                      # keyword search (BM25 + LSA hybrid)
rekal --file src/auth/ "token refresh"  # filter by file path (regex)
rekal --actor agent "migration"         # filter by actor type
rekal --author alice@co.com "billing"   # filter by author
rekal -n 5 "error handling"            # limit results
```

Output is scored JSON with session IDs, snippets, and metadata.

### 2. Drill down — read the full conversation

```bash
rekal query --session 01JNQX...        # turns only
rekal query --session 01JNQX... --full # turns + tool calls + files touched
```

### 3. Raw SQL — for edge cases

```bash
rekal query "SELECT id, user_email, branch FROM sessions ORDER BY captured_at DESC LIMIT 5"
rekal query --index "SELECT * FROM file_cooccurrence WHERE file_a LIKE '%auth%' ORDER BY count DESC"
```

Run `rekal query --help` for the full data DB and index DB schemas.

## Filters (root command)

| Flag | Description |
|------|-------------|
| `--file <regex>` | Filter by file path (regex, git-root-relative) |
| `--commit <sha>` | Filter by git commit SHA |
| `--author <email>` | Filter by author email |
| `--actor <human\|agent>` | Filter by actor type |
| `-n`, `--limit <n>` | Max results (default: 20, 0 = no limit) |

## Self-Service

Run `rekal <command> --help` for detailed help on any command, including
the full DB schemas (`rekal query --help`).

## Guidelines

- Search before modifying files that have prior session history
- Start with `rekal "keyword"` — only drop to raw SQL when the search workflow doesn't cover your need
- Human turns contain the intent; assistant turns contain the reasoning
- `actor_type` distinguishes human-initiated sessions from automated agent sessions
- Join `turns` with `tool_calls` via `session_id` to get context around file changes

## Data Model Notes

- `files_touched` (shown in `--full` output) comes from git diff AND session tool_calls — it includes files that were committed as well as files Written/Edited during the session. Change type `T` (touched) marks entries derived from tool_calls rather than git-native types (M/A/D/R).
- `tool_calls` in `--full` output includes a `path` field (absolute) for file-targeting tools — this is the most complete source for "what files did this session interact with."
- If `files_touched` seems incomplete for a session, query tool_calls directly:
  ```bash
  rekal query "SELECT DISTINCT path FROM tool_calls WHERE session_id = '<id>' AND path IS NOT NULL AND length(path) > 0"
  ```
