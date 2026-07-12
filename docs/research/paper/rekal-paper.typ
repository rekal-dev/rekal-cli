// Rekal paper — compile with: typst compile rekal-paper.typ
// (or: python3 -c "import typst; typst.compile('rekal-paper.typ', output='rekal-paper.pdf')")
// All empirical values are from the 2026-07-12 corpus run; its aggregate
// manifest is committed under docs/research/runs/ per DATA-RUN.md §6.

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
#show raw.where(block: true): set text(size: 7.4pt)

// ---------- Title block (spans both columns) ----------
#place(top, scope: "parent", float: true, clearance: 18pt)[
  #align(center)[
    #text(size: 17pt, weight: "bold")[
      The Commit Is the Label: Four Problems in Agent Memory\ That Version Control Already Solves
    ]
    #v(6pt)
    #text(size: 10.5pt)[Frank Guo#super[1]]
    #v(2pt)
    #text(size: 9pt)[#super[1]Rekal — #link("https://rekal.dev")[rekal.dev] · #link("mailto:guocongmit@gmail.com")[guocongmit\@gmail.com] · #link("https://github.com/rekal-dev/rekal-cli")[github.com/rekal-dev/rekal-cli]]
  ]
]

// ---------- Abstract ----------
#block(inset: (x: 2pt))[
  *Abstract.* Memory systems for AI coding agents are rebuilding, in
  software, guarantees that version control already provides. An agent's
  memory is context assembly under a token budget — for each question, the
  few thousand recorded tokens that change the answer — and the literature
  assembles it with machinery: tiered stores, memory graphs, and compiled
  wikis, guarded by model-judged admission and evaluated on hand-annotated
  benchmarks. The machinery imports four problems it then cannot
  discharge: *annotation* (no ground truth without human labels),
  *staleness* (compiled structure must be maintained), *self-confirmation*
  (a judge that can admit its own mistakes), and *contamination* (one
  shared pool, one open write path). In the software-engineering setting,
  each is answered by a git primitive every team already runs: the commit
  labels which sessions produced which verified change; rebuild and diff
  make derived structure disposable and its drift reviewable; the merge
  externally verifies what enters shared memory; and code review is the
  sole, audited channel across scope boundaries. We build *Rekal*, a
  local-first, single-binary memory engine, on these four guarantees, and
  derive *RekalBench*, the first benchmark for repo-grounded intent
  recall, whose ground truth is mined — not annotated — from the corpus's
  own commit–session links. On a real working corpus of 1,433 sessions
  and 66k turns, hybrid recall attains 0.187 pooled MRR — against 0.171 for
  a strong lexical index and 0.005 for term-frequency grep over the raw
  transcripts — with the semantic layer's contribution concentrated,
  as predicted, on paraphrased provenance queries; a bounded drill then
  assembles answer-bearing context in roughly 1,500 tokens. The prediction
  was registered before the run (\u{00A7}6). The benchmark harness is
  public, fully local, and runnable by anyone on their own history at zero
  annotation cost.
]
#v(4pt)

= Introduction

Code has a ledger; intent does not. Version control records every line a
team ships, but the reasoning that produced those lines — explored designs,
rejected alternatives, the correction a reviewer shouted mid-session — lives
in AI-assistant transcripts that expire with the terminal window. The cost
compounds as agents do more of the work: an agent that cannot remember what
its own team already tried will confidently re-propose it.

Start from what memory *is* for an agent. The context window is the scarce
resource, so memory is not a store — it is *context assembly under a token
budget*: for each question, deliver the few thousand tokens that change the
answer, out of millions of recorded ones. That framing fixes the
requirements. Recall must be *bounded* — a budget-shaped set the agent can
drill, not an unbounded scan @rise2026. What it returns must be *fresh* —
reflecting the repository as of the last commit, not a compaction of last
month. Every memory must be *lineaged* — dereferenceable to the source
turn and the commit it produced, because an unattributable memory cannot
be trusted or audited. Anything *shared* must be verified by something
other than the model's own judgment. And the whole system must be *cheaply
evaluable* — a memory whose quality cannot be measured without an
annotation team will not be measured at all.

The research community meets these requirements with *machinery*: organize
dialogue history into tiered stores @adamem2026, associative graphs
@mragent2026, or compiled wikis @llmwiki2026; guard the store with
model-judged admission @edv2026; evaluate on hand-annotated conversational
suites @locomo2024. Each piece of machinery then becomes its own research
problem, and we name the four: *staleness* — the compiled structure must
be maintained; *self-confirmation* — the judge can admit its own mistakes;
*annotation* — the evaluation does not scale; *contamination* — the shared
pool becomes an attack surface @memsecurity2026. A recent systematic
evaluation
from the data-management perspective — twelve memory systems, five
workloads, eleven datasets — reaches the fitting verdict: *no single
architecture dominates; effectiveness depends on aligning memory structure
with the workload* @anms2026.

