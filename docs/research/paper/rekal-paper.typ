// Rekal paper — compile with: typst compile rekal-paper.typ
// (or: python3 -c "import typst; typst.compile('rekal-paper.typ', output='rekal-paper.pdf')")
// Placeholder values are rendered as red ⟨…⟩ via #tbd — replaced when the
// corpus run (docs/research/04-data-plan.md) completes.

#let tbd(body) = text(fill: rgb("#b91c1c"), style: "italic", [⟨#body⟩])

#set page(paper: "us-letter", margin: (x: 54pt, y: 60pt), columns: 2)
#set columns(gutter: 20pt)
#set text(font: "New Computer Modern", size: 9.5pt)
#set par(justify: true, leading: 0.58em)
#set heading(numbering: "1.1")
#show heading: it => block(above: 1.1em, below: 0.65em, it)
#show heading.where(level: 1): set text(size: 11pt)
#show heading.where(level: 2): set text(size: 10pt)
#show link: set text(fill: rgb("#1d4ed8"))
#show figure.caption: set text(size: 8.5pt)
#set table(stroke: 0.4pt + luma(120), inset: 4pt)
#show table: set text(size: 8.3pt)

// ---------- Title block (spans both columns) ----------
#place(top, scope: "parent", float: true, clearance: 18pt)[
  #align(center)[
    #text(size: 17pt, weight: "bold")[
      The Commit Is the Label: Git-Native, Self-Verifying\ Memory for AI Coding Agents
    ]
    #v(6pt)
    #text(size: 10.5pt)[Frank Guo#super[1]]
    #v(2pt)
    #text(size: 9pt)[#super[1]Rekal — #link("https://rekal.dev")[rekal.dev] · #link("mailto:guocongmit@gmail.com")[guocongmit\@gmail.com] · #link("https://github.com/rekal-dev/rekal-cli")[github.com/rekal-dev/rekal-cli]]
    #v(6pt)
    #text(size: 8.5pt, fill: rgb("#b91c1c"), style: "italic")[
      DRAFT v0.1 — architecture and benchmark design are final; all empirical values marked ⟨·⟩ are pending the corpus run of \u{00A7}5.
    ]
  ]
]

// ---------- Abstract ----------
#block(inset: (x: 2pt))[
  *Abstract.* AI coding agents start every session amnesic: the conversations
  in which a team weighed approaches, rejected alternatives, and decided are
  gone the moment the session ends, so agents re-propose what was already
  ruled out and re-litigate what was already settled. Recent memory systems
  answer with compiled knowledge structures — tiered stores, memory graphs,
  wikis — that must themselves be maintained, and with self-judged experience
  admission that is vulnerable to self-confirmation. We present *Rekal*, a
  git-native memory system for coding agents built on three design
  commitments that invert the prevailing recipe: (1)~*preserve, don't
  compress* — the store is an append-only ledger of raw session turns,
  captured by a post-commit hook, with every derived structure (full-text,
  LSA, and neural-embedding indexes) disposable and rebuilt rather than
  maintained; (2)~*the commit is the label* — each checkpoint links sessions
  to the commit they produced, giving free, objective supervision that
  dialogue-memory benchmarks must hand-annotate; and (3)~*the merge is the
  verifier* — only sessions whose code landed on the default branch are
  shared to the team, an external admission gate that self-judged memory
  lacks. On these labels we build *RekalBench*, the first benchmark for
  repo-grounded intent recall: five task families (provenance, decision
  recall, dead-end awareness, multi-hop synthesis, decision drift) mined
  from a real corpus of #tbd[N] sessions / #tbd[N] turns, evaluated against
  a no-memory floor, a strong grep/direct-corpus-interaction baseline,
  static distilled notes, and single-signal ablations. Rekal's hybrid
  recall attains #tbd[MRR] versus #tbd[MRR] for the strongest baseline,
  reaches correct answers at #tbd[k]$times$ fewer context tokens, and holds
  retrieval quality flat as the corpus scales while unbounded grep degrades.
  All data stays on the operator's machine; the harness is public and
  self-serve.
]
#v(4pt)

= Introduction

Code has a ledger; intent does not. Version control records every line a
team ships, but the reasoning that produced those lines — explored designs,
rejected alternatives, the correction a reviewer shouted mid-session — lives
in AI-assistant transcripts that expire with the terminal window. The cost
compounds as agents do more of the work: an agent that cannot remember what
its own team already tried will confidently re-propose it.

