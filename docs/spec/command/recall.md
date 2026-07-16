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
3b. **Facet search** — BM25 over each session's facet document (`session_facets.facet_text`: distinct tool paths + command prefixes + steering text, built at index time — no LLM). Answers structural questions whose evidence lives in tool-call metadata the turns never mention. Runs only when `facet_boost > 0` (default 0.3; an explicit 0 makes ranking byte-identical to the pre-facet engine); fails soft to nothing when the facet FTS index is absent (older index.db, or a corpus with no facet material — the index build is guarded).
4. **Group by session** — Pick the best-scoring turn per session. A turn captured from a queue-operation/enqueue steering message (role `human_steering`) has its BM25 score boosted by `steering_boost` (default 1.3×) before this comparison — it is the highest-intent text in the corpus (see [agent-metadata.md](../../agent-metadata.md)). A harness-written compaction summary (role `summary`) is boosted by `summary_boost` (default 1.15×) — the densest recall anchor in the corpus, but machine text, so it stays below steering.
5. **Normalize and combine** — Normalize all scores to [0,1]. When nomic is available: 3-way scoring (defaults — BM25: 0.35 keyword precision, Nomic: 0.55 semantic understanding, LSA: 0.10 corpus co-occurrence; configurable, see Tuning below). When nomic is unavailable: 2-way fallback — the nomic share falls back to LSA and the pair is renormalized (defaults: BM25 0.35, LSA 0.65). The facet layer is then added as a fourth, additive term — `hybrid += facet_boost × facetNorm` (default 0.3) — before sessions with a non-null `parent_session_id` (subagent/workflow transcripts) have their combined score discounted by `subagent_downweight` (default 0.7×), relative to trunk turns of equal textual relevance.
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
| `--explain` | Adds per-layer scores (`layers`: bm25/lsa/nomic/facet, normalized, pre-weight) and `related` (sessions sharing touched files, query-time join) to each result |

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
      "summary_turn_index": 41,
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

`summary_turn_index` points at the session's latest compaction-summary turn
(role `summary`) when one exists; omitted otherwise. It is a pointer, not a
payload — the summary itself is 10-17KB and is never inlined into recall
output (progressive disclosure). Drill it with
`rekal query --session <id> --role summary`.

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

Recall reads config at query time from two tiers, highest-to-lowest:
**local `.rekal/config.json` → global `~/.config/rekal/config.json` →
built-in defaults.** The global file (path honors `$REKAL_CONFIG_HOME` then
`$XDG_CONFIG_HOME`, else `~/.config/rekal/`) supplies machine-wide defaults so
a backend or tuned weights can be set once for every repo. The merge is
per-key: **`embedding` inherits wholesale** (a repo either uses the global
backend or replaces the whole block), **`weights` merge field-by-field** (a
repo can override just `bm25` and inherit the rest), and **`local_import` is
not inherited** — cross-repo import intent stays per-repo. Both files are
gitignored/local-only and never pushed or synced; the write path (`rekal
index --include-all/--include/--no-local`) only ever touches the local file,
so global values are never baked into a repo.

- **`weights`** — layer mix (`bm25`/`lsa`/`nomic`, normalized ratios; an explicit `0` disables a layer), `steering_boost`, `summary_boost`, `subagent_downweight`, and `facet_boost` (additive facet-layer scale; default 0.3, `0` disables the layer and makes ranking byte-identical to the pre-facet engine). Applied per query; changing them never requires a reindex. Invalid values fall back to defaults with a warning. When no semantic vectors are available the nomic share falls back to LSA and the pair is renormalized (2-way fallback).
- **`embedding`** — when set, the recall query is embedded by the configured HTTP backend instead of the embedded nomic model. `provider` selects the wire protocol: `openai` (default) for any OpenAI-compatible `/embeddings` server (vLLM, Ollama, LM Studio, TEI, or a gateway), or `bedrock` for the Amazon Bedrock runtime (Cohere Embed models — `cohere.embed-english-v3`/`cohere.embed-multilingual-v3` — authenticated by a Bedrock API key as the bearer token, no SigV4). Under `openai`, a **Cohere Embed model** (model name contains `cohere`, e.g. served through a gateway in front of Bedrock) automatically gets Cohere's required `input_type` in the request body — `search_query` for the query, `search_document` for stored turns — so Cohere works over the plain `/embeddings` shape without text prefixes; other models omit it. For `bedrock`, `endpoint` is the runtime host (`https://bedrock-runtime.<region>.amazonaws.com`) and the same asymmetry rides `input_type` natively. The API key supports a real string (`api_key`), a `$VAR` reference inside `api_key`, or an explicit env var name (`api_key_env`, which wins when set and non-empty); absent key ⇒ no Authorization header. `endpoint` expands `$VAR` the same way. The vectors compared against are the ones the index was built with (keyed by model name), so a backend/model mismatch skips the semantic layer — falling back to 2-way scoring — rather than comparing incompatible vectors. Rebuild with `rekal index` after changing provider/model/endpoint.

