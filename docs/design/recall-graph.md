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
  wire (like `checkpoint_state`), so it never touches the git transport. It is
  ensured in `MigrateDataSchema` (which runs on every open) via
  `CREATE TABLE IF NOT EXISTS`, **not** only in `dataDDL` — an existing store
  written by an older rekal never re-runs the full DDL, so a table added to
  `dataDDL` alone would never appear there. Additive, so no schema-version bump.
- **`index.db.session_reach`** is the derived aggregate the hot read path uses:
  `(target_session_id, reach_count, drill_count, last_query, top_query,
  last_ts)`, rebuilt from `recall_edges` in `PopulateIndex` /
  `PopulateIndexIncremental`. Created on demand (`EnsureReachSchema`) so old
  index DBs upgrade in place, columns included.
  The two counts stay apart because they are different evidence. A **recall
  edge** says only that this engine ranked the session into some window — its
  own past output. A **drill edge** says an agent chose to open it. `top_query`
  is the query that reached the session most often (ties broken by recency);
  `last_query` keeps its literal meaning for anyone querying the table.
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
  s5 conf=0.50 t12 [reached 9× drilled 2×· "jwt expiry"] "…snippet…"
  s6 conf=0.47 t3 [reached 4×· "jwt expiry"] "…snippet…"   ← surfaced, never opened
  s8 conf=0.44 t7 "…snippet…"          ← never reached: no suffix
```

The drill count is printed separately, and only when there is one: "the ranker
keeps offering this" is not the same recommendation as "an agent read this".
`--json` carries a `reached: {count, drills, query}` field (omitempty). On a
cold store every seed is unreached, so the digest is byte-identical to before
the feature.

**Display-only by default.** The reach signal ships as a hint — no silence-gate
change, no retune — and the agent judges. A ranking layer now sits on the
**drill** half of that signal: `weights.reach_boost` (default `0.2`) adds a
max-normalized `drill_count` term to the hybrid score (`hybrid += reach_boost ×
reachNorm`, before the subagent discount, ranking-only — never
`absoluteConfidence`).

Ranking on drills rather than on every edge is deliberate. Recall returns ~20
seeds per call, so on any store smaller than a few hundred sessions a single
query marks most of the corpus: measured on a 37-session store, 36 sessions
carried reach (median 22.5), the top slot was a three-turn session and an empty
session sat at 36 — while the corpus held 741 recall edges against 6 drills.
Boosting on that is the ranker rewarding whatever it surfaced before, noise
included. A drill is evidence from outside the ranker.

The layer is self-activating — a cold store has no drill edges, so ranking is
byte-identical until agents start drilling — and `0` disables the
`session_reach` lookup entirely. This is the first realized step of the
authority-ranking direction below; the full session↔session PageRank remains a
later layer.

## Benchmarks

Capture is gated behind `REKAL_BENCH` / `REKAL_SKIP_CHECKPOINT`
(`session.BenchEnv`), so RekalBench/LoCoMo runs never pollute a store's graph
and stay comparable.

## Non-goals (later layers)

- **Source attribution / session↔session edges.** L1 keys edges by target only.
  Attributing which session did the reaching (full bidirectional graph,
  traversal) needs checkpoint-time reconciliation.
- **Full authority-boosted ranking.** The opt-in `weights.reach_boost` layer
  (above) is a first step — a flat max-normalized reach term. Genuine
  PageRank-of-memory (propagating authority across session↔session edges) still
  needs the source-attribution graph and is a later layer.
- **Team-shared graph.** Sharing the graph over the wire — now a clean switch,
  since the record already lives in data.db: add a `recall_edges` codec frame
  and a merged-only gating decision. This puts each dev's query text + access
  pattern on the shared branch, a deliberate privacy trade to make later.