The research community has converged on the diagnosis but not the cure.
Long-horizon memory systems organize dialogue history into tiered stores
@adamem2026, associative graphs @mragent2026, or compiled wikis
@llmwiki2026, and report strong gains on conversational benchmarks such as
LoCoMo @locomo2024. Yet three structural problems recur across this
literature. First, *derived structure goes stale*: systems that compile
knowledge offline acknowledge freshness and maintenance of that structure as
an open problem, and the more elaborate the structure, the heavier the
liability @llmwiki2026 @adamem2026 @mragent2026 @sag2026. Second,
*self-judged experience poisons itself*: when the same agent executes,
summarizes, and admits its own memories, wrong-but-self-consistent
trajectories are stored as successes and amplified on reuse — the
self-confirmation trap @edv2026. Third, *evaluation lacks ground truth*:
dialogue corpora have no objective notion of "the memory that mattered," so
benchmarks depend on costly human annotation @locomo2024 @beliefshift2026.

This paper's observation is that the software-engineering setting dissolves
all three problems at once, if the memory system is built *into git* rather
than beside it:

- *Freshness.* Rekal's store is two databases with strictly separated
  roles: `data.db`, an append-only ledger of raw, scrubbed session turns —
  the only source of truth — and `index.db`, every derived structure
  (BM25 full-text, latent-semantic and neural embeddings, co-occurrence and
  lineage tables), *disposable by construction* and rebuilt from the ledger
  at any time. Derived structure that can be thrown away cannot go stale.

- *Supervision.* A post-commit hook checkpoints the active session(s) and
  records which sessions produced which commit. The commit is an objective,
  free, abundant label connecting a unit of intent to a verified unit of
  code — supervision that dialogue memory must manufacture by annotation.

- *Verification.* Rekal shares a session with the team only when its commit
  is reachable from the default branch (merge-commit, rebase, or
  patch-equivalent squash). Code review, CI, and the merge itself act as an
  *external* admission gate on shared memory — a stronger verifier than the
  model-consensus gates proposed to escape the self-confirmation trap
  @edv2026 — while unmerged and abandoned work remains local as explicitly
  *negative* knowledge: the map of dead ends.

*Contributions.* (1) The design and implementation of Rekal, a local-first,
git-native memory engine for coding agents (single binary; capture, storage,
hybrid recall, and team sync with no server, API, or telemetry), together
with an agent-facing skill layer that operationalizes active, multi-step
memory reconstruction @mragent2026 over the engine's primitives.
(2) *RekalBench*, the first benchmark for repo-grounded intent recall, whose
ground truth is mined from the corpus's own commit–session structure rather
than annotated. (3) An empirical study on a real working corpus
(#tbd[N sessions, N turns, N tool calls across N repositories]) against the
baselines that matter in practice — including the strongest current
objection, direct corpus interaction over raw transcripts with shell tools
@dci2026 — with token-cost and corpus-scale analyses. (4) Ablations
isolating each recall signal (lexical, latent-semantic, neural, steering
boost, subagent down-weighting), enabled by query-time weighting that
requires no reindexing.

= Related Work

*Long-horizon agent memory.* AdaMem organizes dialogue into
working/episodic/persona/graph tiers with question-conditioned routing and a
write path protected from reads, reporting state-of-the-art LoCoMo F1 of
44.65 with GPT-4.1-mini @adamem2026. MRAgent represents memory as a
cue–tag–content graph and — centrally — replaces one-shot lookup with
*active reconstruction*, proving a formal separation between active and
passive retrieval and reporting a 23% relative multi-hop gain on LoCoMo with
lower token cost than strong baselines @mragent2026. EMem argues the
opposite of aggressive consolidation: fine-grained, self-contained discourse
units with normalized entities and *source-turn attribution* form a simple
baseline that rivals elaborate systems @emem2025. Rekal takes EMem's side of
the compression axis at the storage layer (raw turns, attributed snippets)
and MRAgent's side of the retrieval axis at the interaction layer (skills
drive iterative search–facet–zoom–drill loops over engine primitives).

*Compiled versus query-time structure.* LLM-Wiki compiles a corpus into
bidirectionally linked pages traversed with search/read/follow tools,
outperforming graph-RAG systems by 2.0–8.1 F1 on multi-hop QA @llmwiki2026,
but inherits a compilation artifact that must be maintained. SAG instead
instantiates relational structure *at query time* by SQL joins over flat
event/entity indexes, making incremental update trivial @sag2026. Rekal's
storage is architecturally aligned with SAG: DuckDB tables (full-text,
embeddings, file co-occurrence, checkpoint links) with per-query weighting
and joins, and no compiled artifact other than content-hash-cached
embeddings.

