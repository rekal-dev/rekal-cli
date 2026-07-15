# Paper: "Why Git Is the Memory Solution for the Agentic Development Lifecycle"

The unified flagship paper — *Why Git Is the Memory Solution for the
Agentic Development Lifecycle* (subtitle: git-bound, routed memory for
coding agents), coining the ADLC term. It
supersedes v1 ("The Commit Is the Label", available in git history),
whose guarantee arguments it compresses into §3 and whose retrievability
numbers it absorbs into the seed-stage study (§5). The structure follows
`../07-paper-restructure.md` (ladder body, coverage-at-cost ending).

- `rekal-paper.typ` — Typst source (canonical)
- `refs.bib` — bibliography (BibTeX; the literature-map source set + SPM)
- `rekal-paper.pdf` — compiled output, checked in for convenience

## Build

```bash
pip install typst          # bundles the compiler; no LaTeX needed
python3 -c "import typst; typst.compile('rekal-paper.typ', output='rekal-paper.pdf')"
# or, with the typst CLI installed:  typst compile rekal-paper.typ
```

## Data provenance

Empirical values come from two sources. (1) The 2026-07-12 single-corpus
rung-1 run (Corpus A raw-grep floor, hybrid/BM25 composition, drill cost);
its aggregate manifest is committed under `../runs/` per `DATA-RUN.md` §6.
(2) The consolidated multi-corpus run record (8-corpus retrieval matrix,
B1′ parsed-grep floors, mechanism sweeps, SPM facet results, and the
mode/sufficiency tables with their ablations) — held on the operator's
machine; corpora appear in the paper anonymized by workload class only
(Corpus A/B + six small), per the anonymization rule in
`../07-paper-restructure.md`. Before submission, the consolidated record's
aggregate tables should be committed under `../runs/` in the same
anonymized form, and every paper number proofread against it — several
table cells were transcribed from the source record and must be verified.
Derived cells needing exact values from the record: the best-config row
of the seed-stage table (≈0.31 Corpus A, ≈0.24 Corpus B = tuned baseline
+ facet marginal). Seed-stage detail tables (facet operating points,
embedding options) live in Appendix A; the coverage-at-cost waterfall
(Table 4) is the paper's summary artifact.

## Extending the result

Specified but not yet run (§12 of the paper): synthesis on the mined T4
multi-hop gold, a second judge model with agreement, the wild-question
kind-distribution (which upgrades §8.3 to a single expected-cost figure),
and gate recalibration after the facet term enters the score mix. All
reuse the same public harness at zero further labeling cost.
