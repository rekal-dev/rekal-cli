# Source List — Agent Memory, Knowledge Bases, and Retrieval (2026)

Compiled from a working discussion synthesizing recent literature on LLM agent memory,
knowledge-base retrieval, self-improvement, and experience learning. All links verified
against live search (not recalled from training) as of July 2026. Organized by theme, in
roughly the order the discussion covered them.

---

## Core papers (deep-read / discussed in depth)

**1. Retrospective Harness Optimization (RHO)**
Improving LLM agents by re-solving a diverse, difficulty-balanced subset of past tasks in
parallel, diagnosing failures via self-consistency/disagreement, and accepting harness updates
only via head-to-head self-preference over the incumbent.
- arXiv: https://arxiv.org/abs/2606.05922
- HTML: https://arxiv.org/html/2606.05922

**2. Retrieval as Reasoning — LLM-Wiki**
Compiles a document corpus into structured, bidirectionally linked wiki pages; agent traverses
via search/read/follow-link tools; self-correcting "Error Book" repairs recurring construction
errors. Operationalizes Karpathy's "LLM Wiki" pattern.
- arXiv: https://arxiv.org/abs/2605.25480
- HTML: https://arxiv.org/html/2605.25480

**3. AdaMem — Adaptive User-Centric Memory for Long-Horizon Dialogue Agents**
Working/episodic/persona/graph memory tiers; question-conditioned retrieval routing; write-path
protected from read-path (recency-ordered consolidation); multi-agent (Memory/Research/Working)
answer pipeline. SOTA on LoCoMo and PERSONAMEM.
- arXiv: https://arxiv.org/abs/2603.16496
- Authors: Shannan Yan, Jingchen Ni, Leqi Zheng, et al. — Tsinghua University / WeChat Vision, Tencent

**4. MRAgent — "Memory is Reconstructed, Not Retrieved: Graph Memory for LLM Agents"**
Cue–Tag–Content associative memory graph; active multi-step reconstruction (not passive
lookup); formal theorem proving active retrieval strictly more powerful than passive retrieval
(separation via Binary-Tree Needle-in-a-Haystack construction); finding that reasoning depth
cannot be substituted by parallel retrieval breadth.
- arXiv: https://arxiv.org/abs/2606.06036
- HTML: https://arxiv.org/html/2606.06036
- GitHub: https://github.com/Ji-shuo/MRAgent
- Authors: Shuo Ji, Yibo Li, Bryan Hooi — National University of Singapore (ICML 2026)

**5. EMem — "A Simple Yet Strong Baseline for Long-Term Conversational Memory of LLM Agents"**
Argues against aggressive compression; represents history as fine-grained, self-contained
elementary discourse units (EDUs) with normalized entities and source-turn attribution, organized
in a heterogeneous graph for associative recall.
- arXiv: https://arxiv.org/abs/2511.17208
- GitHub: https://github.com/KevinSRR/EMem
- Authors: Sizhe Zhou, Jiawei Han — University of Illinois Urbana-Champaign

**6. SAG — "SQL-Retrieval Augmented Generation with Query-Time Dynamic Hyperedges"**
Replaces offline knowledge-graph construction with query-time relational (SQL join) expansion:
each chunk becomes one event plus indexing entities, written to SQL/vector/full-text indexes;
hyperedges instantiated dynamically per query via reverse joins, not precomputed. Cheap,
append-only-friendly incremental updates.
- arXiv: https://arxiv.org/abs/2606.15971
- GitHub: https://github.com/Zleap-AI/SAG
- Authors: Yuchao Wu, Junqin Li, XingCheng Liang, et al. — Zleap AI

**7. EDV — "Escaping the Self-Confirmation Trap: An Execute-Distill-Verify Paradigm for Agentic
Experience Learning"**
Names the Self-Confirmation Trap (wrong-but-self-consistent trajectories mistaken for success,
amplified through memory reuse). Decouples Execute (heterogeneous parallel agents) → Distill
(third-party, not self-summarization) → Verify (consensus-gated admission to memory).
- arXiv: https://arxiv.org/abs/2606.24428
- HTML: https://arxiv.org/html/2606.24428
- Authors: Shiding Zhu, Yudi Qi, Yajie Wang, et al.

**8. AutoMem — "Automated Learning of Memory as a Cognitive Skill"**
Treats memory management itself as a trainable skill (metamemory): promotes file-system
operations to first-class memory actions; a strong LLM reviews complete trajectories and
iteratively revises the memory structure (prompts/schemas/action vocabulary) that shapes how the
agent uses memory, automating both structure design and executor proficiency.
- arXiv: https://arxiv.org/abs/2607.01224
- HTML: https://arxiv.org/html/2607.01224
- Project site: https://autolearnmem.github.io/
- Authors: Shengguang Wu, Hao Zhu, Yuhui Zhang, Xiaohan Wang, Serena Yeung-Levy — Stanford University

---

## Retrieval-interface and traversal literature (supporting)