*Retrieval interfaces for agents.* Direct corpus interaction (DCI) equips an
agent with grep, file reads, and shell over the raw corpus — no index at all
— and beats sparse, dense, and reranking retrieval on several benchmarks
@dci2026, with trained variants stronger still @grepseek2026. This is the
serious objection to any index: transcripts are files; why not grep them?
RISE supplies the answer at scale: unbounded shell interaction degrades
sharply with corpus size — at one million documents DCI accuracy falls to
60% with a third of runs hitting wall-clock failure at roughly \$1.10 per
query, while a *bounded interaction space* constructed by retrieval holds
78% accuracy at \$0.28 @rise2026. Rekal is precisely a bounded-interaction-
space constructor for intent history: hybrid scoring bounds the space to a
ranked set of sessions; windowed, role-filtered drilling explores inside it.
\u{00A7}6 reproduces the crossover on session data.

*Experience admission and memory integrity.* EDV names the
self-confirmation trap and decouples execution, distillation, and
verification, gating memory admission on consensus @edv2026. Surveys of
memory security catalogue write-path poisoning attacks and contamination
propagation across sessions and agents @memsecurity2026 @provtrace2026, and
controlled experiments show sanitization is only effective *before*
summarization, not after @statecontam2026. Rekal's answers are structural:
the write path is the operator's own post-commit hook (no open write API);
content is scrubbed *before* insertion and stored uncompressed; every memory
carries provenance to a commit; and the shared tier admits only
merge-verified experience. Governance concerns for enterprise memory
@governedmem2026 map onto git's existing controls (review, branch
protection, ACLs).

*Self-improvement from traces.* RHO improves an agent harness from unlabeled
past trajectories, accepting changes only by head-to-head self-preference
over the incumbent @rho2026; AutoMem treats memory management itself as a
trainable skill, revising the memory structure from trajectory review
@automem2026; LRAT mines retrieval supervision from agent behavior rather
than human labels @lrat2026. Rekal's corpus natively contains all three
ingredients — trajectories, their outcomes (commits, merges), and the
agent's own recall behavior — and \u{00A7}7 sketches the resulting
data flywheel. BeliefShift motivates our decision-drift task: memories are
beliefs that get revised, and a memory system must surface the *current*
decision @beliefshift2026.

// ---------- Figure 1: architecture ----------
#let archbox(x, y, w, h, fill, body) = place(dx: x, dy: y, block(
  width: w, height: h, fill: fill, radius: 4pt, stroke: 0.6pt + luma(90),
  inset: 4pt, align(center + horizon, text(size: 7.6pt, body))))
#let arrow(x1, y1, x2, y2, label: none, lx: 0pt, ly: 0pt) = {
  place(dx: 0pt, dy: 0pt, line(start: (x1, y1), end: (x2, y2), stroke: 0.7pt + luma(60)))
  // arrowhead
  let dx = x2 - x1; let dy = y2 - y1
  let len = calc.sqrt(dx.pt() * dx.pt() + dy.pt() * dy.pt())
  let ux = dx.pt() / len; let uy = dy.pt() / len
  place(dx: 0pt, dy: 0pt, polygon(fill: luma(60),
    (x2, y2),
    (x2 - 5pt * ux + 2.2pt * uy, y2 - 5pt * uy - 2.2pt * ux),
    (x2 - 5pt * ux - 2.2pt * uy, y2 - 5pt * uy + 2.2pt * ux)))
  if label != none {
    place(dx: (x1 + x2) / 2 + lx, dy: (y1 + y2) / 2 + ly,
      text(size: 6.8pt, fill: luma(50), style: "italic", label))
  }
}

