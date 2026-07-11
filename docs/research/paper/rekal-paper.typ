// Rekal paper — compile with: typst compile rekal-paper.typ
// (or: python3 -c "import typst; typst.compile('rekal-paper.typ', output='rekal-paper.pdf')")
// Placeholder values are rendered as red ⟨…⟩ via #tbd — replaced when the
// corpus run (docs/research/paper/DATA-RUN.md) completes. Results tables are
// keyed to the pre-registered predictions P1–P8 (§6).

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
    #v(6pt)
    #text(size: 8.5pt, fill: rgb("#b91c1c"), style: "italic")[
      DRAFT v0.2 — architecture, benchmark design, and predictions (\u{00A7}6) are final; all empirical values marked ⟨·⟩ are pending the corpus run of \u{00A7}5.
    ]
  ]
]

// ---------- Abstract ----------
#block(inset: (x: 2pt))[
  *Abstract.* Memory systems for AI coding agents are rebuilding, in
  software, guarantees that version control already provides. The
  literature answers agent amnesia with compiled knowledge structures —
  tiered stores, memory graphs, wikis — that must themselves be maintained;
  with self- or consensus-judged admission that is vulnerable to
  self-confirmation; with hand-annotated benchmarks because dialogue has no
  ground truth; and with machine-wide memory pools whose open write path is
  the documented contamination surface. We observe that in the
  software-engineering setting, each of these four open problems is already
  solved by a git primitive every team runs: the *commit* labels which
  sessions produced which verified change (free supervision); *rebuild and
  diff* replace maintenance (derived structure that can be thrown away
  cannot go stale, and structure that regenerates deterministically turns
  drift into a reviewable diff); the *merge* is an external verifier for
  what enters shared memory; and *code review* is the sole, auditable
  channel by which private cross-repo memory can cross a scope boundary. We
  present *Rekal*, a local-first, single-binary memory engine built on
  these four guarantees, and *RekalBench*, the first benchmark for
  repo-grounded intent recall, whose ground truth is mined — not annotated
  — from the corpus's own commit–session structure. On a real working
  corpus of #tbd[N] sessions and #tbd[N] turns, Rekal's hybrid recall
  attains #tbd[MRR] versus #tbd[MRR] for the strongest baseline (direct
  grep over raw transcripts), reaches correct answers at #tbd[k]$times$
  fewer context tokens, and holds retrieval quality flat as the corpus
  scales while unbounded grep degrades. All predictions were registered
  before the run (\u{00A7}6); the harness is public, fully local, and
  runnable by any user on their own history.
]
#v(4pt)

= Introduction

Code has a ledger; intent does not. Version control records every line a
team ships, but the reasoning that produced those lines — explored designs,
rejected alternatives, the correction a reviewer shouted mid-session — lives
in AI-assistant transcripts that expire with the terminal window. The cost
compounds as agents do more of the work: an agent that cannot remember what
its own team already tried will confidently re-propose it.

The research community has converged on the diagnosis, and its cure is
*machinery*: organize dialogue history into tiered stores @adamem2026,
associative graphs @mragent2026, or compiled wikis @llmwiki2026; guard the
store with model-judged admission @edv2026; evaluate on hand-annotated
conversational suites @locomo2024. Each piece of machinery then becomes its
own research problem — the compiled structure must be maintained, the judge
can confirm its own mistakes, the annotation does not scale, and the shared
pool becomes an attack surface @memsecurity2026.

This paper's claim is that for coding agents, the machinery is unnecessary:
*the four open problems it addresses are already solved by infrastructure
every software team runs* (@tab-thesis). The contribution is not a new
structure beside git but a memory system built *into* it — plus the
benchmark that this construction makes possible, because a corpus whose
labels are commits needs no annotators.

