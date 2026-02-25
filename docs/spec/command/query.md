# rekal query

**Role:** Raw SQL over the Rekal data model. Run a single SQL statement against the data DB (default) or the index DB (`--index`). For power users and agents that need schema-level access. Tight to the table schema — no abstraction above SQL.

**Invocation:** `rekal query "<sql>"` or `rekal query --index "<sql>"`.

---

## Preconditions

See [preconditions.md](../preconditions.md): git repo, init done. If using `--index`, the index DB must exist (same as recall; preconditions ensure it or run index if missing).

---

## What query does

1. **Run shared preconditions** — Git root, init done. If `--index`, ensure index exists (run index if missing/empty).
2. **Choose target** — Data DB (`.rekal/data.db`) by default; index DB (`.rekal/index.db`) if `--index`.
3. **Execute** — Run the given SQL as a single statement against the chosen DB. Read-only (SELECT only; no INSERT/UPDATE/DELETE/DDL).
4. **Output** — Result rows in a machine-friendly form (e.g. JSON or tab-separated). Same soul as recall: for tool result.

---

## Flag

| Flag | Meaning |
|------|--------|
| `--index` | Run SQL against the **index DB** instead of the data DB. |

Default (no flag) = data DB.

---

## Tied to the data model

Queries run against the actual table schema. No views or stored procedures; the schema is the contract.

**Data DB** (default): `.rekal/data.db`

| Table | Purpose |
|-------|--------|
| `sessions` | One row per captured session (ULID, session_hash, payload JSON, captured_at, …). Append-only. |
| `checkpoints` | One row per git commit (id, git_sha, git_branch, user_email, ts, actor_type, agent_id). |
| `files_touched` | Files changed per checkpoint (checkpoint_id, file_path, change_type). |
| `checkpoint_sessions` | Junction: checkpoint_id → session_id (which sessions at which commit). |

**Index DB** (`--index`): `.rekal/index.db`

| Table | Purpose |
|-------|--------|
| `turns_ft` | Turn-level fulltext (session_id, turn_index, role, content, …). FTS on content. |
| `tool_calls_index` | Tool calls per session (tool, path, cmd_prefix, call_order). |
| `files_index` | File paths per checkpoint (checkpoint_id, file_path). |
| `session_facets` | Session metadata (user_email, git_branch, turn_count, actor_type, agent_id, …). |
| `file_cooccurrence` | (file_a, file_b, count) — files that change together. |

Authoritative schema: system design doc (data DB §8.1, index DB §8.8). This spec summarizes for the command; implementation uses the same DDL.

---

## Read-only

Only SELECT is allowed. No INSERT, UPDATE, DELETE, or DDL. Prevents accidental mutation of shared or local state.

---

## Examples

```bash
rekal query "SELECT id, git_sha, user_email FROM checkpoints ORDER BY ts DESC LIMIT 5"
rekal query "SELECT session_id, file_path FROM files_touched WHERE file_path LIKE '%auth%'"
rekal query --index "SELECT file_a, file_b, count FROM file_cooccurrence WHERE file_a = 'src/auth/middleware.go' ORDER BY count DESC LIMIT 10"
rekal query --index "SELECT session_id, user_email, turn_count FROM session_facets WHERE actor_type = 'human'"
```