#figure(
  block(width: 100%, height: 208pt, {
    let cw = 228pt // usable column width
    // agents row
    archbox(0pt, 0pt, 52pt, 24pt, rgb("#eef2ff"))[Claude\ Code]
    archbox(58pt, 0pt, 52pt, 24pt, rgb("#eef2ff"))[Codex]
    archbox(116pt, 0pt, 52pt, 24pt, rgb("#eef2ff"))[Gemini]
    archbox(174pt, 0pt, 52pt, 24pt, rgb("#eef2ff"))[OpenCode]
    arrow(113pt, 24pt, 113pt, 44pt, label: [post-commit hook: parse + scrub + dedup], lx: -108pt, ly: -12pt)
    // data.db
    archbox(40pt, 44pt, 146pt, 30pt, rgb("#fff7ed"))[*`data.db`* — append-only ledger\ raw turns · tool calls · `checkpoints(sha)` · `checkpoint_sessions`]
    // fork arrows
    arrow(80pt, 74pt, 60pt, 96pt, label: [rebuild (disposable)], lx: -58pt, ly: -4pt)
    arrow(150pt, 74pt, 172pt, 96pt, label: [merged-only\ gate], lx: 4pt, ly: -8pt)
    // index.db
    archbox(0pt, 96pt, 118pt, 38pt, rgb("#ecfdf5"))[*`index.db`* — derived, rebuilt\ BM25 FTS · LSA · neural emb.\ file co-occurrence · lineage]
    // orphan branch
    archbox(132pt, 96pt, 94pt, 38pt, rgb("#fdf2f8"))[orphan branch\ `rekal/<email>`\ zstd wire, merged work only]
    // recall
    arrow(59pt, 134pt, 59pt, 156pt, label: [hybrid recall (query-time weights)], lx: -50pt, ly: -12pt)
    archbox(0pt, 156pt, 118pt, 30pt, rgb("#eff6ff"))[*agent* — skills drive\ search → facets → zoom → drill]
    // sync
    arrow(179pt, 134pt, 179pt, 156pt, label: [git push / sync], lx: 4pt, ly: -10pt)
    archbox(132pt, 156pt, 94pt, 30pt, rgb("#f5f5f4"))[teammates\ (their `data.db`)]
  }),
  kind: image,
  caption: [Rekal architecture. One source of truth (append-only `data.db`);
  all derived structure disposable (`index.db`); sharing gated on merge
  verification. No server, no API, no telemetry — git is the only wire.],
) <fig-arch>

= Rekal: Design

== Capture: preserve, don't compress

A post-commit hook parses the active assistant session(s) — adapters cover
Claude Code, Codex, Gemini, and OpenCode — deduplicates turns, scrubs
secrets and anonymizes paths *before any byte is stored*
(the ordering shown to be the only safe one @statecontam2026), and appends
to `data.db`. Nothing is summarized at write time: following the
preservation result of @emem2025, the unit of storage is the raw turn with
role, order, timestamp, and tool-call structure, and every recall result
carries `snippet_turn_index` attribution back to its source turn. Write-time
compression is lossy and unfixable; query-time distillation improves with
every model generation.

One class of distillation is nonetheless *harvested*: when the harness
compacts a long session it writes an LLM summary of the conversation so far
back into the transcript (10–17KB enumerating files touched, decisions,
errors and fixes). Rekal captures these as turns with a dedicated role
`summary` — already paid for, cumulative (the latest subsumes the rest),
and never confused with human intent. Rekal generates no summaries of its
own: a commit-time summary would be query-blind, written before any
question exists, whereas the drill is query-aware and paid lazily. Rows
stored before the role existed are reclassified in the derived views by a
stable content fingerprint, scoped to the originating harness — the
append-only ledger is never rewritten.

== Storage: one ledger, disposable indexes

`data.db` is the only source of truth and is append-only. `index.db` holds
everything derived: BM25 full-text over turns, latent-semantic (LSA) and
neural (nomic-class) session embeddings, file-touch and co-occurrence
tables, session lineage (parent/subagent), and facets. Any migration,
corruption, or model upgrade is handled by deletion and rebuild;
content-hash-keyed embedding caching makes rebuilds incremental. This is the
structural answer to the maintenance problem that compiled-memory systems
name and defer @llmwiki2026 @adamem2026 @sag2026: *the freshness of a
structure you can throw away is not a research problem.*

== Recall: a bounded interaction space

