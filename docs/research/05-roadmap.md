# Roadmap: literature-derived product adjustments

The second half of the goal: where the papers show a genuinely better design,
adjust Rekal. Ordered by leverage-per-effort. Every item obeys the standing
rule from RHO (`02` §1): **no change ships without beating the incumbent on
the RekalBench regression set.** That is why the benchmark comes first — it
is the acceptance test for everything below.

## R0 — Run the benchmark (prerequisite for all else)
`03`/`04`. Also the marketing asset: the corpus card + numbers are the data
behind the positioning. *Effort: days. Source: RHO discipline.*

## R1 — Implicit relevance logging → per-corpus weight tuning (LRAT)
After a recall, which session the agent drills into (`rekal query --session`
following a search) is an implicit relevance label. Log locally, opt-in
(`.rekal/config.json: telemetry_local: true`; a small `query_log` table in
`index.db` — never in `data.db`, never exported). After N labels, a
`rekal tune` pass grid-searches weights against the log and proposes a
config change (applied only if it beats incumbent — R0 harness).
*Effort: small (log table + offline script first). Source: LRAT, RHO.*
*This is the data flywheel: every user's Rekal gets better on their own
corpus, privately.*

## R2 — Expose per-layer scores + related-session joins in recall JSON (SAG)
Add to each result (behind `--explain` or always): per-layer scores
(bm25/lsa/nomic before weighting) and a `related` list (sessions sharing ≥k
files, via `file_cooccurrence` reverse join at query time). Cheap; makes the
bench's ablation analysis trivial and gives skills a native zoom edge.
*Effort: small. Source: SAG; also serves 03 §6.*

## R3 — Decision-drift surfacing (BeliefShift)
When recall returns multiple sessions touching the same files with steering
turns, order/annotate by recency so "the later decision" is visible
(`superseded_candidate: true` on older hits in the same file-cluster). Query-
time only, derived from existing facets. The `rekal-distill` skill then says
it explicitly in the decision library.
*Effort: small-medium. Source: BeliefShift; bench task T5 measures it.*

## R4 — Formalize the reflect → rules loop (LLM-Wiki's Error Book, EDV)
`rekal-reflect` already mines `human_steering` into rules. Two upgrades:
(a) a conventional output home (`docs/agent-rules.md` or a CLAUDE.md section)
so rules are load-bearing, and (b) an EDV-style admission rule *documented in
the skill*: promote a rule only when observed ≥3× across sessions (already
drafted) **and** at least once on a merged branch (external verification —
never distill a rule from dead-end-only evidence).
*Effort: docs/skill only. Source: LLM-Wiki, EDV.*

## R5 — Dead-end labeling in recall output (EDV)
Recall results from branches that never merged should carry
`merged: false` (computable via the existing gate helpers) so agents/skills
can present them as boundary knowledge ("tried, didn't land"), never as
endorsed practice. The boundary library (rekal-distill) gets its signal from
the engine instead of inferring it.
*Effort: medium (plumb gitx ancestry check into recall path; cache it).
Source: EDV; bench task T3 measures the payoff.*

## R6 — Skill self-improvement from own usage traces (AutoMem)
Rekal's corpus contains sessions where the agent *used* rekal. Periodically
census those (rekal-census scope: `cmd_prefix LIKE 'rekal%'`), find where the
agent flailed (repeated searches, wrong drills), and revise the skill texts.
Manual at first; the AutoMem loop (LLM proposes skill edits, accepted on
bench wins) later.
*Effort: process first, automation later. Source: AutoMem, RHO.*

## R7 — Security posture note + contamination regression test
Document the memory-lifecycle story: scrub-before-store (the order proven
safe by the state-contamination paper), append-only auditability, provenance
chain, write path = own git hooks only, merged-gate as admission control.
Add one regression test: a poisoned string (prompt-injection-shaped) in a
session must arrive in recall output *scrubbed/inert* per current scrub
rules, and its provenance must be traceable.
*Effort: small. Source: papers 14, 16, 17. Also a sales page later (15).*

## R8 — Bench harness graduates into the repo
Once `scripts/bench/` stabilizes (R0), promote to `cmd/rekalbench` or keep as
maintained scripts — decision after v0 results. Publishing the harness is
what makes the benchmark a community artifact (and RekalBench citable).
*Effort: medium. Source: 03 §6.*

## Explicit non-goals (decided against, with reasons)

- **Compile-time knowledge graph / wiki over sessions** (LLM-Wiki, MRAgent
  storage layer): re-introduces the maintenance problem Rekal's architecture
  exists to avoid; SAG shows query-time joins suffice. Revisit only if T4
  multi-hop numbers embarrass us.
- **Write-time summarization/consolidation** (AdaMem tiers): EMem's result +
  contamination paper both argue for preserve-raw; Rekal distills at query
  time via skills instead.
- **A memory server / API**: against the soul; also the security surveys
  show open write paths are the attack surface. Git remains the only wire.
- **Chasing LoCoMo SOTA**: wrong domain (chat personas); our benchmark is
  repo-grounded intent recall, where we define the standard.

## Sequence

R0 → (R2, R4, R7 in parallel — all small) → R1 → R3/R5 → R6/R8.
R0's numbers gate everything; if grep (B1) wins rung 1, the priority flips
to engine work (weights, snippet quality) before any feature above.