This paper takes that verdict literally. For coding agents, the workload
already has a structure — the repository — and *the four open problems the
machinery addresses are already solved by infrastructure every software
team runs* (@tab-thesis). The contribution is not a new structure beside
git but a memory system built *into* it — plus the benchmark that this
construction makes possible, because a corpus whose labels are commits
needs no annotators.

#figure(
  scope: "parent",
  placement: top,
  table(
    columns: (0.8fr, 1fr, 1.3fr),
    align: (left, left, left),
    table.header([*Open problem*], [*The literature's machinery*], [*The git-native answer (Rekal mechanism)*]),
    [*Annotation.* Where does supervision come from?],
    [Human annotation of dialogue corpora @locomo2024; LLM judges grading themselves],
    [*The commit.* A post-commit hook links sessions to the change they produced — free, objective, abundant (\u{00A7}4.1); RekalBench mines its gold from these links (\u{00A7}5)],
    [*Staleness.* How does derived memory stay current?],
    [Consolidation daemons; compiled wikis and graphs whose maintenance is "future work" @llmwiki2026 @adamem2026],
    [*Rebuild + diff.* Indexes are disposable and rebuilt (\u{00A7}4.2); materialized views regenerate deterministically, so drift arrives as a reviewable `git diff`],
    [*Self-confirmation.* Who verifies what enters shared memory?],
    [Self- or consensus-judged admission @edv2026],
    [*The merge.* Only sessions whose commit landed on the default branch are shared; review + CI + merge are the external verifier (\u{00A7}4.3)],
    [*Contamination.* How does private memory cross a boundary?],
    [One machine-wide pool, implicitly readable and writable everywhere @automem2026 @adamem2026 — the documented contamination surface @memsecurity2026 @statecontam2026],
    [*The review.* Cross-repo memory is index-only and structurally unpushable; its sole egress is an origin-labeled page on a PR (\u{00A7}4.4)],
  ),
  caption: [The thesis. Four problems the agent-memory literature treats as
  open research, and the git primitive that answers each. The paper walks
  this table: \u{00A7}4 gives the mechanisms, \u{00A7}6 registers the
  predictions, \u{00A7}7 reports the measurements.],
) <tab-thesis>

*Contributions.* (1) The design and implementation of Rekal, a local-first,
git-native memory engine for coding agents (single binary; capture, storage,
hybrid recall, and team sync with no server, API, or telemetry), with an
agent-facing skill layer that operationalizes active, multi-step memory
reconstruction @mragent2026. (2) *RekalBench*, the first benchmark for
repo-grounded intent recall, whose ground truth is mined from the corpus's
own commit–session structure rather than annotated. (3) A pre-registered
empirical study (predictions fixed before the run, \u{00A7}6) on a real
working corpus against the baselines that matter in practice — including
the strongest current objection, direct shell interaction over raw
transcripts @dci2026 — with token-cost, drill-strategy, and corpus-scale
analyses. (4) Ablations isolating each recall signal, enabled by query-time
weighting that requires no reindexing.

= A Worked Example

Everything in this paper appears once, concretely, in the following
five-minute story. (Illustrative; the exact JSON shapes are the shipped
ones.)

On Monday, an engineer and an agent rework webhook delivery. Mid-session
the engineer interrupts: *"don't retry on a fixed delay — it stampedes the
downstream on recovery."* The agent switches to exponential backoff with
jitter; the change merges. The post-commit hook captures the session —
turns, tool calls, the interruption tagged with its own role
(`human_steering`) — scrubs secrets, and appends it to the repo's ledger
with a link to the commit SHA. Nobody wrote documentation.

On Friday, a different agent, different machine, same team, picks up a
related task and asks:

```json
$ rekal "should webhook retries use a fixed delay?"
{ "results": [{
    "session_id": "01JNQX8F2K9M",
    "score": 0.87,
    "snippet": "don't retry on a fixed delay - it
       stampedes the downstream on recovery",
    "snippet_turn_index": 41,
    "snippet_role": "human_steering",
    "summary_turn_index": 118,
    "session": { "commit": "9c2f41a",
      "files": ["delivery/retry.go", ...] }
}]}
```

Four properties of the answer carry the paper's argument. The top snippet
is the *steering turn itself* — ranking boosts the moments a human
redirected the agent, because that is where decisions actually get made.
`snippet_role` says a human said this; machine text can never masquerade as
intent. The result is a *pointer structure*, not a payload: `snippet_turn_
index` locates the evidence, `summary_turn_index` points at a
harness-written distillation of the whole session (10–17KB, already paid
for during context compaction — never inlined), and the agent drills
exactly as deep as the question warrants. And the `commit` field is the
provenance anchor: the claim "we ruled this out" dereferences to a change
that survived review and merge. The dead end was re-proposed on Friday and
ruled out in one query — by a colleague's Monday, not by documentation.

The rest of the paper is this example generalized: \u{00A7}3 the engine
that produced it, \u{00A7}4 the four guarantees behind its fields,
\u{00A7}5 how a corpus of such sessions labels itself, \u{00A7}6–7 the
measurements.

= The Engine

// ---------- Figure 1: architecture ----------
#let archbox(x, y, w, h, fill, body) = place(dx: x, dy: y, block(
  width: w, height: h, fill: fill, radius: 4pt, stroke: 0.6pt + luma(90),
  inset: 4pt, align(center + horizon, text(size: 7.6pt, body))))