`rekal "<question>"` scores sessions by a weighted hybrid of BM25, LSA, and
neural similarity, boosts turns where a human redirected the agent
(`human_steering` — the moments decisions actually got made), boosts
harvested compaction summaries by a smaller factor (dense anchors, but
machine text must not outrank human intent at equal relevance),
down-weights subagent chatter, and returns scored JSON with per-session
snippets. Weights live in local config and apply at *query time* — changing
them requires no reindex, which is also what makes single-signal ablations
free (\u{00A7}5). Each result additionally carries `summary_turn_index` — a
pointer to the session's latest compaction summary, never the payload
itself: inlining 10–17KB per result would spend context before any question
justified it. The agent then drills *inside* the bounded set: the pointed-at
summary as the cheapest whole-session overview, windowed turn ranges, role
filters, tool-call and file views, full transcript only as a last resort.
In RISE's terms @rise2026, search constructs the interaction space and the
drill tools explore it; the skill layer (six shipped playbooks: base
search, provenance, self-reflection, four-library distillation, exhaustive
census, and wiki materialization — committed topic pages whose admission
gate is code review, not a judge model) is the active-reconstruction policy
@mragent2026 that sequences those primitives. The wiki playbook additionally
exploits the harness's workflow decomposition: topic clusters are
independent and one cluster's sessions fit one context, so generation fans
out as one subagent per topic whose transcript lands back in the ledger
with lineage (`parent_session_id`, `workflow_name`) — the cost of
distillation is measurable *from the store it distilled*, and recall's
subagent down-weight keeps the meta-work from crowding out what it
recorded.

== Sharing: the merge is the verifier

`rekal push` exports to a per-author git orphan branch only those
checkpoints whose commit is reachable from the default branch, with
patch-equivalence detection for squash merges; everything else waits,
releasing automatically if its branch later lands. The shared tier therefore
admits only experience that survived review, CI, and merge — an external
verification gate, in contrast to self- or consensus-judged admission
@edv2026. Unmerged and abandoned branches remain in the local ledger as
labeled *negative* knowledge: RekalBench task T3 tests exactly whether a
system can warn "we tried this; it didn't land."

== Scopes: wide reads locally, narrow writes globally

Rekal's memory has three scopes with asymmetric permeability. The *repo
ledger* is truth. The *machine-wide index* optionally folds in the
operator's other repos' sessions (explicit opt-in) — index-only,
origin-labeled, and structurally unpushable: an imported session has no
checkpoint in this repo's ledger, so no code path can ever export it. The
*team wire* admits merged work only. Knowledge crosses a scope boundary
exclusively through a human-visible artifact: sessions reach the team when
their commit merges, and cross-repo experience becomes committed text in a
repo through exactly one channel — a wiki page, generated in an explicit
cross-repo mode that labels every foreign citation with its origin, shipped
as a PR whose body declares what is crossing. Machine-wide memory stores in
the literature @automem2026 @adamem2026 make the opposite choice: one pool,
implicitly readable and writable everywhere — precisely the open write path
the security analyses identify as the contamination and exfiltration
surface @memsecurity2026 @statecontam2026. Rekal's rule compresses to:
*wide reads locally, narrow writes globally — egress is always a diff
someone approved.*

= RekalBench

No benchmark exists for repo-grounded intent recall: conversational-memory
suites (LoCoMo @locomo2024, LongMemEval, PERSONAMEM) evaluate chat-persona
recall, and IR suites (BEIR, BRIGHT) evaluate document QA. RekalBench's
defining property is that its ground truth is *mined, not annotated*: the
corpus's own commit–session structure supplies labels.

// ---------- Figure 2: bench pipeline ----------
#figure(
  block(width: 100%, height: 166pt, {
    archbox(0pt, 0pt, 108pt, 30pt, rgb("#fff7ed"))[git history\ commits · branches · merges]
    archbox(120pt, 0pt, 106pt, 30pt, rgb("#fff7ed"))[session ledger\ turns · steering · files · lineage]
    arrow(54pt, 30pt, 90pt, 52pt)
    arrow(173pt, 30pt, 137pt, 52pt)
    archbox(58pt, 52pt, 112pt, 26pt, rgb("#ecfdf5"))[SQL label miner\ T1–T5 gold pairs]
    arrow(113pt, 78pt, 113pt, 96pt, label: [LLM paraphrase + n-gram leakage filter], lx: -104pt, ly: -11pt)
    archbox(58pt, 96pt, 112pt, 24pt, rgb("#eff6ff"))[query set (JSONL)\ dev 10% / test 90%]
    arrow(113pt, 120pt, 113pt, 130pt)
    archbox(24pt, 130pt, 180pt, 26pt, rgb("#f5f5f4"))[systems B0–B6 → MRR · Recall\@k ·\ judged accuracy · tokens-to-correct · scale sweep]
  }),
  kind: image,
  caption: [RekalBench pipeline. Labels are mined from structure the tool
  already records; paraphrase plus an n-gram overlap filter breaks lexical
  leakage between query and target.],
) <fig-bench>

