# Rekal Data DB Schema

Data DB (`.rekal/data.db`) is the source of truth. Append-only, never rebuilt. Committed to the rekal orphan branch for sharing via push/sync.

Engine: DuckDB.

---

## `sessions`

One row per captured Claude Code session. Inserted by `rekal checkpoint`. Deduplicated by `session_hash` — if the hash matches an existing row, checkpoint skips it.

```sql
CREATE TABLE IF NOT EXISTS sessions (
    id                VARCHAR PRIMARY KEY,
    parent_session_id VARCHAR,
    session_hash      VARCHAR NOT NULL,
    captured_at       TIMESTAMP NOT NULL,
    actor_type        VARCHAR NOT NULL DEFAULT 'human',
    agent_id          VARCHAR,
    user_email        VARCHAR,
    branch            VARCHAR
);
```

| Column | Description |
|--------|-------------|
| `id` | ULID generated at capture time |
| `parent_session_id` | FK → `sessions.id`. Null for top-level (human-initiated) sessions. Set for Task subagent sessions — points to the parent that spawned them. Forms a tree: human → subagent → nested subagent |
| `session_hash` | SHA-256 hex of the raw `.jsonl` file content. Dedup key |
| `captured_at` | When the session was captured (UTC) |
| `actor_type` | Who initiated the session: `"human"` (interactive user) or `"agent"` (automated process). See [role vs actor_type](#role-vs-actor_type) |
| `agent_id` | Identifier for the agent if `actor_type` is `"agent"`. Null for human |
| `user_email` | Git `user.email` at capture time |
| `branch` | Git branch from session metadata |

---

## `turns`

Conversation turns extracted from session JSONL. One row per human prompt or assistant text response.

```sql
CREATE TABLE IF NOT EXISTS turns (
    id              VARCHAR PRIMARY KEY,
    session_id      VARCHAR NOT NULL REFERENCES sessions(id),
    turn_index      INTEGER NOT NULL,
    role            VARCHAR NOT NULL,
    content         VARCHAR NOT NULL,
    ts              TIMESTAMP
);
```

| Column | Description |
|--------|-------------|
| `id` | ULID |
| `session_id` | FK → `sessions.id` |
| `turn_index` | 0-based position within the session |
| `role` | Who said this: `"human"` (user prompt), `"human_steering"` (out-of-band queue-operation message typed while an agent was already working — the highest-intent signal in the corpus; boosted in recall ranking, see [agent-metadata.md](../agent-metadata.md)), or `"assistant"` (Claude response). See [role vs actor_type](#role-vs-actor_type) |
| `content` | Text content of the turn. Tool results and thinking blocks are excluded |
| `ts` | Timestamp from the JSONL line (UTC) |

**Included:** Human prompts (text only), assistant text responses.

**Excluded:** Tool result content (file bodies, command outputs), thinking blocks, system prompts, `isSidechain` messages, file history snapshots.

---

## `tool_calls`

Tool invocations extracted from assistant messages. One row per `tool_use` block.

```sql
CREATE TABLE IF NOT EXISTS tool_calls (
    id              VARCHAR PRIMARY KEY,
    session_id      VARCHAR NOT NULL REFERENCES sessions(id),
    call_order      INTEGER NOT NULL,
    tool            VARCHAR NOT NULL,
    path            VARCHAR,
    cmd_prefix      VARCHAR
);
```

| Column | Description |
|--------|-------------|
| `id` | ULID |
| `session_id` | FK → `sessions.id` |
| `call_order` | 0-based position within the session |
| `tool` | Tool name: `Write`, `Edit`, `Read`, `Bash`, `Glob`, `Grep`, `Task`, etc. |
| `path` | File path argument (from `file_path` or `path` input field). Null for tools without a path |
| `cmd_prefix` | First 100 characters of `command` input (Bash tool only). Null otherwise |

**Included:** Tool name, file path, command prefix.

**Excluded:** Full tool input (file content being written), tool output/results.

---

## `checkpoints`

One row per checkpoint commit on the orphan branch. The `id` is the commit SHA on `rekal/<email>` — this is the checkpoint ID.

```sql
CREATE TABLE IF NOT EXISTS checkpoints (
    id              VARCHAR PRIMARY KEY,
    git_sha         VARCHAR NOT NULL,
    git_branch      VARCHAR NOT NULL,
    user_email      VARCHAR NOT NULL,
    ts              TIMESTAMP NOT NULL,
    actor_type      VARCHAR NOT NULL DEFAULT 'human',
    agent_id        VARCHAR
);
```

| Column | Description |
|--------|-------------|
| `id` | Commit SHA on the `rekal/<email>` orphan branch. The checkpoint ID |
| `git_sha` | HEAD commit SHA of the **main repo** at checkpoint time |
| `git_branch` | Active branch of the main repo at checkpoint time |
| `user_email` | Git `user.email` |
| `ts` | Checkpoint timestamp (UTC) |
| `actor_type` | `"human"` or `"agent"` |
| `agent_id` | Agent identifier if applicable |

---

## `files_touched`

Files changed in the main repo commit associated with a checkpoint. Derived from `git diff --name-status HEAD~1 HEAD`.

```sql
CREATE TABLE IF NOT EXISTS files_touched (
    id              VARCHAR PRIMARY KEY,
    checkpoint_id   VARCHAR NOT NULL REFERENCES checkpoints(id),
    file_path       VARCHAR NOT NULL,
    change_type     VARCHAR NOT NULL
);
```

| Column | Description |
|--------|-------------|
| `id` | ULID |
| `checkpoint_id` | FK → `checkpoints.id` |
| `file_path` | Relative path from git root |
| `change_type` | Git status letter: `A` (added), `M` (modified), `D` (deleted), `R` (renamed) |

---

## `checkpoint_sessions`

Junction table linking checkpoints to the sessions that were active at that point.

```sql
CREATE TABLE IF NOT EXISTS checkpoint_sessions (
    checkpoint_id   VARCHAR NOT NULL REFERENCES checkpoints(id),
    session_id      VARCHAR NOT NULL REFERENCES sessions(id),
    PRIMARY KEY (checkpoint_id, session_id)
);
```

---

## `role` vs `actor_type`

These are orthogonal concepts:

**`role`** (on `turns`) — who is speaking in this conversation turn:
- `"human"` — the user's prompt
- `"assistant"` — Claude's response

Every session has turns with both roles regardless of who started it.

**`actor_type`** (on `sessions`, `checkpoints`) — who initiated and owns the session:
- `"human"` — a person using Claude Code interactively
- `"agent"` — an automated process (CI, Task subagent, scheduled job)

An agent-driven session still has `role: "human"` turns — they're generated by the agent, not typed by a person. A human-driven session still has `role: "assistant"` turns from Claude.

---

## Session hierarchy

Sessions form a tree via `parent_session_id`:

```
human session (parent_session_id = null, actor_type = "human")
  └─ Task subagent (parent_session_id = human session, actor_type = "agent")
       └─ nested subagent (parent_session_id = parent subagent, actor_type = "agent")
```

Cross-user relationships are handled by `user_email` + `rekal sync`. Each user's sessions are independent; team context is merged at sync time.

---

## Implementation status

| Table | Populated by | Status |
|-------|-------------|--------|
| `sessions` | `rekal checkpoint` | Done |
| `turns` | `rekal checkpoint` | Done |
| `tool_calls` | `rekal checkpoint` | Done |
| `checkpoints` | `rekal checkpoint` | TODO — insert after orphan branch commit |
| `files_touched` | `rekal checkpoint` | TODO — from `git diff --name-status` |
| `checkpoint_sessions` | `rekal checkpoint` | TODO — link checkpoint to sessions |

---

# Rekal Index DB Schema

Index DB (`.rekal/index.db`) is derived from the data DB. Local-only, never synced. Rebuilt from scratch by `rekal index` or `rekal sync`. Incrementally updated by `rekal checkpoint`.

Engine: DuckDB.

---

## `turns_ft`

Full-text search index over conversation turns. Copy of `turns` from data DB, indexed by DuckDB's FTS extension for BM25 scoring.

```sql
CREATE TABLE IF NOT EXISTS turns_ft (
    id              VARCHAR PRIMARY KEY,
    session_id      VARCHAR NOT NULL,
    turn_index      INTEGER NOT NULL,
    role            VARCHAR NOT NULL,
    content         VARCHAR NOT NULL,
    ts              VARCHAR
);
```

---

## `tool_calls_index`

Indexed copy of tool calls for fast lookup by tool, path, or session.

```sql
CREATE TABLE IF NOT EXISTS tool_calls_index (
    id              VARCHAR PRIMARY KEY,
    session_id      VARCHAR NOT NULL,
    call_order      INTEGER NOT NULL,
    tool            VARCHAR NOT NULL,
    path            VARCHAR,
    cmd_prefix      VARCHAR
);
```

---

## `files_index`

Denormalized file changes with session linkage for file-based filtering.

```sql
CREATE TABLE IF NOT EXISTS files_index (
    checkpoint_id   VARCHAR NOT NULL,
    session_id      VARCHAR NOT NULL,
    file_path       VARCHAR NOT NULL,
    change_type     VARCHAR NOT NULL
);
```

---

## `session_facets`

Aggregated session metadata for fast filtering and display.

```sql
CREATE TABLE IF NOT EXISTS session_facets (
    session_id      VARCHAR NOT NULL,
    user_email      VARCHAR,
    git_branch      VARCHAR,
    actor_type      VARCHAR,
    agent_id        VARCHAR,
    captured_at     TIMESTAMP,
    turn_count      INTEGER,
    tool_call_count INTEGER,
    file_count      INTEGER,
    checkpoint_id   VARCHAR,
    git_sha         VARCHAR
);
```

---

## `session_embeddings`

Vector embeddings for semantic search. Stores both LSA and nomic-embed-text vectors, keyed by `(session_id, model)`.

```sql
CREATE TABLE IF NOT EXISTS session_embeddings (
    session_id      VARCHAR NOT NULL,
    embedding       FLOAT[],
    model           VARCHAR NOT NULL,
    generated_at    TIMESTAMP NOT NULL,
    PRIMARY KEY (session_id, model)
);
```

| Column | Description |
|--------|-------------|
| `session_id` | FK → session being embedded |
| `embedding` | Vector as FLOAT array. Dimension depends on model |
| `model` | Model identifier: `"lsa-v1"` (variable dim) or `"nomic-v1.5"` (768 dim) |
| `generated_at` | When the embedding was computed |

**Scoring weights:**
- When nomic is available (3-way): BM25 0.3, LSA 0.2, Nomic 0.5
- When nomic is unavailable (2-way fallback): BM25 0.4, LSA 0.6

---

## `file_cooccurrence`

File co-occurrence graph derived from tool calls. Two files that appear in the same session are co-occurring.

```sql
CREATE TABLE IF NOT EXISTS file_cooccurrence (
    file_a          VARCHAR NOT NULL,
    file_b          VARCHAR NOT NULL,
    session_count   INTEGER NOT NULL,
    PRIMARY KEY (file_a, file_b)
);
```

---

## `index_state`

Metadata about the last index build.

```sql
CREATE TABLE IF NOT EXISTS index_state (
    session_count   INTEGER,
    turn_count      INTEGER,
    embedding_dim   INTEGER,
    last_indexed_at TIMESTAMP
);
```