#let arrow(x1, y1, x2, y2) = {
  place(dx: 0pt, dy: 0pt, line(start: (x1, y1), end: (x2, y2), stroke: 0.7pt + luma(60)))
  let dx = x2 - x1; let dy = y2 - y1
  let len = calc.sqrt(dx.pt() * dx.pt() + dy.pt() * dy.pt())
  let ux = dx.pt() / len; let uy = dy.pt() / len
  place(dx: 0pt, dy: 0pt, polygon(fill: luma(60),
    (x2, y2),
    (x2 - 5pt * ux + 2.2pt * uy, y2 - 5pt * uy - 2.2pt * ux),
    (x2 - 5pt * ux - 2.2pt * uy, y2 - 5pt * uy + 2.2pt * ux)))
}
#let alabel(x, y, body) = place(dx: x, dy: y,
  text(size: 6.8pt, fill: luma(50), style: "italic", body))

#figure(
  block(width: 100%, height: 212pt, {
    // agents row
    archbox(0pt, 0pt, 52pt, 24pt, rgb("#eef2ff"))[Claude\ Code]
    archbox(58pt, 0pt, 52pt, 24pt, rgb("#eef2ff"))[Codex]
    archbox(116pt, 0pt, 52pt, 24pt, rgb("#eef2ff"))[Gemini]
    archbox(174pt, 0pt, 52pt, 24pt, rgb("#eef2ff"))[OpenCode]
    arrow(113pt, 24pt, 113pt, 50pt)
    alabel(6pt, 32pt)[post-commit hook:\ parse · scrub · dedup]
    // data.db
    archbox(30pt, 50pt, 166pt, 32pt, rgb("#fff7ed"))[*`data.db`* — append-only ledger\ raw turns · tool calls · `checkpoints(sha)` · `checkpoint_sessions`]
    // fork arrows
    arrow(78pt, 82pt, 58pt, 108pt)
    alabel(0pt, 88pt)[rebuild]
    arrow(150pt, 82pt, 172pt, 108pt)
    alabel(180pt, 86pt)[merged-\ only gate]
    // index.db
    archbox(0pt, 108pt, 120pt, 40pt, rgb("#ecfdf5"))[*`index.db`* — derived, rebuilt\ BM25 FTS · LSA · neural emb.\ co-occurrence · lineage]
    // orphan branch
    archbox(134pt, 108pt, 92pt, 40pt, rgb("#fdf2f8"))[orphan branch\ `rekal/<email>`\ zstd wire, merged only]
    // recall
    arrow(58pt, 148pt, 58pt, 174pt)
    alabel(66pt, 155pt)[hybrid recall\ (query-time weights)]
    archbox(0pt, 174pt, 120pt, 32pt, rgb("#eff6ff"))[*agent* — skills drive\ search → facets → zoom → drill]
    // sync
    arrow(180pt, 148pt, 180pt, 174pt)
    alabel(134pt, 155pt)[git push\ / sync]
    archbox(134pt, 174pt, 92pt, 32pt, rgb("#f5f5f4"))[teammates\ (their `data.db`)]
  }),
  kind: image,
  caption: [Rekal architecture. One source of truth (append-only `data.db`);
  all derived structure disposable (`index.db`); sharing gated on merge
  verification. No server, no API, no telemetry — git is the only wire.],
) <fig-arch>