== Tasks

#figure(
  table(
    columns: (auto, 1fr, 1fr, auto),
    align: (left, left, left, left),
    table.header([*Task*], [*Gold label source*], [*Query generation*], [*n*]),
    [T1 Provenance], [commit → producing session(s), via the `checkpoint_` `sessions` links], [paraphrase of commit message + changed paths (not indexed content)], [#tbd[≈500]],
    [T2 Decision recall], [`human_steering` turns], [paraphrase of surrounding context; steering turn held out of prompt; 4-gram Jaccard ≤ 0.3], [#tbd[≈300]],
    [T3 Dead-end awareness], [sessions on never-merged branches], ["have we tried X?" from the branch's cumulative intent], [#tbd[≈50]],
    [T4 Multi-hop synthesis], [session pairs linked by file co-occurrence / lineage], [generator-validated two-session questions], [#tbd[≈100]],
    [T5 Decision drift], [later session reversing an earlier decision on the same files (hand-confirmed sample)], ["what is our current approach to X?"], [#tbd[≈30]],
  ),
  caption: [RekalBench task families. All labels derive from recorded
  structure; only T5 needs light manual confirmation.],
) <tab-tasks>

*Leakage controls.* Commit messages are not part of indexed session text, so
T1 queries are lexically de-correlated by construction; all tasks add LLM
paraphrase with an n-gram overlap ceiling, and we report the residual
overlap distribution. *Label noise.* A commit's linked sessions can include
incidental chatter; we hand-audit 50 T1 pairs and report label precision
(#tbd[value]). *Splits.* 10% dev (tuning allowed) / 90% test, one-shot —
following the incumbent-versus-candidate discipline of @rho2026.

= Experimental Setup

== Corpus

One operator's real working store, folded per-repo (labels require the
repo's own checkpoint ledger) plus machine-wide for scale sweeps. The
primary repo alone holds approximately 500 sessions, 63k turns, and 48k
tool calls (preliminary counts; final corpus card: #tbd[repos, sessions,
turns, tool calls, steering turns, linked commits, date range]). No session
content leaves the machine; the paper reports aggregates only.

== Systems

#figure(
  table(
    columns: (auto, 1fr),
    align: (left, left),
    table.header([*ID*], [*System*]),
    [B0], [No memory — question (plus repo code where the task allows) only],
    [B1], [Grep/DCI @dci2026 — same agent model with `rg`/`jq`/`sed` over raw transcript JSONL; GrepSeek-informed prompt @grepseek2026; equal turn budget],
    [B2], [Static notes — one-time LLM-distilled `MEMORY.md` (≤8k tokens), the folk practice],
    [B3], [BM25-only (Rekal weights `{1,0,0}`) — query-time ablation, no reindex],
    [B4], [Neural-only (weights `{0,0,1}`) — ablation],
    [B5], [Rekal full hybrid + steering boost + subagent down-weight],
    [B6], [Rekal + skills — B5 driven by the shipped playbooks (rungs 2–4)],
  ),
  caption: [Systems under test.],
) <tab-systems>

== Metrics

Rung 1 (retrievability): MRR, Recall\@{1, 5, 10}, nDCG\@10 with bootstrap
CIs. Rung 2 (answer quality): LLM-judged correctness against the gold turn
(distinct generate/answer/judge models; 50-sample human agreement check).
Rung 3 (efficiency): context tokens loaded until first correct answer,
wall-clock, and dollar cost — the axis RISE and MRAgent make primary
@rise2026 @mragent2026. Rung 3 additionally includes a *judge-free
drill-strategy proxy*: on queries where recall places the gold session in
the top results, we compare the two drills an agent can make — a raw
turn window around the matched turn versus the single turn
`summary_turn_index` points at — on tokens ingested and gold-term coverage
(fraction of the label's distinctive content words present in the drilled
text). It measures context-assembly cost, not answer quality, and runs
with no LLM in the loop.

*The L3 gate (wiki experiment).* Materializing a browsing layer — committed
topic pages generated by the wiki playbook — is a cache of memory, and
whether the cache is worth its cost is an empirical question, not an
architectural one. We measure both sides. *Generation cost*, from the
ledger itself: the workflow's subagent transcripts carry lineage, so
turns/tool-calls per topic page are a SQL query over `session_facets`, and
the tokens the run ingested are its harness accounting. *Cache payoff*, on
broad queries (T1/T3-style): answering from the materialized page versus
recall+drill, on tokens and gold-term coverage — the same proxy columns as
the drill table. The static-notes baseline (B2) predicts the failure mode:
pages age. We therefore also report the regeneration diff rate (pages
invalidated per week of new sessions) as the cache's maintenance price.
The diff is itself the observable: generation is deterministic (stable
topic and edge ordering), so a re-run over unchanged history is an empty
diff, and any non-empty diff *is* structural drift — an edge appearing in
the index's topic graph is a correlation that did not exist last run, a
removed edge is one that decayed, each transition reviewed before it is
admitted. Mutating graph stores @adamem2026 @sag2026 cannot show this
drift to anyone; here `git log` over the index is a reviewed time series
of the project's conceptual structure, and the maintenance problem is
converted into code review. Finally, *cross-repo contribution*: pages
generated twice, own-repo evidence only versus explicit cross-repo mode
(\u{00A7}3.4's reviewed-egress channel), report the fraction of pages with
foreign citations and the broad-query coverage delta the foreign evidence
buys — the value of machine-wide memory, measured at the only gate through
which it can become shared. Scale sweep: metrics and B1 latency at 10/25/50/100%
date-cut subsets. Freshness: recall bucketed by target-session age; index
rebuild wall-clock versus corpus size. Rung 4 (agent-in-the-loop): 10–20
real tasks, A/B on human-steering count, re-proposal of known dead ends, and
time-to-done.

= Results

#text(fill: rgb("#b91c1c"), style: "italic")[All values in this section are
placeholders pending the corpus run; tables define the exact shape of the
report.]

== Retrievability (rung 1)

#figure(
  table(
    columns: (auto, auto, auto, auto, auto, auto),
    align: (left, center, center, center, center, center),
    table.header([*System*], [*T1\ MRR*], [*T1\ R\@5*], [*T2\ MRR*], [*T3\ R\@5*], [*T4\ b\@10*]),
    [B1 grep/DCI], tbd[·], tbd[·], tbd[·], tbd[·], tbd[·],
    [B2 static notes], [—], [—], [—], [—], [—],
    [B3 BM25-only], tbd[·], tbd[·], tbd[·], tbd[·], tbd[·],
    [B4 neural-only], tbd[·], tbd[·], tbd[·], tbd[·], tbd[·],
    [B5 Rekal hybrid], tbd[*·*], tbd[*·*], tbd[*·*], tbd[*·*], tbd[*·*],
  ),
  caption: [Retrieval quality by task (test split, bootstrap 95% CIs).
  B2 has no per-query retrieval and is evaluated at rungs 2–3 only.],
) <tab-rung1>

Planned analyses: per-signal contribution (B3/B4 vs B5), steering-boost
on/off delta on T2, and label-precision-adjusted upper bounds.

== Answer quality and token cost (rungs 2–3)

#figure(
  table(
    columns: (auto, auto, auto, auto),
    align: (left, center, center, center),
    table.header([*System*], [*Judged acc.*], [*Tokens → correct*], [*\$ / query*]),
    [B0 no memory], tbd[·], [—], [—],
    [B1 grep/DCI], tbd[·], tbd[·], tbd[·],
    [B2 static notes], tbd[·], tbd[·], tbd[·],
    [B5 Rekal], tbd[·], tbd[·], tbd[·],
    [B6 Rekal + skills], tbd[*·*], tbd[*·*], tbd[*·*],
  ),
  caption: [Answer quality and efficiency on a 200-query stratified subset.
  The headline claim has the shape: equal-or-better accuracy at
  #tbd[k]$times$ fewer tokens.],
) <tab-rung23>

#figure(
  table(
    columns: (auto, auto, auto, auto),
    align: (left, center, center, center),
    table.header([*Drill strategy*], [*Tokens*], [*Coverage*], [*Cov. / 1k tok*]),
    [window (5-turn, at match)], tbd[·], tbd[·], tbd[·],
    [summary-first (pointer)], tbd[·], tbd[·], tbd[·],
  ),
  caption: [Judge-free drill-strategy proxy (`run_rung3.py`), paired on
  queries where recall reaches the gold session and a compaction summary
  exists. Hypothesis: the trade is question-shaped — summary-first buys
  broad coverage at a fixed 10–17KB price and should win "what happened
  here" queries (T1/T3); the window should win pointed lookups (T2). The
  corpus run decides.],
) <tab-drill>

== The L3 gate: is a materialized browsing layer worth its cost?

#figure(
  table(
    columns: (auto, auto, auto),
    align: (left, center, center),
    table.header([*Measure*], [*Value*], [*Source*]),
    [pages generated / sessions cited per page], tbd[·], [wiki-run PR],
    [generation: turns + tool calls per page], tbd[·], [ledger (`workflow_name` lineage)],
    [generation: tokens ingested per page], tbd[·], [harness accounting],
    [broad-query answering: page vs recall+drill, tokens], tbd[· vs ·], [proxy, drill-table columns],
    [broad-query answering: page vs recall+drill, coverage], tbd[· vs ·], [proxy, drill-table columns],
    [regeneration diff rate (pages invalidated / week)], tbd[·], [watermark vs new sessions],
    [cross-repo mode: pages with foreign citations / coverage delta], tbd[· / ·], [origin labels; A/B on generation mode],
  ),
  caption: [The wiki experiment. A committed topic-page layer is a *cache of
  memory*: generation cost is measured from the ledger's own lineage records
  (the memory system accounts for its own distillation), payoff on broad
  queries against recall+drill, and the maintenance price as the rate at
  which new sessions invalidate pages — the static-notes failure mode (B2),
  made continuous and measurable. A cached in-index L3 layer is built only
  if this table says so.],
) <tab-wiki>