**9. Direct Corpus Interaction (DCI) — "Beyond Semantic Similarity: Rethinking Retrieval for
Agentic Search via Direct Corpus Interaction"**
Agent searches the raw corpus directly with general-purpose terminal tools (grep, file reads,
shell commands) — no embedding model, vector index, or retrieval API. Outperforms strong
sparse/dense/reranking baselines on several benchmarks; requires no offline indexing.
- arXiv: https://arxiv.org/abs/2605.05242

**10. GrepSeek — "Training Search Agents for Direct Corpus Interaction"**
Trains search agents to use shell-command corpus interaction directly; outperforms non-agentic
and untrained/trained agentic retrieval baselines on 4/7 QA benchmarks, especially multi-hop.
- arXiv: https://arxiv.org/abs/2605.29307

**11. RISE — "Towards Retrieving Interaction Spaces for Agentic Search"**
Argues unbounded DCI doesn't scale; proposes constructing a bounded "interaction space" via
retrieval (e.g., BM25) that the agent then explores with shell-style tools.
- arXiv: https://arxiv.org/abs/2606.06880

**12. Learning to Retrieve from Agent Trajectories (LRAT)**
Mines retrieval supervision directly from agent behavioral signals (browsing actions, unbrowsed
rejections, post-browse reasoning) rather than human click logs; trains retrievers that improve
evidence recall and task success across agent architectures.
- arXiv: https://arxiv.org/abs/2604.04949
- HTML: https://arxiv.org/html/2604.04949v1
- Project site: https://yuqi-zhou.github.io/LRAT-homepage/

---

## Temporal belief, freshness, and drift

**13. BeliefShift — "Benchmarking Temporal Belief Consistency and Opinion Drift in LLM Agents"**
Longitudinal benchmark (2,400 human-annotated multi-session trajectories) evaluating belief
dynamics — temporal consistency, contradiction detection, evidence-driven revision — rather than
treating user information as static facts. Introduces BRA, DCS, CRR, ESI metrics.
- arXiv: https://arxiv.org/abs/2603.23848

---

## Governance, security, and contamination

**14. "A Survey on Long-Term Memory Security in LLM Agents: Attacks, Defenses, and Governance
Across the Memory Lifecycle"**
Maps contamination propagation through persistence, statefulness, and propagation (lateral
inter-agent, vertical user-to-org, temporal session-to-session); catalogs write-path attacks
(AgentPoison, InjecMEM, MINJA, MemoryGraft, eTAMP) and the "incomplete forgetting" problem.
- arXiv: https://arxiv.org/abs/2604.16548
- HTML: https://arxiv.org/html/2604.16548v2

**15. Governed Memory — "A Production Architecture for Multi-Agent Workflows"**
Identifies five structural failures of enterprise agent memory without shared governance:
silos, governance fragmentation, unstructured memories unusable downstream, redundant context
delivery, silent quality degradation. Proposes dual memory model + tiered governance routing.
- arXiv: https://arxiv.org/abs/2603.17787
- Author: Hamed Taheri — Personize.ai

**16. "State Contamination in Memory-Augmented LLM Agents"**
Introduces the sub-threshold propagation gap (SPG): shows raw transcript reuse drives overt
toxicity while compressed memory carries hidden sub-threshold influence; sanitizing before
summarization works, sanitizing only the completed summary leaves contamination intact.
- arXiv: https://arxiv.org/abs/2605.16746

**17. "From Agent Traces to Trust: A Survey of Evidence Tracing and Execution Provenance in LLM
Agents"**
Survey covering memory provenance under conflict/contradiction/contamination — stored memories
that support incompatible claims, become invalidated by later evidence, or encode poisoned
content later reused as trustworthy context.
- arXiv: https://arxiv.org/abs/2606.04990

---

## Notes for the research agent

- All arXiv IDs above were confirmed via live web search in July 2026, not recalled from
  parametric memory — treat them as verified pointers, but re-check before citing formally, as
  preprints may update versions.
- Papers 1–8 are the core set with deep engagement (full or substantial partial reads); papers
  9–17 are supporting/contextual and were read at abstract depth.
- Several of these papers (notably 4, 5, and 6) are in active tension on the
  compile-time-vs-query-time-structure axis and the compression-vs-preservation axis — useful as
  a starting contrast set if the agent is asked to map the design space rather than summarize
  individual systems.
- The single most-repeated open problem across papers 2, 3, 4, and 6 is freshness /
  maintenance of derived structure over time — each defers or only partially addresses it.

## 18. Are We Ready For An Agent-Native Memory System? (added post-v0.1)
- Zhou, Wei, et al. (Shanghai Jiao Tong). arXiv 2606.24775, June 2026.
- https://arxiv.org/abs/2606.24775
- Data-management evaluation: four-module anatomy (representation & storage,
  extraction, retrieval & routing, maintenance); 12 systems + 2 baselines,
  5 workloads / 11 datasets; finds no single architecture dominates —
  effectiveness depends on aligning memory structure with the workload.
- Role here: the anchor survey — its anatomy opens the paper's intro
  (requirements under the token constraint), its no-free-lunch verdict is
  the premise, and its maintenance column is the design space Rekal exits.
