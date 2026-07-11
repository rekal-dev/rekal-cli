# Literature map: 17 papers → what each means for Rekal

Papers 1–8 deep-read; 9–17 at abstract depth. Links in `00-sources.md`.
Each entry: what it shows → **verdict for Rekal** (SUPPORT / THREAT / STEAL /
FRAME) → the action it implies.

The set is organized by two axes the papers themselves fight over, plus one
shared open problem:

- **Compile-time vs query-time structure** — build a knowledge graph/wiki
  offline (LLM-Wiki, MRAgent, EMem-G) vs. instantiate structure per-query
  from flat indexes (SAG, DCI/RISE). *Rekal is query-time*: DuckDB indexes +
  query-time weights + per-query drill; the only "compiled" artifacts
  (embeddings) are cache-invalidated by content hash.
- **Compression vs preservation** — summarize aggressively (most dialogue
  systems) vs. keep fine-grained units with attribution (EMem). *Rekal
  preserves*: full turns, append-only, snippet + `snippet_turn_index`
  pointing back to source.
- **Shared open problem: freshness/maintenance of derived structure** —
  deferred by papers 2, 3, 4, 6. *Rekal's disposable-index architecture is
  the structural answer* (see `01`).

---

## Core set

### 1. RHO — Retrospective Harness Optimization (arXiv 2606.05922)
Improves an agent *harness* from unlabeled past trajectories: re-solve a
difficulty-balanced coreset of past tasks in parallel, diagnose by
self-consistency, accept harness changes only by pairwise self-preference
over the incumbent. Domains: software engineering, technical, knowledge work.
- **Verdict: STEAL (methodology).** Rekal's corpus *is* the trajectory store
  RHO assumes. Two thefts: (a) the eval discipline — never adopt a change
  (weights, prompts, skills) without head-to-head comparison on replayed past
  queries; this becomes RekalBench's regression mode. (b) roadmap: a
  `rekal reflect`-style loop that proposes weight/skill changes from past
  sessions and accepts only on measured wins (05 §R1).

### 2. LLM-Wiki — Retrieval as Reasoning (arXiv 2605.25480)
Compiles a corpus into bidirectionally-linked wiki pages; agent traverses via
search/read/follow-link tools; an "Error Book" repairs recurring construction
errors. SOTA on HotpotQA/MuSiQue/2Wiki (+2.0–8.1 F1 over HippoRAG 2,
LightRAG, GraphRAG).
- **Verdict: FRAME + STEAL.** Validates *agent-native traversal over
  compiled structure* — but its compilation is expensive and its freshness
  story is the deferred open problem. Rekal's counter-position: same
  traversal ergonomics (search → follow session/file/lineage links) without
  a compile step to go stale. STEAL the Error Book: `rekal-reflect` already
  mines corrections; formalize its output as a durable rules file (05 §R4).

### 3. AdaMem (arXiv 2603.16496)
Working/episodic/persona/graph memory tiers with question-conditioned
routing; write-path protected from read-path. SOTA on LoCoMo (44.65 F1 with
GPT-4.1-mini, +2.89 over LangMem) and PERSONAMEM.
- **Verdict: FRAME.** The tier taxonomy maps cleanly onto Rekal facets
  (working = current session; episodic = past sessions; persona = steering-
  derived rules; graph = file co-occurrence + lineage) — useful vocabulary
  when writing the paper/blog. Its weakness is ours to exploit: tiers are
  *maintained structures* with a consolidation pipeline that must be kept
  healthy; Rekal derives equivalents at query time.

### 4. MRAgent — Memory is Reconstructed (arXiv 2606.06036, ICLR/ICML 2026)
Cue–Tag–Content associative graph + *active reconstruction*: the LLM
iteratively explores/prunes retrieval paths using accumulated evidence.
Formal theorem: active retrieval strictly more powerful than passive
(binary-tree needle construction); reasoning depth cannot be substituted by
retrieval breadth. +23% relative over strongest baseline on LoCoMo multi-hop;
fewer total tokens than Mem0 on LongMemEval.
- **Verdict: SUPPORT (the theory for our skills).** Rekal's skill suite *is*
  an active-reconstruction policy over primitives: search → facets →
  co-occurrence zoom → lineage → windowed drill. The theorem is the citation
  for why `rekal` + skills beats one-shot top-k RAG, and why the CLI exposes
  navigation primitives rather than a single answer endpoint. Benchmark
  implication: include multi-hop tasks (T4) where one-shot retrieval fails.

