# rekal query

**Role:** Raw SQL over the Rekal data model. Run a single SQL statement against the data DB (default) or the index DB (`--index`). For power users and agents that need schema-level access.

**Invocation:** `rekal query "<sql>"` or `rekal query --index "<sql>"`.

---

## Preconditions

See [preconditions.md](../preconditions.md): git repo, init done.

---

## What query does

1. **Run shared preconditions** — Git root, init done.
2. **Choose target** — Data DB (`.rekal/data.db`) by default; index DB (`.rekal/index.db`) if `--index`.
3. **Execute** — Run the given SQL as a single statement. Read-only (SELECT only).
4. **Output** — Tab-separated results with column headers.

---

## Flag

| Flag | Meaning |
|------|--------|
| `--index` | Run SQL against the **index DB** instead of the data DB |

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
