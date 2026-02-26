# rekal (root) — recall

**Role:** Hybrid search over captured sessions. Root invocation is recall — no `search` subcommand. When `rekal` is called with a query or filter flags and no recognised subcommand, it runs recall against the index DB. Primary consumer is the agent.

**Invocation:** `rekal [filters...] [query]`. Subcommands (init, clean, checkpoint, etc.) take precedence when present.

---

## Preconditions

See [preconditions.md](../preconditions.md): git repo, init done. If the index is not populated, recall auto-rebuilds it before searching.

---

## What recall does

1. **Run shared preconditions** — Git root, init done.
2. **Open index DB** — Load FTS extension. If index is empty (`last_indexed_at` not set), run a full index rebuild automatically.
3. **Dispatch search mode:**
   - **With query text** → Hybrid search (BM25 + LSA combined scoring).
   - **Without query text** → Filter-only search (latest sessions matching filters).
4. **Output** — Structured JSON to stdout. Fields: `results`, `query`, `filters`, `mode`, `total`.

---

## Search modes

### Hybrid search (query provided)

1. **BM25 search** — Full-text search on `turns_ft.content`. Returns up to 200 candidate hits scored by BM25.
2. **LSA search** — Rebuild LSA model from session content, project query into embedding space, compute cosine similarity against stored session embeddings. Non-fatal if LSA fails.
3. **Group by session** — Pick the best-scoring turn per session.
4. **Normalize and combine** — Normalize BM25 and LSA scores to [0,1], combine with weights (BM25: 0.4, LSA: 0.6).
5. **Apply filters** — Actor, author, commit, file regex — all ANDed.
6. **Return top N** — Sorted by hybrid score descending.

### Filter search (no query)

Query `session_facets` with filter WHERE clauses, ordered by `captured_at DESC`. Returns the first snippet from each session.

---

## Filters

| Flag | Description |
|------|-------------|
| `--file <regex>` | Sessions that touched a file matching the regex (git-root-relative paths) |
| `--commit <sha>` | Sessions linked to a git commit (SHA prefix match) |
| `--checkpoint <ref>` | Reserved for future use |
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
      }
    }
  ],
  "query": "JWT expiry",
  "filters": {"file": "", "actor": "", "commit": "", "author": ""},
  "mode": "hybrid",
  "total": 3
}
```

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