#figure(
  scope: "parent",
  placement: top,
  table(
    columns: (0.8fr, 1fr, 1.3fr),
    align: (left, left, left),
    table.header([*Open problem*], [*The literature's machinery*], [*The git-native answer (Rekal mechanism)*]),
    [Supervision — where do labels come from?],
    [Human annotation of dialogue corpora @locomo2024; LLM judges grading themselves],
    [*The commit.* A post-commit hook links sessions to the change they produced — free, objective, abundant (\u{00A7}4.1); RekalBench mines its gold from these links (\u{00A7}5)],
    [Freshness — how does derived memory stay current?],
    [Consolidation daemons; compiled wikis and graphs whose maintenance is "future work" @llmwiki2026 @adamem2026],
    [*Rebuild + diff.* Indexes are disposable and rebuilt (\u{00A7}4.2); materialized views regenerate deterministically, so drift arrives as a reviewable `git diff`],
    [Verification — who admits experience to shared memory?],
    [Self- or consensus-judged admission @edv2026],
    [*The merge.* Only sessions whose commit landed on the default branch are shared; review + CI + merge are the external verifier (\u{00A7}4.3)],
    [Scope — how does private memory cross a boundary?],
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

== Supervision: the commit is the label

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

== Freshness: rebuild and diff

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

== Verification: the merge is the gate

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

== Scope: review as the egress channel

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
    [T1 Provenance], [commit → producing session(s), via the `checkpoint_` `sessions` links], [paraphrase of commit message + changed paths (not indexed content)], [#tbd[≈500]],
    [T2 Decision recall], [`human_steering` turns], [paraphrase of surrounding context; steering turn held out of prompt; 4-gram Jaccard ≤ 0.3], [#tbd[≈300]],
    [T3 Dead-end awareness], [sessions on never-merged branches], ["have we tried X?" from the branch's cumulative intent], [#tbd[≈50]],
    [T4 Multi-hop synthesis], [session pairs linked by file co-occurrence / lineage], [generator-validated two-session questions], [#tbd[≈100]],
    [T5 Decision drift], [later session reversing an earlier decision on the same files (hand-confirmed sample)], ["what is our current approach to X?"], [#tbd[≈30]],
  ),
  caption: [RekalBench task families. All labels derive from recorded
  structure; only T5 needs light manual confirmation. T5 is motivated by
  the belief-revision result of @beliefshift2026: memory must surface the
  *current* decision.],
) <tab-tasks>

*Label noise.* A commit's linked sessions can include incidental chatter;
we hand-audit 50 T1 pairs and report label precision (P8). *Splits.* 10%
dev (tuning allowed) / 90% test, one-shot — the incumbent-versus-candidate
discipline of @rho2026.

== Corpus and systems

One operator's real working store, folded per-repo (labels require the
repo's own checkpoint ledger) plus machine-wide for scale sweeps. The
primary repo alone holds approximately 500 sessions, 63k turns, and 48k
tool calls (final corpus card: #tbd[repos, sessions, turns, tool calls,
steering turns, linked commits, date range]). No session content leaves the
machine; the paper reports aggregates only.

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
    [B5], [Rekal full hybrid + steering boost + summary boost + subagent down-weight],
    [B6], [Rekal + skills — B5 driven by the shipped playbooks (rungs 2–4)],
  ),
  caption: [Systems under test. B1 is not a straw man: direct corpus
  interaction beats sparse, dense, and reranking retrieval on several
  published benchmarks @dci2026 @grepseek2026 — it is the strongest current
  objection to any index.],
) <tab-systems>

*Metrics.* Rung 1 (retrievability): MRR, Recall\@{1, 5, 10}, nDCG\@10 with
bootstrap CIs. Rung 2 (answer quality): LLM-judged correctness against the
gold turn (distinct generate/answer/judge models; 50-sample human agreement
check). Rung 3 (efficiency): context tokens to first correct answer,
wall-clock, dollar cost — the axis RISE and MRAgent make primary @rise2026
@mragent2026 — plus a *judge-free drill-strategy proxy*: on queries where
recall reaches the gold session, the raw window around the matched turn
versus the single turn `summary_turn_index` points at, compared on tokens
ingested and gold-term coverage. It measures context-assembly cost, not
answer quality, and runs with no LLM in the loop. The *L3 gate* (wiki
experiment) prices the browsing cache of \u{00A7}4.2: generation cost from
the ledger's own workflow lineage, payoff against recall+drill on broad
queries, decay as regeneration diff rate, and a cross-repo A/B (own-repo
evidence versus reviewed-egress cross-repo mode). Scale sweep: metrics and
B1 latency at 10/25/50/100% date-cut subsets. Freshness: recall bucketed by
target-session age; rebuild wall-clock versus corpus size. Rung 4
(agent-in-the-loop): 10–20 real tasks, A/B on steering count, dead-end
re-proposals, time-to-done.

= Predictions (registered before the run)

The four guarantees of @tab-thesis divide by *how they are verified*.
Problems 3 and 4 are answered by construction: the merge gate and the
egress restriction are code paths, checkable by reading them — an imported
session has no checkpoint, so no export path exists; no experiment can add
to that. What experiments test is each guarantee's *yield*: whether the
commit-link supervision produces clean labels (P8 — problem 1), whether
the freshness discipline is affordable and the browsing cache earns its
keep (P6, P7 — problem 2), whether the negative knowledge the merge gate
preserves is actually recallable (T3 inside P1 — problem 3), and whether
reviewed cross-repo egress buys measurable coverage (P7's A/B — problem
4). P1–P5 are the table stakes beneath all four: a memory whose recall
loses to grep needs no philosophy.

Stated falsifiably, before any table is filled; \u{00A7}7's tables are
keyed to them. Where a prediction fails, the failure is reported, not
reframed.

+ *P1 (retrievability).* B5 beats B1 on pooled test-split MRR and nDCG\@10
  with non-overlapping bootstrap CIs.
+ *P2 (signals).* B5 beats each single-signal ablation (B3, B4) pooled;
  disabling the steering boost measurably hurts T2.
+ *P3 (drill strategies).* The trade is question-shaped: summary-first wins
  gold-term coverage on broad tasks (T1/T3); the raw window wins
  coverage-per-token on pointed lookups (T2).
+ *P4 (judged efficiency).* B5/B6 match or beat B1's judged accuracy at
  ≥ 2× fewer tokens-to-correct.
+ *P5 (scale, the RISE crossover).* B1's accuracy and latency degrade
  monotonically with corpus size; B5 holds flat within CI. A crossover
  point exists on session data @rise2026.
+ *P6 (freshness — problem 2).* Rung-1 quality shows no decay with
  target-session age; full index rebuild stays under minutes at the
  corpus's full size.
+ *P7 (the L3 gate — problems 2 and 4).* Wiki pages beat recall+drill on
  tokens for broad queries at comparable coverage; pages decay at a
  measurable weekly rate; cross-repo mode adds coverage on topics that
  span repos. If pages age fast or lose on coverage, the cached L3 layer
  is not built — that negative result is a finding of this paper, not a
  gap in it.
+ *P8 (label validity — problem 1).* T1 label precision ≥ 0.9 on a
  50-pair human audit.

= Results

#text(fill: rgb("#b91c1c"), style: "italic")[All values in this section are
placeholders pending the corpus run; tables define the exact shape of the
report and are keyed to \u{00A7}6's predictions.]

== Retrievability (P1, P2)

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
  Verdicts: P1 #tbd[holds / fails], P2 #tbd[holds / fails]. B2 has no
  per-query retrieval and is evaluated at rungs 2–3 only.],
) <tab-rung1>

== Drill strategies (P3)

#figure(
  table(
    columns: (auto, auto, auto, auto),
    align: (left, center, center, center),
    table.header([*Drill strategy*], [*Tokens*], [*Coverage*], [*Cov. / 1k tok*]),
    [window (5-turn, at match)], tbd[·], tbd[·], tbd[·],
    [summary-first (pointer)], tbd[·], tbd[·], tbd[·],
  ),
  caption: [Judge-free drill-strategy proxy, paired on queries where recall
  reaches the gold session and a compaction summary exists. Verdict: P3
  #tbd[holds / fails] (per-task split reported alongside).],
) <tab-drill>

== Answer quality and token cost (P4)

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
  Verdict: P4 #tbd[holds / fails].],
) <tab-rung23>