Rekal is one binary embedding its database engine, embedding models, and
compression dictionary. A post-commit hook parses the active assistant
session(s) — adapters cover Claude Code, Codex, Gemini, and OpenCode —
deduplicates turns, scrubs secrets and anonymizes paths *before any byte is
stored* (the ordering shown to be the only safe one @statecontam2026), and
appends to `data.db`. Nothing is summarized at write time: following the
preservation result of @emem2025, the unit of storage is the raw turn with
role, order, timestamp, and tool-call structure, and every recall result
carries source-turn attribution. Write-time compression is lossy and
unfixable; query-time distillation improves with every model generation.

One class of distillation is nonetheless *harvested*: when the harness
compacts a long session it writes an LLM summary of the conversation so far
back into the transcript. Rekal captures these as turns with a dedicated
role `summary` — already paid for, cumulative (the latest subsumes the
rest), and never confused with human intent. Rekal generates no summaries
of its own: a commit-time summary would be query-blind, written before any
question exists, whereas the drill is query-aware and paid lazily.

Recall scores sessions by a weighted hybrid of BM25, latent-semantic, and
neural similarity; boosts `human_steering` turns (the example's Monday
interruption) and, by a smaller factor, harvested summaries; down-weights
subagent transcripts; and returns the pointer-structured JSON of \u{00A7}2.
Weights live in local config and apply at *query time* — no reindex, which
is also what makes the single-signal ablations of \u{00A7}7 free. In RISE's
terms @rise2026, search constructs a *bounded interaction space* and the
drill tools explore it; a layer of six shipped skills (base search,
provenance, self-reflection, distillation, exhaustive census, wiki
materialization) is the active-reconstruction policy @mragent2026 that
sequences those primitives.

= Four Guarantees from Git

The engine is ordinary; the guarantees are not. This section walks
@tab-thesis row by row, confronting in each row the machinery it replaces.

== Annotation: the commit is the label

Every checkpoint records which sessions produced which commit
(`checkpoint_sessions`). This single join is the paper's namesake and its
most consequential design choice: it connects a unit of *intent* (the
session) to a verified unit of *code* (the commit) — automatically, at zero
marginal cost, thousands of times per repository-year. Dialogue-memory
research must manufacture this connection by human annotation @locomo2024
or accept LLM judges grading their own homework; here the connection is a
side effect of using version control. Everything downstream leans on it:
provenance answers in recall (\u{00A7}2), the merged-only sharing gate
(\u{00A7}4.3), and the entire benchmark (\u{00A7}5), whose five task
families are all mined from this one link plus git topology.

== Staleness: rebuild and diff

`data.db` is the only source of truth and is append-only. `index.db` holds
everything derived — full-text, embeddings, co-occurrence, lineage, facets
— and is *disposable by construction*: any migration, corruption, or model
upgrade is handled by deletion and rebuild (content-hash-keyed embedding
caching keeps rebuilds incremental). This is the structural answer to the
maintenance problem that compiled-memory systems name and defer
@llmwiki2026 @adamem2026 @sag2026: *the freshness of a structure you can
throw away is not a research problem.*

The same discipline extends to memory the team can browse. A wiki skill
materializes topic pages (`docs/wiki/<topic>.md` plus a graph-mapped index)
from file co-occurrence clusters and the sessions behind them, shipped as
an ordinary pull request. Generation is *deterministic* — topics and edges
sorted, thresholds fixed — so a re-run over unchanged history is an empty
diff, and any non-empty diff *is* structural drift: an edge appearing in
the topic graph is a correlation that did not exist last run; a removed
edge is one that decayed; each transition is reviewed before admission.
Mutating graph stores cannot show this drift to anyone; here `git log` over
the index is a reviewed time series of the project's conceptual structure.
The maintenance problem is not solved — it is *converted into code review*,
a process teams already run. Because pages are a cache of memory, not the
memory, their value is an empirical question: \u{00A7}6 registers the
prediction and \u{00A7}7.4 prices the cache (generation cost from the
ledger's own lineage records, payoff against recall+drill, decay as a
measured rate — the static-notes failure mode made continuous).

== Self-confirmation: the merge is the gate

`rekal push` exports to a per-author git orphan branch only those
checkpoints whose commit is reachable from the default branch (with
patch-equivalence detection for squash merges); everything else waits,
releasing automatically if its branch later lands. The shared tier
therefore admits only experience that survived review, CI, and merge — an
*external* verification gate, in contrast to the self- or consensus-judged
admission proposed to escape the self-confirmation trap @edv2026: when the
same model family executes, summarizes, and admits its own memories,
wrong-but-self-consistent trajectories are stored as successes and
amplified on reuse. The merge gate has a second, subtler yield: unmerged
and abandoned branches remain in the local ledger as labeled *negative*
knowledge — the map of dead ends — and \u{00A7}5's task T3 tests exactly
whether a system can warn "we tried this; it didn't land."

