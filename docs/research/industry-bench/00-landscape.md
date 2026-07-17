# Memory Benchmarks Landscape — Reference

Compiled: 2026-07-17 (operator-supplied survey, transcribed into the program).
Purpose: the raw map an agent uses to (1) select a benchmark, (2) build a
Rekal adapter, (3) run comparative tests. Selection *decisions* live in
[01-selection.md](01-selection.md); this file is the unfiltered landscape.

---

## 1. General conversational / personal memory benchmarks

Test long-term recall of facts a user told a chat assistant across multiple
sessions. No code, no commits — pure dialogue.

| Benchmark | What it tests | Repo / data location | Notes |
|---|---|---|---|
| **LoCoMo** | Very long-term conversational memory across many sessions | Cited via arXiv; dataset released alongside paper (search "LoCoMo dataset github") | Widely used, but audited in 2026: ~6.4% of answer key is wrong; judge accepts up to 63% of intentionally wrong answers — treat scores with caution |
| **LongMemEval / LongMemEval-S** | 5 core long-term memory abilities (single-hop, multi-hop, temporal, knowledge update, abstention) | https://openreview.net/forum?id=pZiyCaVuti · https://arxiv.org/html/2410.10813v2 | LongMemEval-S often fits inside modern context windows — partly a context-window test rather than a true memory test |
| **PersonaMem** | Personalized long-horizon persona-consistency QA | Referenced in "Benchmarking Long-Term Memory for Personalized Agents" (arXiv 2604.20006) | Good fit for persona/diary-style map testing |
| **MSC (Multi-Session Chat)** | Classic multi-session dialogue recall | Older; still cited as baseline (see arXiv 2402.17753) | Simpler; useful as a sanity-check baseline |

Reference survey: "Benchmarking Long-Term Memory for Personalized Agents" —
https://arxiv.org/html/2604.20006v1

## 2. Unified / open-source benchmark harnesses

Ready-to-run code bundling multiple datasets + eval scripts — best starting
point for plugging in a new system.

| Harness | Covers | Repo |
|---|---|---|
| **mem0ai/memory-benchmarks** | Open-source eval suite for memory-augmented LLM systems | https://github.com/mem0ai/memory-benchmarks |
| **supermemoryai/memorybench** | Unified benchmark for conversational memory + RAG across multiple datasets | https://github.com/supermemoryai/memorybench |
| **GoodAI LTM Benchmark** | Long-term memory + continual learning for LLM agents | https://github.com/GoodAI/goodai-ltm-benchmark |
| **rohitg00/agentmemory** | Persistent memory for AI coding agents; includes LongMemEval runner | https://github.com/rohitg00/agentmemory/blob/main/benchmark/LONGMEMEVAL.md |
| **MemoryAgentBench** | Unified suite: long-term, interactive, adaptive memory via multi-turn interactions | https://www.emergentmind.com/topics/memoryagentbench |

## 3. Scale / stress benchmarks (token-pressure)

| Benchmark | Description | Location |
|---|---|---|
| **Agent Memory Benchmark (AMB) / BEAM tiers** | Tiered stress test (e.g., BEAM 10M) — retrieval under heavy token pressure | https://automem.ai/benchmarks/ |
| **LMEB** | Embedding-level long-horizon retrieval | https://arxiv.org/html/2603.12572v3 |

Known BEAM 10M reference scores (self-reported, directional only):
Hindsight ~64.1% · AutoMem ~57.4% · Honcho ~40.6%.

## 4. Model-level (not tool-level) leaderboard

| Leaderboard | What it ranks | Location |
|---|---|---|
| **Letta Leaderboard** | Base LLMs on agentic memory *operations* over synthetic long-horizon tasks | https://www.letta.com/blog/letta-leaderboard/ |

## 5. Vendor self-reported comparison pages (use cautiously — inconsistent judges/prompts)

| Source | Claim | Location |
|---|---|---|
| Zep vs Mem0 | Zep 94.7% LoCoMo / 90.2% LongMemEval vs Mem0 91.6% / 90.4% (top_50) | https://blog.getzep.com/lies-damn-lies-statistics-is-mem0-really-sota-in-agent-memory/ |
| OMEGA vs Mem0 vs Zep | OMEGA 95.4% LongMemEval vs Zep 71.2% | https://omegamax.co/blog/omega-vs-mem0-vs-zep |
| Cognee vs Mem0/Graphiti/LightRAG | Head-to-head on HotPotQA, TwoWikiMultiHop, MuSiQue | https://www.cognee.ai/blog/deep-dives/knowledge-graph-memory-benchmarks |
| AutoMem "Honest Comparison" | Mem0, Zep, Letta, and others compared | https://automem.ai/blog/agent-memory-in-2026-an-honest-comparison-of-mem0-zep-letta-and-the-rest |
| Particula.tech test | Mem0 vs Zep vs Letta vs Cognee | https://particula.tech/blog/agent-memory-frameworks-tested-mem0-zep-letta-cognee-2026 |
| CortexDB | Reproducible LongMemEval-S / LoCoMo repro path (one-command) | https://cortexdb.ai/docs/research/benchmark-paper |

## 6. Known benchmark flaws (read before trusting any single score)

- **LoCoMo**: answer-key audit found ~6.4% incorrect labels; the judge
  accepts intentionally wrong answers up to 63% of the time.
  https://dev.to/penfieldlabs/we-audited-locomo-64-of-the-answer-key-is-wrong-and-the-judge-accepts-up-to-63-of-intentionally-33lg
- **LongMemEval-S** often fits entirely within modern context windows, so it
  partly measures context-window size rather than memory retrieval.
  https://www.reddit.com/r/AIMemory/comments/1s1jlnd/serious_flaws_in_two_popular_ai_memory_benchmarks/
- LoCoMo and LongMemEval measure structurally different capabilities — do
  not treat scores as interchangeable.
  https://agents.stackoverflow.com/tils/5561fef0-8306-4deb-9c86-ed935a212707

## 7. Coding-agent-specific benchmark (Rekal's own)

| Benchmark | Description | Location |
|---|---|---|
| **RekalBench** | Self-labeled, mined from git commit↔session links; labels generated on any user's own repo at zero cost | This repo, `docs/research/` + `scripts/bench/` |

Naming hazard: `janbjorge/rekal` (SQLite semantic memory for LLM
conversations) is an unrelated project — do not confuse or cite it as ours.

## 8. Original adapter sketch (superseded)

The operator's original 6-step adapter sketch (synthetic commit generator →
map-as-persona-digest → episodic recall → synthesis for knowledge-update →
collapsed router → metric alignment) is preserved here as provenance; the
binding version, with contracts and decisions, is
[02-adapter-architecture.md](02-adapter-architecture.md).
