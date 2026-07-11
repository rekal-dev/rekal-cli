# Positioning: the claim and the evidence ladder

Working backwards: write the claim first, then derive exactly what evidence
would make it true, then (in `03`/`04`) the cheapest honest way to get it.

## The claim

> **Rekal is the natural memory tool for AI coding agents.** An agent with
> Rekal answers "why is this code like this?" and "what did we already try?"
> correctly, at a fraction of the context tokens, from memory that maintains
> itself — captured at every commit, verified by the merge gate, never
> hand-curated, never stale, never leaving git and the machine.

Decomposed, that is five falsifiable sub-claims:

| # | Sub-claim | Falsifier |
|---|---|---|
| C1 | Rekal retrieves the right prior session for a real question better than the alternatives | lower Recall@k/MRR than grep or vector-only on self-labeled pairs |
| C2 | The retrieved context yields *correct answers*, not just plausible hits | LLM-judged answer accuracy no better than baselines |
| C3 | It does so at materially lower token cost | tokens-to-correct-answer ≥ transcript-dump or grep-drill |
| C4 | Zero-maintenance is real | index rebuild cost grows badly, or recall decays as corpus ages |
| C5 | Agents with Rekal do real tasks better | A/B shows no reduction in steering count / dead-end re-exploration |

## Why "natural"

Every competing memory system in the literature has to *manufacture* three
things Rekal gets for free from git:

1. **Ground truth.** Dialogue-memory systems (AdaMem, EMem, MRAgent) evaluate
   on hand-annotated benchmarks (LoCoMo, LongMemEval, PERSONAMEM) because
   conversations have no objective labels. Coding sessions do: **the commit**.
   `checkpoint_sessions` links every commit to the sessions that produced it —
   free, abundant, objective supervision for both benchmarking *and* future
   learned components.
2. **A verification gate.** EDV shows self-judged memories poison themselves
   (the self-confirmation trap). Rekal's merged-only export is an external
   verifier: shared memory admits only experience whose code landed on main —
   human-reviewed, CI-tested. No dialogue system has an equivalent signal.
3. **A freshness story.** LLM-Wiki, AdaMem, MRAgent, and SAG all defer
   "maintenance of derived structure over time" — the single most repeated
   open problem in the source set. Rekal's answer is architectural, not
   algorithmic: the source of truth (`data.db`) is append-only and raw; every
   derived structure (`index.db`) is disposable and rebuilt from it.
   *Derived structure you can throw away cannot go stale.*

## The evidence ladder

Four rungs, cheapest first. Each rung is publishable on its own; each higher
rung strengthens the claim. Stop-loss: if rung 1 fails against grep, fix the
engine before climbing.

### Rung 1 — Retrievability (automatic, thousands of labels)
Given (query → known-relevant session) pairs mined from the corpus's own
structure, compare Recall@k / MRR / nDCG across systems.
- **Labels:** `checkpoint_sessions` (commit→session), steering turns
  (question→its session), file paths (file→sessions that authored it).
- **Cost:** zero human annotation; pure SQL + a query-paraphrase pass.
- **Proves:** C1. **Threat model it must beat:** grep/DCI over raw JSONL,
  vector-only, BM25-only (all runnable as Rekal weight ablations or trivially
  scripted).

### Rung 2 — Answer quality (LLM-judged QA)
Generate provenance/decision questions from held-out session material; an
answering agent gets context only through the system under test; an LLM judge
scores correctness against the source turn.
- **Proves:** C2. Mirrors how LoCoMo/LongMemEval are actually scored, so the
  numbers are legible to the research community.

### Rung 3 — Token efficiency (the industry-standard axis)
Same questions as rung 2, but the reported metric is **tokens loaded to reach
a correct answer** (and wall-clock). MRAgent and RISE both make efficiency a
headline result; RISE's $1.10 → $0.28/query framing is the template.
- **Proves:** C3, and quantifies the progressive-drill design
  (search → `--role human_steering` → windowed turns → `--full` only if needed).

### Rung 4 — Agent-in-the-loop A/B (small N, most convincing)
Real tasks in a repo with rich history, same agent, with vs. without Rekal
(and vs. grep-only). Measure: human steering interventions, re-proposal of
known-abandoned approaches, time-to-done, final diff quality.
- **Proves:** C5. Expensive (needs a human or scripted rater); run last, on
  ~10–20 tasks, as the headline anecdote-with-numbers.

### C4 (zero-maintenance) — measured alongside
Report index rebuild time and recall-vs-corpus-age curves from the same runs:
rebuild wall-clock at N sessions, and rung-1 metrics bucketed by session age.
No decay + linear rebuild = the freshness claim, quantified.

## What we do *not* claim

Honesty budget — stated up front in anything published:
- Retrievability (rung 1) is a proxy: a hit ≠ a useful answer. That is why
  rungs 2–4 exist.
- Self-labels are noisy: a commit's sessions include some irrelevant chatter.
  We measure label precision on a 50-pair hand-audited sample and report it.
- LoCoMo-style dialogue benchmarks are *out of scope*: Rekal is not a chat
  persona memory. We compare on our domain (repo-grounded intent recall),
  where no established benchmark exists — RekalBench is the contribution.
- Single-corpus results (one operator's machine) are a case study until
  replicated on a second corpus; the harness must be runnable by anyone
  (`rekal` + scripts, everything local).