== Contamination: review as the egress channel

Rekal's memory has three scopes with asymmetric permeability. The *repo
ledger* is truth. The *machine-wide index* optionally folds in the
operator's other repos' sessions (explicit opt-in) — index-only,
origin-labeled, and structurally unpushable: an imported session has no
checkpoint in this repo's ledger, so no code path can export it. The *team
wire* admits merged work only. Knowledge crosses a scope boundary
exclusively through a human-visible artifact: sessions reach the team when
their commit merges, and cross-repo experience becomes committed text
through exactly one channel — a wiki page generated in an explicit
cross-repo mode that labels every foreign citation with its origin, shipped
as a PR whose body declares what is crossing. Machine-wide stores in the
literature make the opposite choice @automem2026 @adamem2026: one pool,
implicitly readable and writable everywhere — precisely the open write path
the security analyses identify as the contamination and exfiltration
surface @memsecurity2026 @provtrace2026 @statecontam2026, and controlled
experiments show sanitization only works *before* summarization, which is
why Rekal scrubs before insertion and stores raw. Governance concerns for
enterprise memory @governedmem2026 map onto git's existing controls. The
rule compresses to: *wide reads locally, narrow writes globally — egress is
always a diff someone approved.*

= RekalBench

No benchmark exists for repo-grounded intent recall: conversational-memory
suites (LoCoMo @locomo2024, LongMemEval, PERSONAMEM) evaluate chat-persona
recall, and IR suites (BEIR, BRIGHT) evaluate document QA. RekalBench's
defining property follows from \u{00A7}4.1: *the corpus labels itself.*

Watch one label being born. The Monday session of \u{00A7}2 checkpointed
against commit `9c2f41a`. The miner (plain SQL over the ledger, plus git)
emits, with no human in the loop:

```json
{ "task": "t1",
  "gold": ["01JNQX8F2K9M"],
  "commit": "9c2f41a",
  "source": { "subject": "webhooks: switch retries
     to exponential backoff",
     "files": ["delivery/retry.go", ...] } }
```

An LLM paraphrases the source material into a natural developer question
("how did we end up handling webhook retry timing?"), a 4-gram Jaccard
ceiling (≤ 0.30, one aggressive-paraphrase retry, else the label is
dropped and logged) breaks lexical leakage between query and target, and
the pair joins the query set. The gold is objective — that session
verifiably produced that commit — and the supervision cost is zero. The
same construction yields all five task families (@tab-tasks); the steering
turn of \u{00A7}2 is likewise a T2 gold, and the never-merged branches of
\u{00A7}4.3 are T3 gold *because git says so*.

// ---------- Figure 2: bench pipeline ----------
#figure(
  block(width: 100%, height: 174pt, {
    archbox(0pt, 0pt, 108pt, 30pt, rgb("#fff7ed"))[git history\ commits · branches · merges]
    archbox(120pt, 0pt, 106pt, 30pt, rgb("#fff7ed"))[session ledger\ turns · steering · files · lineage]
    arrow(54pt, 30pt, 90pt, 56pt)
    arrow(173pt, 30pt, 137pt, 56pt)
    archbox(58pt, 56pt, 112pt, 26pt, rgb("#ecfdf5"))[SQL label miner\ T1–T5 gold pairs]
    arrow(113pt, 82pt, 113pt, 106pt)
    alabel(122pt, 88pt)[LLM paraphrase +\ n-gram leakage filter]
    archbox(58pt, 106pt, 112pt, 24pt, rgb("#eff6ff"))[query set (JSONL)\ dev 10% / test 90%]
    arrow(113pt, 130pt, 113pt, 142pt)
    archbox(14pt, 142pt, 200pt, 28pt, rgb("#f5f5f4"))[systems B0–B6 → MRR · Recall\@k ·\ judged accuracy · tokens-to-correct · scale sweep]
  }),
  kind: image,
  caption: [RekalBench pipeline. Labels are mined from structure the tool
  already records; paraphrase plus an n-gram overlap ceiling breaks lexical
  leakage between query and target.],
) <fig-bench>

== Tasks