Failures anywhere in tuning/embedding degrade recall gracefully (fewer layers), never break it. A query embedder / index model mismatch (e.g. index built with Cohere HTTP embeddings but recall falling back to embedded nomic) is recorded in scoring-lineage `skipped.nomic` with the index's model list — it is never a silent empty skip. When `embedding` is configured but fails to resolve, recall does **not** fall back to embedded nomic (that was the mismatch footgun); fix the config or rebuild the index for the embedder you intend.

### Scoring lineage (global-only diagnostics)

Observe-only NDJSON logging of each retrieval layer's contribution and
stage timings. **Off by default** — when disabled, no timers run and
ranking is byte-identical to a lineage-free build. **Global-only**: set it
in `~/.config/rekal/config.json` (honors `$REKAL_CONFIG_HOME` /
`$XDG_CONFIG_HOME`). A value in `.rekal/config.json` is ignored, and the
write path never persists it into a repo file — diagnostics stay a
machine-local developer switch.

```json
{
  "scoring_lineage": {
    "enabled": true,
    "path": "scoring-lineage.ndjson",
    "max_candidates": 50,
    "rotation": {
      "max_megabytes": 10,
      "max_backups": 5,
      "max_age_days": 14,
      "compress": true
    }
  }
}
```

| Field | Default | Meaning |
|-------|---------|---------|
| `enabled` | `false` | Master switch |
| `path` | empty → **stderr** | Append NDJSON here. Relative paths resolve under the global config dir; absolute paths are used as-is. Local file only — never a network sink. |
| `max_candidates` | `50` | Cap per-session `candidate` events (top of the pre-group ranked pool) |
| `rotation` | 10 MB / 5 backups / 14 days / gzip | Size-based roller (lumberjack) when `path` is set. Ignored for stderr. |

Every line is one JSON object with a shared envelope:

```json
{"ts":"2026-07-16T03:00:00.000Z","v":1,"run_id":"a1b2c3d4e5f60708","event":"query", ...}
```

`run_id` joins all events from one recall. Schema version is `v` (currently 2).

Each hybrid recall emits, in order:

1. **`query`** (start) — query string, mode, filters, weights, intended
   `embedder_backend` (`http` | `embedded`) and `embedder_model`.
2. **`candidate`** (mid, ≤ `max_candidates`) — per session: raw + normalized
   scores for bm25 / lsa / nomic / facet, weighted contributions, role boost
   on the winning turn, subagent discount, final hybrid score.
3. **`result`** (end) — final post-group `returned` rows (rank, session_id,
   score, snippet_turn_index), `counts` (`bm25_hits`, `candidates`,
   `after_filter`, `after_group`, `returned`, …), `timings_ms` (`bm25`,
   `lsa`, `nomic`, `embed_query`, `facet`, `combine`, `build`, `group`,
   `total`), `semantic` (`used`, `backend`, `model` — whether the deep
   vector layer scored and which embedder did it; known only after search),
   `tokens` (`embed_query_chars`, `payload_bytes`), and skip reasons when
   a layer soft-failed.

The hybrid weight / timing / skip key `nomic` is the historical name of the
deep semantic *layer*, not the model. Prefer `semantic.model` /
`embedder_model` for the real identity (`nomic-v1.5`, Cohere, …).

Stdout JSON is unchanged (agents keep parsing it). Lineage is a separate
stream from `--explain` (which adds thin `layers`/`related` fields on
results); they can run together.
