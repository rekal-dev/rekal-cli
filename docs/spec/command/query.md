# rekal query

**Role:** Two modes: raw SQL over the Rekal data model, or session drill-down. The `--session` flag is the second step in progressive context loading — after recall returns snippets, the agent drills into specific sessions for full turns.

**Short forms:** `-q` sql · `-i` index · `-s` session · `-F` full · `-o` offset · `-n` limit · `-r` role · `-j` json

**Invocation:** `rekal query --sql "<sql>"` (explicit SQL mode; a bare positional `rekal query "<sql>"` is accepted as shorthand), `rekal query --index --sql "<sql>"`, or `rekal query --session <id> [--full] [--offset N] [--limit N] [--role human|assistant|human_steering|summary]`. `--sql`, a positional statement, and `--session` are mutually exclusive.

---

## Preconditions

See [preconditions.md](../preconditions.md): git repo, init done.

---

## Two modes

### SQL mode (`--sql "<statement>"`, or a bare positional statement)

Run a single SELECT statement against the data DB or index DB. The mode is
explicit via `--sql`; a positional statement is the accepted shorthand.

1. **Choose target** — Data DB (`.rekal/data.db`) by default; index DB (`.rekal/index.db`) if `--index`.
2. **Execute** — Read-only (SELECT only). Rejects non-SELECT statements.
3. **Output** — TSV rows by default (header + values). With `--json`, one JSON object per row (NDJSON).

### Session drill-down (`--session <id>`)

Returns the full conversation for a specific session. This is the progressive loading drill-down — after `rekal <query>` returns a seed digest (or `--json` results), the agent calls `rekal query --session <id>` to get full turns.

1. **Query session** — Fetch session metadata from `sessions` table.
2. **Query turns** — Fetch turns ordered by `turn_index`, applying `--role` filter if set.
3. **Count total** — Run a COUNT query (respecting `--role` filter) to populate `total_turns`.
4. **Paginate** — Apply `--offset` and `--limit` to the turn query.
5. **If `--full`** — Also fetch tool calls and files touched.
6. **Output** — Readable turn transcript by default; with `--json`, a single JSON object with session metadata, pagination fields, turns, and optionally tool calls and files. Optional harness metadata (`agent_id`, `team_name`, `workflow_name`, `parent_session_id`) is included when present and omitted for sessions from agents without the concept — see [agent-metadata.md](../../agent-metadata.md). Also always includes `child_session_ids` — the sessions whose `parent_session_id` points at this one (subagent/workflow transcripts folded under it in recall) — empty when there are none, so an agent can navigate from a collapsed recall result into the exact transcript that matched.

`--session` and positional SQL are mutually exclusive. `--offset`, `--limit`, and `--role` require `--session`.

#### Pagination output fields

| Field | Type | Description |
|-------|------|-------------|
| `total_turns` | int | Total turns matching the role filter (always present) |
| `offset` | int | Number of turns skipped (omitted when 0) |
| `limit` | int | Max turns returned (omitted when 0 / no limit) |
| `has_more` | bool | True when more turns exist beyond this page (omitted when false or no limit) |

---

## Flags

| Flag | Meaning |
|------|--------|
| `--sql <statement>` | SQL SELECT to run (explicit SQL mode). A bare positional statement is accepted as shorthand; the two are mutually exclusive, and both are mutually exclusive with `--session` |
| `--index` | Run SQL against the **index DB** instead of the data DB |
| `--session <id>` | Show session conversation by ID (drill-down mode) |
| `--full` | Include tool calls and files in session output (requires `--session`) |
| `--offset <n>` | Skip first N turns (default: 0, requires `--session`) |
| `--limit <n>` | Max turns to return, 0 = no limit (default: 0, requires `--session`) |
| `--role <human\|assistant\|human_steering\|summary>` | Filter turns by role (requires `--session`). Matches exactly — queue-operation steering turns are stored as role `human_steering` (see [agent-metadata.md](../../agent-metadata.md)) and are not returned by `--role human`; `summary` turns are harness-written compaction distillations (rows stored as `human` by pre-summary versions are reclassified by content fingerprint at read time, scoped to `source = 'claude'` sessions so other agent types are untouched); omit `--role` to see all turns. |
| `--json` | JSON instead of the default text/TSV — session drill → one object; SQL → NDJSON. |

---

## Schema

**Data DB** (default):