#figure(
  table(
    columns: (auto, 1fr, 1fr, auto),
    align: (left, left, left, left),
    table.header([*Task*], [*Gold label source*], [*Query generation*], [*n*]),
    [T1 Provenance], [commit → producing session(s), via the `checkpoint_` `sessions` links], [paraphrase of commit message + changed paths (not indexed content)], [152],
    [T2 Decision recall], [`human_steering` turns], [paraphrase of surrounding context; steering turn held out of prompt; 4-gram Jaccard ≤ 0.3], [263],
    [T3 Dead-end awareness], [sessions on never-merged branches], ["have we tried X?" from the branch's cumulative intent], [0#super[†]],
    [T4 Multi-hop synthesis], [session pairs linked by file co-occurrence / lineage], [generator-validated two-session questions], [—],
    [T5 Decision drift], [later session reversing an earlier decision on the same files (hand-confirmed sample)], ["what is our current approach to X?"], [—],
  ),
  caption: [RekalBench task families, all mined from recorded structure. This
  study evaluates T1 and T2 — the families this corpus labels at scale
  (415 pairs). T3–T5 are defined by the same construction and evaluated
  wherever a corpus supplies their structure: #super[†]this corpus is
  effectively single-branch (four branches, all merged), so it yields no
  dead-end labels; T4/T5 (the latter motivated by the belief-revision
  result of @beliefshift2026 — memory must surface the *current* decision)
  await a repo with the requisite topology and run on the same public
  harness.],
) <tab-tasks>

*Label noise.* A commit's linked sessions can include incidental chatter;
because the label is the objective commit–session link rather than a
subjective judgment, this noise lowers measured scores (a hit on chatter
misses the gold turn) rather than inflating them, so the retrieval numbers
are conservative. To keep the query independent of the target, T1 queries
are paraphrased from the commit message and changed paths — content the
index never sees — under a 4-gram Jaccard ceiling that drops leaky pairs.
*Splits.* 10% dev (tuning allowed) / 90% test (35/380 after the leakage
filter), one-shot — the incumbent-versus-candidate discipline of @rho2026.

== Corpus and systems

One operator's real working store, folded per-repo (labels require the
repo's own checkpoint ledger) plus machine-wide for scale sweeps. The
primary repo holds 1,433 sessions, 66,323 turns, and 88,144 tool calls,
with 1,219 steering turns and 432 harvested compaction summaries across
152 commits carrying linked sessions (1,060 commit–session links), spanning
2026-03-22 to 2026-07-10; the machine-wide fold adds 3,618 imported sessions
from 18 other projects for the scale sweeps. No session content leaves the
machine; the paper reports aggregates only.

#figure(
  table(
    columns: (auto, 1fr),
    align: (left, left),
    table.header([*ID*], [*System*]),
    [B1], [Grep-rank over raw transcript JSONL — the non-agentic direct-corpus-interaction proxy @dci2026: sessions ranked by term-frequency hits, no parsing or indexing],
    [B3], [BM25-only (Rekal weights `{1,0,0}`) — query-time ablation, no reindex],
    [B4], [Neural-only (weights `{0,0,1}`) — ablation],
    [B5], [Rekal full hybrid + steering boost + summary boost + subagent down-weight],
  ),
  caption: [Systems under test. B1 is the honest lexical baseline for
  retrieval: it is what "just grep the transcripts" scores when the
  transcripts are raw. The agentic form of direct corpus interaction — a
  model driving `rg`/`jq` over a turn budget @dci2026 @grepseek2026 — is a
  judged-rung comparison the same public harness supports. B3/B4 are
  free query-time ablations of B5.],
) <tab-systems>

*Metrics.* Retrievability is scored by MRR, Recall\@{1, 5, 10}, and
nDCG\@10 with 1,000-resample bootstrap CIs, on the held-out test split.
Beyond retrieval, the same self-labeled query set drives a bounded
context-assembly measure — the tokens a drill ingests to cover the gold
turn's distinctive content, with no LLM in the loop — reported in
\u{00A7}7. Judged answer quality (distinct generate/answer/judge models
against the gold turn) and a corpus-scale sweep are the natural extensions
the public harness enables at zero further labeling cost.

= Prediction and Protocol

The four guarantees of @tab-thesis divide by *how they are verified*.
Self-confirmation and contamination are answered *by construction*: the
merge gate and the egress restriction are code paths, checkable by reading
them — an imported session has no checkpoint, so no export path exists, and
no experiment can add to that assurance. Annotation is answered by the
benchmark's very existence: the corpus labels itself (\u{00A7}5), so the
supervision cost that dialogue-memory work must pay is simply not incurred.
What remains genuinely open — the one thing a reader should demand
evidence for — is *retrievability*: does memory built on these guarantees
actually surface the right prior session better than the alternative of not
building it at all? A system whose recall loses to grep needs no philosophy.