== Scale and freshness (the RISE crossover, C4)

We re-run rung 1 at date-cut corpus subsets. The hypothesis, transplanted
from @rise2026: B1's accuracy and latency degrade with corpus size (broad
shell scans over #tbd[N]k turns), while Rekal's bounded space holds flat;
index rebuild remains #tbd[seconds–minutes] and recall shows no decay with
target-session age. Deliverables: accuracy-versus-corpus-size and
tokens-versus-accuracy curves (Fig.~#tbd[3], #tbd[4]).

== Agent-in-the-loop (rung 4)

#tbd[10–20 tasks; steering interventions, dead-end re-proposals,
time-to-done, with and without Rekal] — run last, only if rungs 1–3 hold.

= Discussion and Limitations

*What the numbers can and cannot say.* Rung 1 is a retrievability proxy — a
hit is not a useful answer; that is why rungs 2–4 exist. Self-labels are
noisy (audited, reported). A single-operator corpus is a case study until
replicated: the harness is public, fully local, and runnable by any Rekal
user on their own store, which is the honest path to multi-corpus evidence.
LoCoMo-style persona memory is out of scope by design.

*The grep objection, taken seriously.* DCI is strong @dci2026 and our B1 is
deliberately not a straw man (published prompt, equal model and budget). Our
position is conditional, matching RISE @rise2026: below some corpus size,
grep is fine — and Rekal still adds parsing, scrubbing, provenance, and team
sync; above it, a bounded interaction space is the difference between
answers and wall-clock failures. The crossover point on session data is an
empirical output of this work, not an assumption.

