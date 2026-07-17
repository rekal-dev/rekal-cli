# Paper #2 Plan — Claims Ladder and Evidence Gaps

Working title: *"Does Git-Bound Memory Transfer? An ADLC Memory System on
the General Long-Term-Memory Benchmarks"* (title TBD by WS-G; keep the
question form — the paper is an honest transfer study, not a leaderboard
grab).

Relationship to paper #1 (arXiv:2607.14390): #1 argued the design and
measured it on self-labeled ground truth. The standing objection is
"benchmarks you mined yourself, on your home domain." #2 answers it on the
field's shared datasets with the field's own judges, and imports #1's
signature discipline: token-budget accounting and pre-registered gates.

## Claims ladder (weakest → strongest; each row cites the run that proves it)

| # | Claim | Evidence required | Status |
|---|---|---|---|
| 1 | The commit anchor is a *format*, not a domain restriction: general conversational corpora ingest through the unchanged production pipeline. | WS-B round-trip + full LongMemEval-S ingest verification | open |
| 2 | Stock hybrid recall is competitive with purpose-built memory systems on single-hop/temporal categories, at a fraction of the context tokens. | WS-E/F headline table, LongMemEval-S + full | open |
| 3 | The confidence gate transfers: abstention-category accuracy from the *same* SILENCE mechanism shipped for coding, with disclosed false-silence rate. | Per-category results + gate analysis | open |
| 4 | Routing transfers: persona-map + synthesis modes beat single-shot retrieval on preference/knowledge-update categories, mirroring #1's answer-assembly finding. | Ablation: routed vs recall-only, same corpus/pins | open |
| 5 | Tokens-per-question remains 10²–10³ below full-context at comparable accuracy — the cost column the vendor pages omit. | Token columns across all tables + full-context baseline | open |
| 6 | (stretch) The structure holds under 10M-token pressure (BEAM tier). | WS-F stretch run | open |

Rows that fail become findings sections (the mechanism-graveyard model from
paper #1) — a documented non-transfer is publishable content, silence about
it is not.

## Planned structure

1. **The objection** — self-labeled home-turf results, and why transfer must
   be shown on shared ground.
2. **The mapping** — session→commit as synthetic history; what each of the
   four substrates becomes in a chat corpus (the table from
   [02 §2](02-adapter-architecture.md)); what was recalibrated (gates, dev
   split only, pre-registered) and what was frozen (everything else).
3. **Setup** — benchmarks and their known flaws (leading with them, not
   burying them: LoCoMo audit, LongMemEval-S window criticism); pins;
   baselines we ran ourselves.
4. **Results** — per-category tables with mandatory token columns;
   gate/abstention analysis; routed-vs-recall ablation.
5. **What didn't transfer** — reserved; filled from failed ladder rows.
6. **Cost** — tokens-per-question across systems; the thin-wire argument in
   a non-git domain.
7. **Threats** — synthetic-history artifacts (backdated commits, marker
   files), judge limitations, out-of-domain calibration.

## Venue / positioning notes

- Primary target: same track as paper #1 (cs.SE crossover to cs.AI/cs.IR
  memory work); decide after results.
- The benchmark-flaw handling (LoCoMo dual-column with/without bad labels)
  is itself a small contribution — surface it, cite the Penfield audit.
- Reuse paper #1's Typst tooling (`docs/research/paper/`); provenance README
  convention carries over.

## Kill criteria

Program is stopped (not massaged) if either: (a) claim 1 fails — ingestion
requires core changes, meaning the "format not domain" thesis is wrong as
stated; or (b) claims 2 AND 4 both fail badly after honest calibration —
then the finding is written up as a negative-result note in
`runs/consolidated/` instead of a paper, and that note is still published to
the repo.