We registered one prediction before running the benchmark, and a
mechanistic expectation about *where* it would hold:

#block(inset: (left: 6pt))[
*Prediction (retrievability).* On the held-out test split, Rekal's hybrid
recall (B5) beats term-frequency grep over the raw transcripts (B1) on
pooled MRR and nDCG\@10 with non-overlapping bootstrap intervals.

*Expectation (signal composition).* The semantic layer earns its keep
specifically on paraphrased provenance queries (T1), where the question
shares little surface form with its target; on decision recall (T2), where
a human's steering words tend to recur, lexical matching already suffices.
]

The protocol follows the incumbent-versus-candidate discipline of @rho2026:
weights may be tuned on the 10% dev split; the 90% test split is scored once.

= Results

#figure(
  scope: "parent",
  placement: top,
  table(
    columns: (auto, auto, auto, auto, auto, auto),
    align: (left, center, center, center, center, center),
    table.header([*System*], [*Pooled MRR* (95% CI)], [*nDCG\@10*], [*T1 MRR*], [*T1 R\@5*], [*T2 MRR*]),
    [B1 grep-rank (raw)], [0.005 [.001,.011]], [0.004], [0.014], [0.015], [0.000],
    [B4 neural-only], [0.057 [.040,.078]], [0.059], [0.130], [0.194], [0.018],
    [B3 BM25-only], [0.171 [.145,.202]], [0.219], [0.248], [0.425], [0.130],
    [B5 Rekal hybrid], [*0.187 [.158,.217]*], [*0.227*], [*0.301*], [*0.537*], [*0.124*],
  ),
  caption: [Retrieval quality on the held-out test split (n=380; pooled MRR
  with bootstrap 95% CIs, other columns point estimates). The prediction
  holds: Rekal's hybrid recall exceeds raw-transcript grep by a factor of
  37 on pooled MRR (0.187 vs 0.005) and 57 on nDCG\@10, with
  non-overlapping intervals — parsing and indexing the ledger, not scanning
  it, is what makes recall work. The composition expectation also holds:
  the hybrid's margin over the strong lexical index (B3) is entirely a
  provenance effect (T1 MRR 0.301 vs 0.248, R\@5 0.537 vs 0.425), while on
  decision recall the two are level (T2 0.124 vs 0.130) — the semantic
  layer adds exactly where paraphrase opens a gap.],
) <tab-rung1>

*Reading the table.* Two facts carry the result. First, the gap between any
Rekal configuration and raw grep is categorical, not marginal: even
BM25-only over parsed, scrubbed turns (0.171) outscores term-frequency grep
over the raw JSONL (0.005) by thirty-fold, because raw transcripts are
adversarial retrieval material — tool dumps, base64 blobs, duplicated
sidechains — and the index is not. This is the paper's structural claim
made measurable: the value is in treating the ledger as parsed memory, not
as text to scan. Second, the hybrid's advantage over strong lexical search
is real but *located* — it lives in provenance queries, exactly where the
registered expectation put it, and this is the alignment-by-construction
premise in miniature: on a single-repo corpus, question and session share
vocabulary, so lexical carries most of the signal and the neural layer pays
off precisely when a paraphrase widens the surface-form gap.

Once recall reaches the gold session, a five-turn window around the matched
turn assembles the answer-bearing context — covering roughly 69% of the
gold turn's distinctive content words — in about 1,500 tokens, two to three
orders of magnitude below dumping the session (10–17KB) or the transcript.
The bounded interaction space @rise2026 is what keeps the token budget
small: recall narrows to a handful of sessions, the drill reads only the
neighborhood that matters.

= Positioning

The design confrontations live inline in \u{00A7}4; what remains is where
Rekal sits in the field's own coordinate systems.