### 5. EMem (arXiv 2511.17208, UIUC)
Anti-compression baseline: represent history as fine-grained, self-contained
elementary discourse units with normalized entities and **source-turn
attribution**, in a heterogeneous graph. Simple, beats elaborate systems.
- **Verdict: SUPPORT (strongest design validation).** Rekal made the same
  bet: preserve turns, never summarize into the store, attribute every
  snippet to its source turn. Cite EMem wherever anyone asks "why don't you
  distill sessions into summaries at write time?" Answer: write-time
  compression is lossy and unfixable; query-time distillation (skills) is
  free to improve.

### 6. SAG — Query-Time Dynamic Hyperedges (arXiv 2606.15971)
Replaces offline KG construction with query-time SQL joins: chunks become
events + indexing entities in SQL/vector/FTS indexes; hyperedges instantiate
per query via reverse joins. Cheap append-only incremental updates; strong on
MuSiQue.
- **Verdict: SUPPORT (architecture twin).** SAG is the academic argument for
  Rekal's exact storage design (DuckDB = SQL + FTS + vector arrays;
  `file_cooccurrence`/`checkpoint_sessions` = the join tables; append-only
  friendly). STEAL: surface query-time joins in recall output — related
  sessions via shared files as a `related` field (05 §R5).

### 7. EDV — Escaping the Self-Confirmation Trap (arXiv 2606.24428)
Names the trap: wrong-but-self-consistent trajectories mistaken for success,
amplified by memory reuse. Fix: decouple Execute (heterogeneous agents) →
Distill (third party, not self-summarization) → Verify (consensus gate before
admission to memory).
- **Verdict: SUPPORT (the team-memory moat).** Rekal's merged-only gate is a
  *stronger* verifier than EDV's consensus: code review + CI + actually
  landing on main is external ground truth, not model consensus. Positioning
  line: "your shared memory contains only experience that shipped."
  Also a caution for the local index: unmerged/abandoned sessions are
  valuable as *negative* knowledge (dead-ends) but must be labeled as such
  in skills — never presented as endorsed practice (rekal-distill's boundary
  library already frames this correctly).

### 8. AutoMem (arXiv 2607.01224, Stanford)
Metamemory: memory management as a trainable skill; file-system ops promoted
to first-class memory actions; a strong LLM reviews trajectories and revises
the memory *structure* (prompts/schemas/action vocabulary). 1.89–3.74×
progression gains on Crafter/MiniHack/NetHack.
- **Verdict: STEAL (roadmap).** Rekal's embedded skills are its "memory
  structure." AutoMem says: review real usage traces and revise that
  structure automatically. Rekal's corpus contains its *own* usage (sessions
  where the agent called `rekal`) — mine them to improve the skills (05 §R6).

## Retrieval-interface set

### 9. DCI — Direct Corpus Interaction (arXiv 2605.05242)
Agent searches raw corpus with grep/file reads/shell only — no embeddings, no
index. Beats sparse/dense/rerank baselines on several BRIGHT/BEIR sets;
strong on BrowseComp-Plus and multi-hop QA. No offline indexing; adapts to
evolving corpora.
- **Verdict: THREAT (the honest one).** This is the "why not just grep
  ~/.claude/projects?" objection with SOTA numbers behind it. It must be
  RekalBench baseline B1, run fairly. Our counters (to be *measured*, not
  asserted): (a) scale — see RISE below; (b) raw JSONL is adversarial
  material (tool dumps, base64, dedup noise) vs Rekal's parsed/scrubbed
  turns; (c) team memory isn't on your disk to grep — it arrives via sync;
  (d) token cost of grep-then-read at 60k+ turns.

### 10. GrepSeek (arXiv 2605.29307)
Trains agents for shell-based corpus interaction; beats trained agentic
retrieval on 4/7 QA benchmarks, especially multi-hop.
- **Verdict: THREAT (same family).** Notes that *trained* DCI is stronger
  than zero-shot — our grep baseline should use a good prompt, not a straw
  man.

### 11. RISE — Retrieving Interaction Spaces (arXiv 2606.06880)
Unbounded DCI doesn't scale: every broad shell command scans the corpus. At
1M docs DCI falls to 60% accuracy with 33/100 wall-clock failures at
~$1.10/query; RISE (BM25 constructs a bounded per-query workspace, agent
explores it with shell tools) holds 78% at $0.28/query.
- **Verdict: SUPPORT (the killer citation).** Rekal *is* RISE for intent
  history: hybrid search constructs the bounded space (scored sessions),
  `rekal query --session --offset/--limit/--role` is the in-space
  exploration. The positioning sentence: "grep works until your history is
  big enough to matter; Rekal is the bounded interaction space that keeps
  working." RekalBench must include a corpus-scale sweep to reproduce this
  crossover on session data.

