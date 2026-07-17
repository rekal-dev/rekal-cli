# Benchmark Selection — Targets, Order, Rejections

Decision doc. The landscape it selects from is [00-landscape.md](00-landscape.md).

## Selected, in run order — **full tier for all**

Every row below is a **full-corpus** target. Smoke/limit runs are green lights
only; they are not the published number. See
[`scripts/industry-bench/FULL_TIER.md`](../../scripts/industry-bench/FULL_TIER.md).

| # | Benchmark | Why | Role in the paper | Full-tier bar |
|---|---|---|---|---|
| 1 | **LongMemEval-S** | Cleanest labels of the general benchmarks; five ability categories map onto Rekal's routing story (single-hop → episodic recall, multi-hop → synthesis, temporal → ledger timestamps, knowledge-update → HEAD-vs-history split, abstention → the confidence gate). | Primary headline table | 500 conversations, all 500 questions |
| 2 | **LongMemEval (full / M)** | Same labels at a size that no longer fits a context window — answers the "S is just a context-window test" objection against our own result. | Scale column of the headline table | Full cleaned LongMemEval-M release |
| 3 | **LoCoMo** | The score every vendor page quotes; required for comparability. Never headlined alone; always with the flaw caveat (~6.4% bad labels, gameable judge). | Comparability appendix | All 10 conversations / 1986 questions + official judge |
| 4 | **MSC** | Cheap sanity baseline; catches adapter regressions before expensive runs. | Smoke + regression table (may appear in appendix) | Full MSC multi-session corpus |
| 5 | **BEAM / AMB tiers** | The only token-pressure benchmark; directly exercises the "thin wire, rich machine" design and our tokens-per-question discipline at 10M tokens. Self-reported leaderboard — treat as directional. | Stress section | Published BEAM tiers through 10M |

## Harness strategy

Prefer running *inside* an existing open harness where one supports the
target dataset, because it gives third-party-comparable plumbing for free:

- First choice: **supermemoryai/memorybench** (multi-dataset, designed for
  plugging in systems).
- Second: **mem0ai/memory-benchmarks** (gives an apples-to-apples Mem0
  baseline run in the same harness).
- The LongMemEval official eval scripts are the scoring authority either
  way ([04-procedures](04-procedures.md#scoring)).

If a harness fights the adapter (assumes a hosted API, assumes
add-memory/search-memory verbs), wrap Rekal behind that verb pair — see the
[shim contract](02-adapter-architecture.md#shim) — rather than forking the
harness.

## Rejected / deferred, with reasons

| Benchmark | Verdict | Reason |
|---|---|---|
| **Letta Leaderboard** | Out of scope | Ranks base *models* on memory operations, not memory *systems*; nothing for Rekal to plug into. |
| **PersonaMem** | Deferred | Persona-consistency is the map-mode transfer test, but dataset access is indirect (via the 2604.20006 survey); revisit if WS-C's persona digest shows promise on LongMemEval's preference questions. |
| **GoodAI LTM** | Deferred | Continual-learning framing requires an interactive agent loop, a bigger lift than QA-style benchmarks; not needed for paper #2's claim. |
| **MemoryAgentBench** | Deferred | Same interactive-loop cost; reconsider for paper #3 (agentic memory operations). |
| **LMEB** | Rejected | Embedding-level benchmark; would evaluate nomic-embed, not Rekal. |
| **HotPotQA / MuSiQue etc.** (Cognee's set) | Rejected | Multi-hop QA over Wikipedia is RAG, not memory-over-time; wrong construct. |

## Baselines to run ourselves (not just quote)

Vendor numbers are judge-and-prompt-inconsistent ([00 §5](00-landscape.md)).
The paper's comparison table only contains numbers produced by **our runs in
one pinned harness**:

1. **Mem0 OSS** — the most-quoted system; runnable locally.
2. **Zep / Graphiti OSS** — the graph-memory contrast.
3. **Naive full-context** — stuff everything in the window (upper bound on
   LongMemEval-S; makes the context-window critique quantitative).
4. **Naive RAG** — single dense index over raw turns, no session structure
   (the "is Rekal's structure earning anything?" floor — the same role the
   grep floor played in paper #1).

Vendor self-reported numbers may appear only in a clearly-labeled
"self-reported" column, cited to [00 §5](00-landscape.md).
