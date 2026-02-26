# rekal query

**Role:** Two modes: raw SQL over the Rekal data model, or session drill-down. The `--session` flag is the second step in progressive context loading — after recall returns snippets, the agent drills into specific sessions for full turns.

**Invocation:** `rekal query "<sql>"`, `rekal query --index "<sql>"`, or `rekal query --session <id> [--full]`.

---

## Preconditions

See [preconditions.md](../preconditions.md): git repo, init done.

---

## Two modes

### SQL mode (default)

Run a single SELECT statement against the data DB or index DB.

1. **Choose target** — Data DB (`.rekal/data.db`) by default; index DB (`.rekal/index.db`) if `--index`.
2. **Execute** — Read-only (SELECT only). Rejects non-SELECT statements.
3. **Output** — One JSON object per row (NDJSON).

### Session drill-down (`--session <id>`)

Returns the full conversation for a specific session. This is the progressive loading drill-down — after `rekal <query>` returns scored snippets, the agent calls `rekal query --session <id>` to get full turns.

1. **Query session** — Fetch session metadata from `sessions` table.
2. **Query turns** — Fetch all turns ordered by `turn_index`.
3. **If `--full`** — Also fetch tool calls and files touched.
4. **Output** — Single JSON object with session metadata, turns, and optionally tool calls and files.

`--session` and positional SQL are mutually exclusive.

---

## Flags

| Flag | Meaning |
|------|--------|
| `--index` | Run SQL against the **index DB** instead of the data DB |
| `--session <id>` | Show session conversation by ID (drill-down mode) |
| `--full` | Include tool calls and files in session output (requires `--session`) |

---

## Schema

**Data DB** (default):

| Table | Purpose |
|-------|--------|
| `sessions` | One row per captured session (id, session_hash, captured_at, actor_type, agent_id, user_email, branch) |
| `turns` | Conversation turns (id, session_id, turn_index, role, content, ts) |
| `tool_calls` | Tool invocations (id, session_id, call_order, tool, path, cmd_prefix) |
| `checkpoints` | Git commit anchors (id, git_sha, git_branch, user_email, ts, actor_type, agent_id, exported) |
| `files_touched` | Files changed per checkpoint (id, checkpoint_id, file_path, change_type) |
| `checkpoint_sessions` | Junction: checkpoint_id → session_id |
| `checkpoint_state` | Incremental state cache (file_path, byte_size, file_hash) |

**Index DB** (`--index`):

| Table | Purpose |
|-------|--------|
| `turns_ft` | Turn-level full-text search (id, session_id, turn_index, role, content, ts) |
| `tool_calls_index` | Tool calls per session (id, session_id, call_order, tool, path, cmd_prefix) |
| `files_index` | Files per checkpoint (checkpoint_id, session_id, file_path, change_type) |
| `session_facets` | Session metadata (session_id, user_email, git_branch, actor_type, agent_id, captured_at, turn_count, tool_call_count, file_count, checkpoint_id, git_sha) |
| `file_cooccurrence` | Files that change together (file_a, file_b, count) |
| `session_embeddings` | LSA vectors (session_id, embedding, model, generated_at) |
| `index_state` | Key-value state (key, value) |

---

## Examples

```bash
rekal query "SELECT id, git_sha, user_email FROM checkpoints ORDER BY ts DESC LIMIT 5"
rekal query "SELECT session_id, file_path FROM files_touched WHERE file_path LIKE '%auth%'"
rekal query --index "SELECT file_a, file_b, count FROM file_cooccurrence WHERE file_a = 'src/auth/middleware.go' ORDER BY count DESC LIMIT 10"
rekal query --index "SELECT session_id, user_email, turn_count FROM session_facets WHERE actor_type = 'human'"
```