| Table | Purpose |
|-------|--------|
| `sessions` | One row per captured session (id, parent_session_id, session_hash, captured_at, actor_type, agent_id, user_email, branch, source, team_name, workflow_name, agent_type, description, spawn_depth) |
| `turns` | Conversation turns (id, session_id, turn_index, role, content, ts) |
| `tool_calls` | Tool invocations (id, session_id, call_order, tool, path, cmd_prefix) |
| `checkpoints` | Git commit anchors (id, git_sha, git_branch, user_email, ts, actor_type, agent_id, exported) |
| `files_touched` | Files changed per checkpoint (id, checkpoint_id, file_path, change_type) |
| `checkpoint_sessions` | Junction: checkpoint_id → session_id |
| `checkpoint_state` | Incremental state cache (file_path, byte_size, file_hash) |
| `recall_edges` | L1 recall citation graph (id, ts, kind, query, target_session_id); one row per session reached (kind `recall`/`drill`). Local-only — never pushed/synced |

**Index DB** (`--index`):

| Table | Purpose |
|-------|--------|
| `turns_ft` | Turn-level full-text search (id, session_id, turn_index, role, content, ts) |
| `tool_calls_index` | Tool calls per session (id, session_id, call_order, tool, path, cmd_prefix) |
| `files_index` | Files per checkpoint (checkpoint_id, session_id, file_path, change_type) |
| `session_facets` | Session metadata (session_id, user_email, git_branch, actor_type, agent_id, captured_at, turn_count, tool_call_count, file_count, checkpoint_id, git_sha, parent_session_id, team_name, workflow_name, agent_type, description, spawn_depth, origin, facet_text) |
| `file_cooccurrence` | Files that change together (file_a, file_b, count) |
| `session_embeddings` | LSA + Nomic vectors (session_id, embedding, model, generated_at); models `lsa-v1`, `nomic-v1.5-c8k` |
| `knowledge_chunks` | Heading-anchored prose sections of tracked files at HEAD (id, path, anchor, breadcrumb, start_line, end_line, content, content_hash, blob_sha) |
| `knowledge_embeddings` | Chunk vectors (content_hash, model, embedding) |
| `session_reach` | L1 reach aggregate derived from data.db.recall_edges (target_session_id, reach_count, last_query, last_ts) |
| `index_state` | Key-value state (key, value) |

> **`ts` is a TIMESTAMP.** `turns.ts` / `turns_ft.ts` are typed timestamps, not
> text — `ts LIKE '2023-05%'` raises a DuckDB Binder error (a common cause of a
> temporal query wrongly reading as "no rows"). Use
> `ts BETWEEN TIMESTAMP '2023-05-01' AND TIMESTAMP '2023-06-01'` or
> `CAST(ts AS VARCHAR) LIKE '2023-05%'`. The FTS-internal tables (`dict`,
> `docs`, `fields`, `stats`, `stopwords`, `terms`) are DuckDB search internals —
> ignore them; query `turns_ft` instead.

---

## Examples

```bash
# Session drill-down with pagination
rekal query --session 01JNQX... --limit 5           # first 5 turns
rekal query --session 01JNQX... --offset 5 --limit 5 # next 5 turns
rekal query --session 01JNQX... --role human         # human turns only
rekal query --session 01JNQX... --role human --limit 3 # first 3 human turns

# Raw SQL
rekal query "SELECT id, git_sha, user_email FROM checkpoints ORDER BY ts DESC LIMIT 5"
rekal query "SELECT session_id, file_path FROM files_touched WHERE file_path LIKE '%auth%'"
rekal query --index "SELECT file_a, file_b, count FROM file_cooccurrence WHERE file_a = 'src/auth/middleware.go' ORDER BY count DESC LIMIT 10"
rekal query --index "SELECT session_id, user_email, turn_count FROM session_facets WHERE actor_type = 'human'"

# Temporal window — ts is TIMESTAMP, use BETWEEN not LIKE
rekal query "SELECT ts, session_id, content FROM turns WHERE ts BETWEEN TIMESTAMP '2023-05-01' AND TIMESTAMP '2023-06-01' AND role='human' ORDER BY ts"

# Knowledge layer — prose chunks of tracked files at HEAD
rekal query --index "SELECT path, anchor, start_line, end_line FROM knowledge_chunks WHERE content ILIKE '%merged-only%' ORDER BY path"
```