== The L3 gate (P7)

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
  caption: [The wiki experiment prices the browsing cache: generation cost
  is measured from the ledger's own lineage records — the memory system
  accounts for its own distillation. Verdict: P7 #tbd[holds / fails /
  cache not built].],
) <tab-wiki>

== Scale and freshness (P5, P6)

We re-run rung 1 at date-cut corpus subsets. Deliverables:
accuracy-versus-corpus-size and tokens-versus-accuracy curves
(Fig.~#tbd[3], #tbd[4]); B1 latency and failure rate versus corpus size;
rebuild wall-clock versus turns; rung-1 quality bucketed by target-session
age. Verdicts: P5 #tbd[holds / fails], P6 #tbd[holds / fails].

== Agent-in-the-loop (rung 4)

#tbd[10–20 tasks; steering interventions, dead-end re-proposals,
time-to-done, with and without Rekal] — run last, only if rungs 1–3 hold.

= Positioning

The design confrontations live inline in \u{00A7}4; what remains is where
Rekal sits on the literature's two axes. On the *compression axis* —
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
interaction @dci2026 @grepseek2026 the position is conditional and matches
RISE @rise2026: below some corpus size grep is fine — Rekal still adds
parsing, scrubbing, provenance, and team sync — and above it a bounded
interaction space is the difference between answers and wall-clock
failures; the crossover on session data is an output of this work (P5).
Self-improvement from traces @rho2026 @automem2026 @lrat2026 is the
flywheel this corpus natively enables (\u{00A7}9); LoCoMo-style persona
memory @locomo2024 is out of scope by design.

= Limitations

Rung 1 is a retrievability proxy — a hit is not a useful answer; that is
why rungs 2–4 exist, and why P3/P7 are labeled cost-coverage proxies, never
answer quality. Self-labels are noisy; P8 quantifies it and results are
reported with label-precision-adjusted upper bounds. A single-operator
corpus is a case study until replicated — the honest path to multi-corpus
evidence is that the harness is public, fully local, and runnable by any
Rekal user on their own store, at zero annotation cost. The wire format
carries no harness identity for cross-team sessions today; the worked
example's JSON is illustrative pending the corpus run's substitution of a
real (redacted) instance.

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
structure into the first benchmark for repo-grounded intent recall — and
every prediction in \u{00A7}6 is falsifiable by running it, locally, on
your own history.

#v(4pt)
*Reproducibility.* Engine, skills, benchmark spec, extraction SQL, runbook
(`DATA-RUN.md`), and this paper's source:
#link("https://github.com/rekal-dev/rekal-cli")[github.com/rekal-dev/rekal-cli]
(`docs/research/`). All benchmark data remains on the operator's machine;
published artifacts are aggregates, prompts, and code.

#bibliography("refs.bib", style: "ieee", title: "References")