On the four-module anatomy of the data-management evaluation @anms2026 —
representation & storage, extraction, retrieval & routing, maintenance —
Rekal classifies as a multi-paradigm hybrid on the first three:
heterogeneous multi-engine storage (relational + full-text + vector in one
embedded database), *deterministic* extraction (parsing and harvesting;
uniquely among the taxonomy's systems, no LLM sits anywhere in the write
path — the property the contamination analyses require), and multi-stage
hybrid retrieval with agentic routing via the skill layer. On the fourth
module it exits the taxonomy's design space entirely: that survey's
maintenance column enumerates LLM-driven semantic consolidation,
capacity-driven eviction, and timestamp multi-versioning — each a paid,
lossy, or complexity-bearing liability — whereas Rekal's maintenance is
*rebuild, diff, and review*: nothing is consolidated (append-only), nothing
is evicted (raw is cheap), and versioning is git itself. The same
evaluation's no-free-lunch finding — effectiveness depends on aligning
memory structure with the workload — is this paper's premise taken to its
limit: the coding workload's structure is the repository, and Rekal aligns
by construction rather than by tuning. On the *compression axis* —
aggressive consolidation @adamem2026 versus preservation with attribution
@emem2025 — Rekal takes EMem's side at the storage layer (raw turns,
attributed snippets) and harvests, rather than generates, distillation. On
the *retrieval axis* — one-shot lookup versus active reconstruction
@mragent2026 — it takes MRAgent's side at the interaction layer: the skill
playbooks drive iterative search–facet–zoom–drill loops over engine
primitives, and LLM-Wiki's page-traversal result @llmwiki2026 is
recovered, without the compilation liability, as the reviewed wiki of
\u{00A7}4.2. Architecturally the storage is aligned with SAG's query-time
joins over flat indexes @sag2026. Against retrieval-free direct corpus
interaction @dci2026 @grepseek2026 the position matches RISE @rise2026: a
bounded interaction space beats an unbounded scan as the corpus grows, and
our result is the sharp form of that argument at the retrieval layer —
term-frequency scanning of raw transcripts is not competitive with recall
over parsed memory even on one developer's history; mapping the full
accuracy-versus-scale crossover against an *agentic* grep baseline is the
natural next rung, on the same public harness. Self-improvement from traces
@rho2026 @automem2026 @lrat2026 is the flywheel this corpus natively enables
(\u{00A7}9); LoCoMo-style persona memory @locomo2024 is out of scope by
design.

= Limitations and Future Work

Retrievability is a proxy: a session that ranks first is a session the agent
*can* reach, not proof that the answer it then reads is correct. The bounded
result is honest about its scope — it measures whether the right memory is
findable and how few tokens a drill needs to cover it, not end-to-end task
success. The natural next rung is judged answer quality against the gold
turn, with distinct generate/answer/judge models; the query set and the
gold labels already exist, so it adds a judge, not a labeling effort.

A single-operator corpus is a case study until replicated. The mitigation is
structural rather than promissory: because the labels are mined, not
annotated, the entire benchmark is public, fully local, and runnable by any
Rekal user on their own store at zero marginal cost — replication does not
wait on us. The same openness covers the task families this corpus cannot
exercise (dead-ends need unmerged branches; drift needs a reversal history)
and the scale sweep against an agentic-grep baseline.

Finally, the neural layer is the weakest signal here (B4, @tab-rung1). That
is consistent with the paper's thesis — a single repo's question and its
answering session share vocabulary — but it also invites a direct test:
substituting a stronger code-tuned embedding model, served over the HTTP
backend (OpenAI-compatible or Amazon Bedrock), reindexing, and swapping back
for free via the content-hash cache. A material lift to the *hybrid* would
raise the shipped default; a null lift would be further evidence that
alignment, not embedding quality, is the binding constraint @anms2026.

= The Flywheel, and What This Buys

The corpus contains its own improvement signal: which recalled sessions an
agent drills into after a search is an implicit relevance label @lrat2026,
enabling private per-corpus weight tuning accepted only on head-to-head
wins over the incumbent configuration @rho2026; trajectory review can
revise the skill layer itself @automem2026; and the wiki's generation runs
are themselves ledgered sessions, so even the distillation loop is
self-accounting. None of this requires new machinery — it is the same
ledger, read again.

Memory for coding agents does not need another compiled structure to
maintain or another self-judged store to poison itself. It needs a ledger
that already has ground truth. Git supplies the label (the commit), the
freshness mechanism (rebuild and diff), the verifier (the merge), and the
egress channel (the review); Rekal supplies the capture, the disposable
indexes, the bounded recall, and the playbooks. RekalBench turns the same
structure into the first benchmark for repo-grounded intent recall, on
which recall over parsed memory beats scanning raw transcripts by a factor
of thirty-seven — and because its labels are mined, not annotated, the
claim is falsifiable by anyone, locally, on their own history.

#v(4pt)
*Reproducibility.* Engine, skills, benchmark spec, extraction SQL, runbook
(`DATA-RUN.md`), and this paper's source:
#link("https://github.com/rekal-dev/rekal-cli")[github.com/rekal-dev/rekal-cli]
(`docs/research/`). All benchmark data remains on the operator's machine;
published artifacts are aggregates, prompts, and code.

#bibliography("refs.bib", style: "ieee", title: "References")