### 12. LRAT — Learning to Retrieve from Agent Trajectories (arXiv 2604.04949)
Mines retrieval supervision from agent behavior (browsing actions, unbrowsed
rejections, post-browse reasoning) instead of human labels; improves evidence
recall and task success.
- **Verdict: STEAL (the data flywheel).** After `rekal` returns hits, which
  session the agent actually drills into — and whether it keeps reading — is
  an implicit relevance label. Log locally (opt-in), use to tune weights per
  corpus (05 §R1). This turns every Rekal user into their own training set,
  privately.

## Temporal set

### 13. BeliefShift (arXiv 2603.23848)
Longitudinal benchmark (2,400 annotated multi-session trajectories) for
belief consistency, contradiction detection, evidence-driven revision;
metrics BRA/DCS/CRR/ESI.
- **Verdict: FRAME + roadmap.** Decisions in a repo drift too ("we chose X"
  → later reversed). Rekal has the raw material (timestamps, branches,
  supersession by later sessions touching the same files). Benchmark task T5
  (decision-drift: retrieve the *latest* decision, flag the reversal) and
  roadmap item 05 §R3 come from here.

## Governance / security set

### 14. Memory-security survey (arXiv 2604.16548)
Write-path attacks (AgentPoison, MINJA, MemoryGraft…), contamination
propagation (lateral, vertical, temporal), incomplete forgetting.
- **Verdict: SUPPORT (audit story).** Append-only + provenance means every
  memory is attributable and the write path is a git hook on *your own
  commits* — a tiny attack surface vs. open write APIs. Document as a
  security posture note; add a contamination regression test (05 §R7).

### 15. Governed Memory (arXiv 2603.17787)
Five enterprise failures: silos, governance fragmentation, unstructured
memories unusable downstream, redundant context delivery, silent quality
degradation.
- **Verdict: FRAME (enterprise page, later).** Rekal's answers: git *is* the
  governance plane (branch protection, review, ACLs); merged-gate is quality
  admission; the wire format is structured. Keep for future enterprise
  positioning; no action now.

### 16. State Contamination (arXiv 2605.16746)
Sub-threshold propagation gap: sanitizing *before* summarization works;
sanitizing the completed summary leaves hidden influence.
- **Verdict: SUPPORT (we already do the right order).** Rekal scrubs
  (secrets, paths, UTF-8) *before* any insert, and stores raw-not-summarized
  — precisely the order this paper shows is safe. Cite in the security note.

### 17. Provenance survey (arXiv 2606.04990)
Evidence tracing under conflict/contamination; memories invalidated by later
evidence reused as trustworthy.
- **Verdict: SUPPORT.** Provenance is Rekal's native primitive
  (artifact→commit→session→turn; `rekal-provenance`). The survey supplies the
  vocabulary for why that chain matters.

---

## Summary table

| Paper | Verdict | One-line consequence |
|---|---|---|
| 1 RHO | STEAL | regression-mode benchmarking; self-tuning loop later |
| 2 LLM-Wiki | FRAME/STEAL | traversal ergonomics w/o compile step; Error Book → reflect rules |
| 3 AdaMem | FRAME | tier vocabulary; their maintenance burden is our contrast |
| 4 MRAgent | SUPPORT | theory for skills-as-active-reconstruction; multi-hop task T4 |
| 5 EMem | SUPPORT | validates preserve-don't-compress + turn attribution |
| 6 SAG | SUPPORT | architecture twin; expose query-time joins in output |
| 7 EDV | SUPPORT | merged-gate = external verifier; label dead-ends as negative |
| 8 AutoMem | STEAL | mine own usage traces to improve skills |
| 9 DCI | THREAT | must-beat baseline B1, run fairly |
| 10 GrepSeek | THREAT | make B1 strong (good prompt) |
| 11 RISE | SUPPORT | the scale argument; corpus-size sweep in bench |
| 12 LRAT | STEAL | implicit relevance logging → weight tuning |
| 13 BeliefShift | FRAME | decision-drift task T5; temporal roadmap |
| 14 security survey | SUPPORT | security posture note + contamination test |
| 15 governed memory | FRAME | enterprise page, later |
| 16 state contamination | SUPPORT | scrub-before-store is the proven-safe order |
| 17 provenance survey | SUPPORT | vocabulary for the provenance chain |
