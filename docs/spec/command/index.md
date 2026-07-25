# rekal index

**Role:** Full rebuild of the index DB from the data DB. Builds into `index.db.rebuilding`, then atomically renames over the live index only on success — a failed rebuild leaves the previous index untouched. Safe to run anytime — no data loss; data DB is source of truth.

**Invocation:** `rekal index`.

---

## Preconditions

See [preconditions.md](../preconditions.md): must be in a git repository and init must have been run.

---

## What index does

1. **Run shared preconditions** — Git root, init done.
2. **Open index DB** — Load FTS extension.
3. **Open a fresh temp index** — Create schema in `index.db.rebuilding` (empty file; nothing to drop). The live `index.db` is not opened for write until the final rename.
4. **Populate from data DB** — Attach `data.db` read-only and bulk-insert:
   - `turns_ft` — All turns from `data_db.turns`
   - `tool_calls_index` — All tool calls from `data_db.tool_calls`
   - `files_index` — Files touched, denormalized via `checkpoint_sessions`
   - `session_facets` — Aggregated session metadata (email, branch, actor, counts, checkpoint/SHA)
   - `file_cooccurrence` — Self-join on tool call paths within same session
5. **Cross-repo local import (if enabled)** — When the `local_import` preference in `.rekal/config.json` is set, fold this machine's other agent sessions from every registered adapter (Claude, Cursor, Codex, Gemini, OpenCode, Copilot, Kiro — other repos and shell) into `turns_ft` / `session_facets`, labeled with an `origin` (`repo:`/`shell:`/`local:`). **Index only — never written to `data.db`, so these sessions can be recalled locally but can never be pushed to the team.** Sessions whose content hash already exists in `data.db` are skipped (file-byte hash for Path refs; `sha256("adapter:DBID")` for OpenCode).
6. **Create FTS index** — DuckDB BM25 full-text search on `turns_ft.content` (only if turns exist).
6b. **Build facet documents + facet FTS index** — `PopulateFacetText` rebuilds `session_facets.facet_text` (distinct tool paths + command prefixes + steering text, capped; derived from the index's own tables, so cross-repo imports are covered) after the import step, then the guarded facet FTS index is created — skipped entirely when no session has facet material.
6c. **Build the knowledge layer (structural)** — Chunk the repo's tracked prose files at HEAD into `knowledge_chunks` + guarded knowledge FTS. Watermarked by commit SHA. Non-fatal. Chunk *vectors* are **not** filled here — see step 10. See [knowledge-layer design](../../design/knowledge-layer.md).
7. **LSA pass** — Build LSA model from session content (only if 2+ sessions), store embeddings in `session_embeddings` with model `lsa-v1`. Local and synchronous.
8. **Write index state** — Record `session_count`, `turn_count`, `embedding_dim`, `last_indexed_at`.
9. **Atomic rename** — Replace live `index.db` with the rebuilt temp file.
10. **Print summary + background embed** — `index rebuilt: N sessions, N turns`, then spawn `rekal embed` detached (log: `.rekal/embed.log`). Deep-semantic session vectors and knowledge chunk vectors converge in budgeted bites without blocking the structural rebuild. See [embed.md](embed.md). Keyword / LSA / facet / knowledge-FTS are searchable immediately.

---

## Safe and idempotent

The index DB can be deleted at any time; `rekal index` rebuilds it completely. No data is lost — the data DB is never modified.

---

## Flags

Cross-repo local import — fold your own local agent history (every registered adapter) from other repos and shell sessions on this machine into recall. These flags **set a persistent preference** (stored in `.rekal/config.json`, gitignored) and rebuild. A plain `rekal index` — and `rekal sync` — then **honor** whatever was last set. Default is off. Mutually exclusive.

| Flag | Effect |
|---|---|
| `--include-all` | Import every local agent session on this machine (all agents, repos + shell). |
| `--include <repo>` | Import local sessions for specific repo path(s). Repeatable. |
| `--no-local` | Stop importing; clears the remembered preference. |

Imported sessions are stored in the index only, never in `data.db`, so they are **structurally incapable of being pushed** to the team. They are labeled with an `origin` (`repo:`/`shell:`) in recall output. Turn the choice off with `--no-local`; `rekal clean` also removes the preference.

---

## When to run

- After sync (sync runs index automatically for `--self` mode; team mode rebuilds inline).
- When index is missing or corrupted (`rm .rekal/index.db && rekal index`).
- After manual edits to data DB.

---

## Semantic embeddings and the cache

The deep-semantic pass uses the backend from `.rekal/config.json` — the `embedding` HTTP backend when configured (batched requests, hard per-request timeout, non-fatal on any failure), otherwise the embedded nomic model. The HTTP backend's `provider` is `openai` (default; any OpenAI-compatible `/embeddings` server) or `bedrock` (Amazon Bedrock runtime, Cohere Embed models, bearer API key). Vectors are stored under the backend's model name, which is also recorded in `index_state` (`embed_model`).

A content-hash-keyed cache (`.rekal/embed-cache.db`; `(model, content_hash) → vector`, vectors only, never text) makes rebuilds cheap: only content the model has never seen is embedded. Switching models invalidates by key construction and costs exactly one full re-embed. The cache is best-effort — unopenable/corrupt cache degrades to embedding everything, never to a build failure.
