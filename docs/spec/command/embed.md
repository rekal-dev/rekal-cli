# rekal embed

**Role:** Fill missing deep-semantic vectors for sessions and knowledge
chunks. Resumable, budgeted, lock-friendly.

**Invocation:** `rekal embed`.

Also started automatically in the background by `rekal index` and
`rekal sync` after the structural rebuild finishes.

---

## Preconditions

See [preconditions.md](../preconditions.md): must be in a git repository and
init must have been run.

---

## Why a separate command

Structural indexing (FTS, facets, LSA, knowledge *chunks*) is local and
fast. Deep semantic vectors (HTTP Cohere / embedded nomic, plus knowledge
chunk vectors) are network-bound and already soft-fail — recall degrades to
keyword/LSA without them. Holding `rekal index` open for thousands of
embedding round-trips blocks the structural rebuild and, because DuckDB is
single-writer, also blocks recall.

So:

| Tier | When | Failure |
|------|------|---------|
| Structural | Synchronous on `index` / `sync` / checkpoint | Hard — this *is* the index |
| Semantic vectors | Background `embed` (budgeted bites) | Soft — keyword/LSA keep working |

---

## What embed does

1. Acquire `.rekal/embed.lock` (exclusive, non-blocking). If another embed
   is running, exit quietly.
2. Loop budgeted bites until nothing remains:
   - Open `index.db`
   - Session vectors: store cache hits for every session missing a vector
     under the configured model; embed up to 16 *uncached* contents per bite
     (`sessionEmbedBudget`)
   - Knowledge vectors: embed up to 16 chunks missing vectors per bite
     (`knowledgeEmbedBudget`; same as the post-commit hook)
   - Close `index.db` (release the write lock so recall/checkpoint can run)
   - Brief yield, then next bite
3. Progress and errors append to `.rekal/embed.log` when spawned from
   `index`/`sync`; to stderr when run by hand.

Safe to interrupt and re-run. The content-hash embed cache makes repeats
cheap.
