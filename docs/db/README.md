# Rekal Data DB Schema

> For the map — what `.tables` shows in each database, how the two relate, and
> which tables reach the wire — start with [overview.md](overview.md). This
> page is the column-by-column detail.


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
| `role` | Who said this: `"human"` (user prompt), `"human_steering"` (out-of-band queue-operation message typed while an agent was already working — the highest-intent signal in the corpus; boosted in recall ranking, see [agent-metadata.md](../agent-metadata.md)), `"summary"` (harness-written compaction distillation — densest recall anchor, boosted below steering; rows written before this role existed are stored as `human` and reclassified by content fingerprint in the derived views, scoped to `source = 'claude'` sessions), or `"assistant"` (Claude response). See [role vs actor_type](#role-vs-actor_type) |
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

## Local-only tables

These four live in `data.db` but are **never encoded onto the wire**. Nothing
in `transport/` reads them, and that is deliberate — see
[SECURITY.md](../../SECURITY.md).

### `checkpoint_state`

Maps a transcript file to the session it was captured as, so a re-capture
**appends** to that conversation instead of storing it again. A live
conversation has different content at every commit, so keying dedup on content
made each commit a brand-new session.

```sql
CREATE TABLE IF NOT EXISTS checkpoint_state (
    file_path   VARCHAR PRIMARY KEY,
    byte_size   BIGINT,
    file_hash   VARCHAR,
    session_id  VARCHAR          -- the conversation this transcript became
);
```

`byte_size` + `file_hash` are the cheap skip: unchanged transcript, no work.

### `recall_edges`

The L1 recall citation graph — one row per session surfaced by a recall or
opened by a drill. Aggregated into `index.db.session_reach`, which produces the
`[reached N×]` hint.

```sql
CREATE TABLE IF NOT EXISTS recall_edges (
    id                 VARCHAR PRIMARY KEY,
    ts                 TIMESTAMP,
    kind               VARCHAR,   -- 'recall' | 'drill'
    query              VARCHAR,
    target_session_id  VARCHAR
);
```

This holds **query text**. Exporting it would publish what each developer
searched for, which is why it never leaves the machine.

### `merge_gate_cache`

Memoized merged-only verdicts, created on demand at first push.

```sql
CREATE TABLE IF NOT EXISTS merge_gate_cache (
    git_sha      VARCHAR NOT NULL,
    target_tip   VARCHAR NOT NULL,   -- mainline tip the verdict was made against
    gate_version INTEGER NOT NULL,   -- db.MergeGateVersion
    shareable    BOOLEAN NOT NULL,
    PRIMARY KEY (git_sha, target_tip, gate_version)
);
```

The tip is in the key because the answer changes when the mainline moves: work
unmerged at one tip may have landed by the next. The version is in the key
because the answer also changes when the rule does. Strictly an accelerator —
it can only skip a repeat, never let through anything the gate would refuse.

### `schema_meta`

Stamped schema version, so an older rekal opening a newer store detects the
mismatch rather than misreading columns it predates.

```sql
CREATE TABLE IF NOT EXISTS schema_meta (key VARCHAR PRIMARY KEY, value VARCHAR);
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

## Who writes what

| Table | Populated by |
|-------|-------------|
| `sessions` | `rekal checkpoint` |
| `turns` | `rekal checkpoint` |
| `tool_calls` | `rekal checkpoint` |
| `checkpoints` | `rekal checkpoint` |
| `files_touched` | `rekal checkpoint` (git diff + Write/Edit tool paths) |
| `checkpoint_sessions` | `rekal checkpoint` |
| `checkpoint_state` | `rekal checkpoint` |
| `recall_edges` | recall/drill spool, drained at checkpoint |
| `merge_gate_cache` | `rekal push` |
| `schema_meta` | schema migration |

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

Aggregated session metadata for fast filtering and display, plus the
session's facet document (`facet_text`: distinct tool paths + command
prefixes + steering text, capped — deterministic, no LLM). The facet
ranking layer BM25-searches `facet_text` via a guarded FTS index
(built only when at least one session has facet material; scaled by
`weights.facet_boost`, default 0.3).

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
    git_sha         VARCHAR,
    facet_text      VARCHAR
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
| `model` | Model identifier: `"lsa-v1"` (variable dim) or `"nomic-v1.5-c8k"` (768 dim) |
| `generated_at` | When the embedding was computed |

**Scoring weights:**
- When nomic is available (3-way): BM25 0.3, LSA 0.2, Nomic 0.5
- When nomic is unavailable (2-way fallback): BM25 0.4, LSA 0.6

---

## `knowledge_chunks`

Heading-anchored sections of tracked prose files at HEAD (markdown / plain
text). Derived and local-only — rebuilt by `rekal index` / watermark refresh
at recall. See [knowledge-layer design](../design/knowledge-layer.md).

```sql
CREATE TABLE IF NOT EXISTS knowledge_chunks (
    id          VARCHAR PRIMARY KEY,
    path        VARCHAR NOT NULL,
    anchor      VARCHAR,
    breadcrumb  VARCHAR,
    start_line  INTEGER NOT NULL,
    end_line    INTEGER NOT NULL,
    content     VARCHAR NOT NULL,
    content_hash VARCHAR NOT NULL,
    blob_sha    VARCHAR NOT NULL
);
```

Guarded FTS over `content` (`fts_main_knowledge_chunks`) — built only when
chunks exist. Recall BM25-caps candidates (≤100), then cosine-re-ranks those
hashes only (no full-corpus embedding scan).

---

## `knowledge_embeddings`

Content-hash + model keyed vectors for knowledge chunks (same embed-cache
path as session embeddings). Filled by background `rekal embed` after
structural index/sync, and by a budgeted bite in the post-commit hook.

```sql
CREATE TABLE IF NOT EXISTS knowledge_embeddings (
    content_hash VARCHAR NOT NULL,
    model        VARCHAR NOT NULL,
    embedding    FLOAT[],
    PRIMARY KEY (content_hash, model)
);
```

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

---

## `session_supersedes`

Maps a collapsed duplicate to the conversation that survived it, so consumers
follow the conversation rather than an id that is no longer in the index.

```sql
CREATE TABLE IF NOT EXISTS session_supersedes (
    old_session_id       VARCHAR PRIMARY KEY,
    survivor_session_id  VARCHAR NOT NULL
);
```

Flattened before it is stored: a value is never itself a key. The index deletes
every key, so a half-resolved chain would strand a subagent's parent or a reach
count on a session that is gone.

---

## `session_reach`

The derived L1 aggregate over `data.db.recall_edges` — how often each session
has been surfaced, and the most recent query that reached it.

```sql
CREATE TABLE IF NOT EXISTS session_reach (
    target_session_id  VARCHAR PRIMARY KEY,
    reach_count        INTEGER NOT NULL,
    last_query         VARCHAR,
    last_ts            TIMESTAMP
);
```

Feeds the optional `reach_boost` ranking layer and the `[reached N×· "query"]`
display hint. Ranking only — deliberately excluded from the silence gate,
because popular is not the same as relevant. Counts follow
`session_supersedes`, so collapsing a duplicate does not reset a well-used
conversation to zero.
