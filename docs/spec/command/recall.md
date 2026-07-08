# rekal (root) — recall

**Role:** Hybrid search over captured sessions. Root invocation is recall — no `search` subcommand. Returns scored snippets and metadata — just enough for the agent to decide what matters. For full session content, use `rekal show <session_id>`.

**Invocation:** `rekal [filters...] [query]`. Subcommands (init, clean, checkpoint, etc.) take precedence when present.

**Progressive loading:** Recall is the first step in a two-step context loading model. The agent calls `rekal <query>` to get lightweight results (snippets + metadata), then calls `rekal query --session <id>` for full turns on the sessions that matter. This keeps token usage tight — the agent controls how much context it loads.

---

## Preconditions

See [preconditions.md](../preconditions.md): git repo, init done. If the index is not populated, recall auto-rebuilds it before searching.

---

## What recall does

1. **Run shared preconditions** — Git root, init done.
2. **Open index DB** — Load FTS extension. If index is empty (`last_indexed_at` not set), run a full index rebuild automatically.
3. **Dispatch search mode:**
   - **With query text** → Hybrid search (BM25 + LSA + Nomic combined scoring).
   - **Without query text** → Filter-only search (latest sessions matching filters).
4. **Output** — Structured JSON to stdout. Fields: `results`, `query`, `filters`, `mode`, `total`. Each result's `session` detail carries optional harness metadata (`agent_id`, `team_name`, `workflow_name`, `parent_session_id`) when present, omitted otherwise — grouping/drill-down data, deliberately not a filter surface (see [agent-metadata.md](../../agent-metadata.md)).

---

## Search modes

### Hybrid search (query provided)

1. **BM25 search** — Full-text search on `turns_ft.content`. Returns up to 200 candidate hits scored by BM25.
2. **LSA search** — Rebuild LSA model from session content, project query into embedding space, compute cosine similarity against stored session embeddings. Non-fatal if LSA fails.
3. **Nomic search** — Deep semantic similarity using nomic-embed-text embeddings. Loads stored nomic vectors from index DB, embeds query with "search_query: " prefix, computes cosine similarity. Non-fatal if nomic is unavailable (unsupported platform) or fails.
4. **Group by session** — Pick the best-scoring turn per session. A turn captured from a queue-operation/enqueue steering message (role `human_steering`) has its BM25 score boosted by `steering_boost` (default 1.3×) before this comparison — it is the highest-intent text in the corpus (see [agent-metadata.md](../../agent-metadata.md)).
5. **Normalize and combine** — Normalize all scores to [0,1]. When nomic is available: 3-way scoring (defaults — BM25: 0.35 keyword precision, Nomic: 0.55 semantic understanding, LSA: 0.10 corpus co-occurrence; configurable, see Tuning below). When nomic is unavailable: 2-way fallback — the nomic share falls back to LSA and the pair is renormalized (defaults: BM25 0.35, LSA 0.65). Sessions with a non-null `parent_session_id` (subagent/workflow transcripts) then have their combined score discounted by `subagent_downweight` (default 0.7×), relative to trunk turns of equal textual relevance.
6. **Apply filters** — Actor, author, commit, file regex — all ANDed.
7. **Fold subagent/workflow hits under their trunk conversation** — Sessions are grouped by walking `parent_session_id` to the root. Each group becomes one top-level result (headed by the group's best-scoring turn, which may belong to a descendant transcript), with the rest nested under `children` — capped to 3 per group so one large workflow can't dominate the result budget. A session with no parent and no matching descendants is unaffected: `children` is omitted.
8. **Return top N conversations** — Sorted by each group's best hybrid score descending.

### Filter search (no query)

Query `session_facets` with filter WHERE clauses, ordered by `captured_at DESC`. Returns the first snippet from each session.

---

## Filters

| Flag | Description |
|------|-------------|
| `--file <regex>` | Sessions that touched a file matching the regex (git-root-relative paths) |
| `--commit <sha>` | Sessions linked to a git commit (SHA prefix match) |
| `--author <email>` | Sessions by this author email |
| `--actor <human\|agent>` | Filter by actor type |
| `-n`, `--limit <n>` | Max results (default: 20) |

Multiple filters = AND.

---

## Output format

```json
{
  "results": [
    {
      "session_id": "...",
      "score": 0.85,
      "snippet": "...",
      "snippet_turn_index": 3,
      "snippet_role": "assistant",
      "session": {
        "author": "alice@example.com",
        "actor": "human",
        "branch": "main",
        "captured_at": "2026-02-25T10:00:00Z",
        "commit": "abc123...",
        "turn_count": 12,
        "tool_call_count": 5,
        "files": ["src/auth.go", "src/auth_test.go"]
      },
      "children": [
        {
          "session_id": "...",
          "score": 0.62,
          "snippet": "...",
          "snippet_turn_index": 0,
          "snippet_role": "assistant",
          "session": {
            "actor": "agent",
            "agent_id": "a1b2c3",
            "workflow_name": "release-checklist",
            "parent_session_id": "..."
          }
        }
      ]
    }
  ],
  "query": "JWT expiry",
  "filters": {"file": "", "actor": "", "commit": "", "author": ""},
  "mode": "hybrid",
  "total": 3
}
```

`children` is present only when other matching transcripts (subagent runs,
workflow steps, other agents in the same team) share this result's trunk
conversation via `parent_session_id`; omitted otherwise. `total` counts
top-level (grouped) results, not raw session hits.

`session.origin` is present only on sessions folded in by the cross-repo
local import (`rekal index --include-all` / `--include`): `repo:/path` for
another repo's working directory, `shell:/path` for a non-repo one. Omitted
for this repo's own sessions and synced teammate sessions. It labels where a
cross-context hit came from so an agent (or human) can judge its relevance —
see [cross-repo-import design](../../design/cross-repo-import.md).

---

## Examples

```bash
rekal "JWT"
rekal "JWT expiry"
rekal --file src/auth/middleware.go "JWT"
rekal --file '^src/auth/' "JWT"
rekal --commit a3f9b12 "JWT"
rekal --author alice@example.com "refactor"
rekal --file src/auth.go --actor human "auth"
rekal "JWT" -n 10
```

---

## Tuning (config.json)

Recall reads `.rekal/config.json` (gitignored) at query time:

- **`weights`** — layer mix (`bm25`/`lsa`/`nomic`, normalized ratios; an explicit `0` disables a layer), `steering_boost`, `subagent_downweight`. Applied per query; changing them never requires a reindex. Invalid values fall back to defaults with a warning. When no semantic vectors are available the nomic share falls back to LSA and the pair is renormalized (2-way fallback).
- **`embedding`** — when set, the recall query is embedded by the configured OpenAI-compatible HTTP endpoint instead of the embedded nomic model. The vectors compared against are the ones the index was built with (keyed by model name), so a backend/model mismatch skips the semantic layer — falling back to 2-way scoring — rather than comparing incompatible vectors. Rebuild with `rekal index` after changing model/endpoint.

Failures anywhere in tuning/embedding degrade recall gracefully (fewer layers), never break it.
