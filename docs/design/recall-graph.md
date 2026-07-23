# The Recall Citation Graph (L1)

Rekal's retrieval is flat: BM25 + LSA + Nomic over turns, plus a knowledge
layer over HEAD prose. Every recall is stateless — the system never learns
*which* memory got used or *how* it was reached. L1 records that: a permanent,
local link between a recall query and the sessions it reached, surfaced back as
a hint on future recalls.

Memory that records **how it was accessed**, not just what it contains.

## The edge

An edge is one session an agent reached while working:

- **recall** — a session surfaced in a `rekal "<query>"` result set. Carries the
  query (the intent). Logged for the returned top set (capped).
- **drill** — a session the agent explicitly opened with
  `rekal query --session <id>`. The *strong* signal: the agent voted with its
  actions. No query is attached (the recall that led there is a separate
  process).

The edge is keyed by its **target** — the reached session. From a target you
can ask "what queries reached this, how often"; that is the hint. Attributing
the *source* session (which session did the reaching) — the full session↔session
graph — is a later layer (see Non-goals).

## Why capture at query time (not parsing)

The obvious idea — mine the transcript for `rekal query --session <id>` calls —
does not work. Tool *results* are discarded at parse time, and the command
*string* is truncated to 100 chars in `tool_calls.cmd_prefix`. Agents wrap
calls (`cd … && export PATH … && ./rekal query --session <ULID>`), pushing the
target ID past the cutoff. Empirically: 0 of 273 `rekal` tool_calls in a live
store were recoverable as drills.

So capture happens **inside `rekal` at query time**, where the query and the
real full session IDs (surfaced + drilled) are known before they are lost.

## Storage: permanent record in data.db, off the hot path

```
recall (hot path)          checkpoint (holds data.db writer)      recall read
──────────────────         ─────────────────────────────────     ───────────
graph.Append  ──►  .rekal/recall-log.ndjson  ──► graph.Drain ──► data.db
(lock-free spool)                                InsertRecallEdges  recall_edges
                                                        │           (permanent,
                                                        ▼            append-only,
                                        PopulateSessionReach          local-only)
                                        (index.db.session_reach) ◄── LoadReach
                                                                     (hint)
```

- **`data.db.recall_edges`** is the permanent, append-only record — the source
  of truth. It is **local-only**: deliberately *not* serialized to the codec /
  wire (like `checkpoint_state`), so it never touches the git transport. Being
  additive (`CREATE TABLE IF NOT EXISTS`), it needs no schema-version bump.
- **`index.db.session_reach`** is the derived aggregate the hot read path uses:
  `(target_session_id, reach_count, last_query, last_ts)`, rebuilt from
  `recall_edges` in `PopulateIndex` / `PopulateIndexIncremental`. Created on
  demand (`EnsureReachSchema`) so old index DBs upgrade in place.
- **The spool** (`.rekal/recall-log.ndjson`, gitignored) exists only so the hot
  recall path never grabs data.db's single writer — that would re-couple recall
  to checkpoint/embed. It is a transient write-ahead buffer, drained at
  checkpoint (which already holds the writer), never the store. A partial
  trailing line (an append caught mid-write) is tolerated.

The record lands and the aggregate refreshes exactly when a checkpoint already
owns the lock, so L1 adds zero new lock contention. Between checkpoints the hint
lags by the un-drained spool tail — fine, the graph is cross-session history.

## The hint

Recall reads the reach aggregate for the surfaced seeds **before** logging this
call's own edges (so a session's own recall never inflates the number shown
now), and attaches it as a display-only field. In the digest:

```
INJECT top=0.62 gap=0.05 12 seeds
  s5 conf=0.50 t12 [reached 4×· "jwt expiry"] "…snippet…"
  s8 conf=0.44 t7 "…snippet…"          ← never reached: no suffix
```

`--json` carries a `reached: {count, query}` field (omitempty). On a cold store
every seed is unreached, so the digest is byte-identical to before the feature.

**Display-only by design.** The reach signal is a hint, not a ranking input —
no weight, no silence-gate change, no retune. A recommendation; the agent
judges. (The ranking seam exists — an additive `Weights.CitationBoost` term —
but is intentionally unused at L1.)

## Benchmarks

Capture is gated behind `REKAL_BENCH` / `REKAL_SKIP_CHECKPOINT`
(`session.BenchEnv`), so RekalBench/LoCoMo runs never pollute a store's graph
and stay comparable.

## Non-goals (later layers)

- **Source attribution / session↔session edges.** L1 keys edges by target only.
  Attributing which session did the reaching (full bidirectional graph,
  traversal) needs checkpoint-time reconciliation.
- **Authority-boosted ranking.** Letting reach-count influence the score
  (PageRank-of-memory) via `Weights.CitationBoost`.
- **Team-shared graph.** Sharing the graph over the wire — now a clean switch,
  since the record already lives in data.db: add a `recall_edges` codec frame
  and a merged-only gating decision. This puts each dev's query text + access
  pattern on the shared branch, a deliberate privacy trade to make later.
