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

Search understands what you mean, not just what you type — powered by an embedded deep semantic model (nomic-embed-text) that runs entirely on your machine. No external APIs, no setup.

## Binary

If `rekal` is not on PATH, run `export PATH="$HOME/.local/bin:$PATH"` first.
The presence of this skill file means the binary is installed.

## When to Use

- Before modifying a file — check what prior sessions touched it
- When you need context about why code looks the way it does
- When the user asks about prior session history
- When working on files that were recently changed by AI agents

## Workflow

### 0. Wiki shortcut — one existence check

If `docs/wiki/index.md` exists, a materialized topic map of this memory is
already committed (see **rekal-wiki**): for broad "what do we know about X"
questions, read the topic page first — it may answer in one file read. Check
its watermark; anything newer, and any pointed or verification question,
goes through recall below. The wiki is a cache of memory, not the memory.

### 1. Search — find relevant sessions

```bash
rekal "JWT expiry"                      # keyword search (BM25 + LSA + Nomic hybrid)
rekal --file src/auth/ "token refresh"  # filter by file path (regex)
rekal --actor agent "migration"         # filter by actor type
rekal --author alice@co.com "billing"   # filter by author
rekal -n 5 "error handling"            # limit results
rekal --explain "error handling"       # + per-layer scores and related sessions (zoom edges)
```

Output is scored JSON. Each result includes:
- `session_id` — use with `rekal query --session <id>` to drill down
- `snippet` — the matching text from the best-matching turn
- `snippet_turn_index` — the turn index of the snippet (use as `--offset` for drill-down)
- `snippet_role` — the snippet's turn role: `human`, `human_steering` (a human correcting the agent mid-task — high intent), `assistant`, or `summary` (a harness-written distillation of the whole session — dense but machine text; drill into the raw turns beneath it for specifics)
- `summary_turn_index` — present when the session has a compaction summary: the turn index of the latest one. The cheapest overview of a long session is one drill away: `rekal query --session <id> --role summary`
- `score`, `actor`, `author`, `branch`, `files` — metadata for filtering
- `children` — subagent/workflow transcripts grouped under this result's trunk conversation (when any matched)
- `origin` — present only on cross-repo hits (see below): `repo:/path` or `shell:/path`, the working directory the session came from. The hit is from *another project on this machine* — weigh its relevance to this repo accordingly. Absent means the session belongs to this repo or a teammate.

### 2. Drill down — progressive context loading

Always start small to minimize token cost, then load more only when needed.

```bash
# Step 1: Use snippet_turn_index from search results to fetch a small window
# around the most relevant turn (e.g. if snippet_turn_index was 12)
rekal query --session 01JNQX... --offset 10 --limit 5

# Step 2: If you need broader context, fetch human turns to understand intent
rekal query --session 01JNQX... --role human

# Cheap overview of a long session: its compaction summaries (if any) — a
# harness-written distillation of files touched, decisions, errors and fixes
rekal query --session 01JNQX... --role summary

# Step 3: Only fetch full output when you actually need tool calls and files
rekal query --session 01JNQX... --full
```

Output includes `total_turns`, `offset`, `limit`, and `has_more` for navigation.

Do NOT load all turns or use `--full` by default. Use `snippet_turn_index` from
search results to jump directly to the relevant part of the conversation.

### 3. Widen memory — cross-repo recall

By default recall covers this repo (plus synced teammate sessions). If the
answer likely lives in the developer's *other* work on this machine, the
developer can widen recall to every local Claude Code session:

```bash
rekal index --include-all           # all repos + shell sessions on this machine
rekal index --include /path/to/repo # just that repo's sessions
rekal index --no-local              # back to this repo only
```

The setting persists across `index`/`sync` rebuilds. Imported sessions are
index-only — recallable here, structurally impossible to push to the team —
and carry the `origin` label in results. Suggest `--include-all` to the user
when a search comes up empty but the problem smells like something they've
solved elsewhere; don't run it unprompted, it's their history to widen.

### 4. Raw SQL — for edge cases

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
| `--explain` | Adds `layers` (normalized bm25/lsa/nomic scores) and `related` (sessions sharing touched files — zoom edges) to each result |

## Companion skills

This is the base skill — search and drill. Four focused skills build on it;
reach for them when the task matches:

- **rekal-provenance** — "how/why was this change made?" Anchor on a file or
  commit, walk back to the session that produced it, emit the why-chain. Use
  when reading unfamiliar code, onboarding, or reviewing a large codebase.
- **rekal-reflect** — before or after a task, mine your own prior sessions
  (especially `human_steering` turns) for recurring corrections and distill
  them into explicit rules. Use to stop repeating mistakes across sessions.
- **rekal-distill** — navigate memory as four libraries (context / decision /
  rules / boundary) and "zoom" around a topic or session. Use when you need a
  structured survey of what is known, undecided, preferred, and abandoned.
- **rekal-wiki** — materialize memory into committed `docs/wiki/<topic>.md`
  pages: topics from file co-occurrence, summaries from the sessions behind
  them (never from noisy commit messages), shipped as a PR so review admits
  the knowledge. Use to bootstrap or refresh a browsable knowledge base.
- **rekal-census** — read and summarise the *whole* corpus (or a bounded slice)
  exhaustively, not just the top matches. Use for "summarise everything",
  onboarding digests, or retrospectives — it scans on raw SQL for coverage,
  not relevance.

## Self-Service

Run `rekal <command> --help` for detailed help on any command, including
the full DB schemas (`rekal query --help`).

## Guidelines

- Search before modifying files that have prior session history
- Start with `rekal "keyword"` — only drop to raw SQL when the search workflow doesn't cover your need
- Use `snippet_turn_index` to jump to the relevant part of a session — don't load everything
- Human turns contain the intent; assistant turns contain the reasoning
- Cross-repo hits (`origin` set) are how a similar problem was solved *elsewhere* — treat them as prior art, not as this repo's conventions
- `actor_type` distinguishes human-initiated sessions from automated agent sessions
- Join `turns` with `tool_calls` via `session_id` to get context around file changes

## Data Model Notes

- `files_touched` (shown in `--full` output) comes from git diff AND session tool_calls — it includes files that were committed as well as files Written/Edited during the session. Change type `T` (touched) marks entries derived from tool_calls rather than git-native types (M/A/D/R).
- `tool_calls` in `--full` output includes a `path` field (absolute) for file-targeting tools — this is the most complete source for "what files did this session interact with."
- If `files_touched` seems incomplete for a session, query tool_calls directly:
  ```bash
  rekal query "SELECT DISTINCT path FROM tool_calls WHERE session_id = '<id>' AND path IS NOT NULL AND length(path) > 0"
  ```
