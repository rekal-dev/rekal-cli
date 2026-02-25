# rekal index

**Role:** Full rebuild of the index DB from the data DB. Drops and recreates all index tables, then repopulates from `.rekal/data.db`. The index supports both **fulltext search** (e.g. turn-level FTS) and **semantic/vector search** (e.g. session embeddings). Normally the index is updated **incrementally on each commit** (by checkpoint); use `rekal index` when the index is missing, stale, corrupted, or after sync. Safe to run anytime — no data loss; data DB is source of truth.

**Invocation:** Subcommand only — `rekal index`.

---

## Preconditions

See [preconditions.md](../preconditions.md): must be in a git repository and init must have been run. Otherwise exit with the shared message (run init) or warning (not a git repo).

---

## Index state

The index layer **knows its state**: what has already been indexed (e.g. which checkpoint_ids or session_ids are present in the index DB, or a single "indexed up to" marker). That state is used to:

- **Incremental update on commit** — When `rekal checkpoint` runs, it writes to the data DB then asks the index to update incrementally (only the new checkpoint and its sessions). No full rebuild.
- **Full rebuild** — When state is missing or invalid, or when incremental is not possible (e.g. after sync, index deleted, or corruption), `rekal index` does a full rebuild and resets state.

So: default path is incremental (checkpoint); full rebuild is the recovery path.

---

## What index does (full rebuild)

1. **Run shared preconditions** — Git root, init done.
2. **Drop and recreate** — DROP all tables in `.rekal/index.db`; run index DB DDL so the schema is fresh.
3. **Stream from data DB** — Read all sessions (and related data) from `.rekal/data.db`.
4. **Populate index tables** — For each session: extract turns → `turns_ft`, tool calls → `tool_calls_index`, facets → `session_facets`. Copy `files_touched` → `files_index`. Compute file co-occurrence from tool calls → `file_cooccurrence`. (Phase 4: embeddings → `session_embeddings`.)
5. **Rebuild indexes** — Recreate all fulltext (FTS), vector/semantic, and B-tree indexes on the index DB so recall can use both keyword and semantic search.
6. **Set state** — Record that the index is now in sync with the data DB (e.g. all checkpoints/sessions indexed).
7. **Exit** — Print a short summary (e.g. session count, turn count, elapsed time). Success or clear error.

---

## Scenarios

| Scenario | Behaviour |
|----------|-----------|
| **On commit** | `rekal checkpoint` writes data DB then triggers **incremental** index update (only new checkpoint/sessions). Index state is updated. |
| **Index missing or empty** | Precondition step or first recall/query runs index (full rebuild) so the index exists. |
| **Index corrupted or stale** | User runs `rekal index` → full rebuild. |
| **After sync** | New data from team; index state does not cover it. Run full rebuild (`rekal index`) or implementation may support incremental merge from new data. |
| **Recall/query** | Need index. If index missing or empty, run index (full rebuild) then proceed. If index exists and has state, use it. |

---

## Safe and idempotent

The index DB can be deleted at any time; `rekal index` rebuilds it completely. No data is lost — the data DB is never modified. Re-running index is idempotent in outcome: you get a full, consistent index.

---

## No flags

No user-facing flags. Same behaviour every run: full rebuild.

---

## When to run

- After init (init may create an empty index; first recall/query triggers index or you run it explicitly).
- After sync (other commands may run index automatically; otherwise run `rekal index` to refresh).
- When index is missing or corrupted (e.g. `rm .rekal/index.db && rekal index`).

The skill tells the agent not to run `rekal index` mid-session unless the user explicitly asks — rebuild is disruptive to in-flight recall.
