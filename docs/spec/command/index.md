# rekal index

**Role:** Full rebuild of the index DB from the data DB. Drops and recreates all index tables, then repopulates from `.rekal/data.db`. Safe to run anytime — no data loss; data DB is source of truth.

**Invocation:** `rekal index`.

---

## Preconditions

See [preconditions.md](../preconditions.md): must be in a git repository and init must have been run.

---

## What index does

1. **Run shared preconditions** — Git root, init done.
2. **Open index DB** — Load FTS extension.
3. **Drop and recreate** — Drop all index tables (`turns_ft`, `tool_calls_index`, `files_index`, `session_facets`, `file_cooccurrence`, `session_embeddings`, `index_state`), then recreate schema.
4. **Populate from data DB** — Attach `data.db` read-only and bulk-insert:
   - `turns_ft` — All turns from `data_db.turns`
   - `tool_calls_index` — All tool calls from `data_db.tool_calls`
   - `files_index` — Files touched, denormalized via `checkpoint_sessions`
   - `session_facets` — Aggregated session metadata (email, branch, actor, counts, checkpoint/SHA)
   - `file_cooccurrence` — Self-join on tool call paths within same session
5. **Create FTS index** — DuckDB BM25 full-text search on `turns_ft.content` (only if turns exist).
6. **LSA pass** — Build LSA model from session content (only if 2+ sessions), store embeddings in `session_embeddings`.
7. **Write index state** — Record `session_count`, `turn_count`, `embedding_dim`, `last_indexed_at`.
8. **Print summary** — `index rebuilt: N sessions, N turns`.

---

## Safe and idempotent

The index DB can be deleted at any time; `rekal index` rebuilds it completely. No data is lost — the data DB is never modified.

---

## No flags

No user-facing flags. Same behaviour every run: full rebuild.

---

## When to run

- After sync (sync runs index automatically for `--self` mode; team mode rebuilds inline).
- When index is missing or corrupted (`rm .rekal/index.db && rekal index`).
- After manual edits to data DB.