*The flywheel (future work).* The corpus contains its own improvement
signal: which recalled sessions an agent drills into after a search is an
implicit relevance label @lrat2026, enabling private per-corpus weight
tuning, accepted only on head-to-head wins over the incumbent configuration
@rho2026; trajectory review can revise the skill layer itself @automem2026.
Temporal belief maintenance (surfacing the *latest* decision, flagging
reversals @beliefshift2026) is a query-time view away (T5 measures it).

= Conclusion

Memory for coding agents does not need another compiled structure to
maintain or another self-judged experience store to poison itself. It needs
a ledger that already has ground truth. Git supplies the label (the commit),
the verifier (the merge), and the transport (the repo); Rekal supplies the
capture, the disposable indexes, the bounded recall, and the agent-facing
playbooks. RekalBench turns that same structure into the first benchmark
for repo-grounded intent recall — and every claim in this paper is
falsifiable by running it, locally, on your own history.

#v(4pt)
*Reproducibility.* Engine, skills, benchmark spec, extraction SQL, and this
paper's source: #link("https://github.com/rekal-dev/rekal-cli")[github.com/rekal-dev/rekal-cli]
(`docs/research/`). All benchmark data remains on the operator's machine;
published artifacts are aggregates, prompts, and code.

#bibliography("refs.bib", style: "ieee", title: "References")
