# Paper: "Why Git Is the Memory Solution for the Agentic Development Lifecycle"

**Accepted on arXiv:** [arXiv:2607.14390](https://arxiv.org/abs/2607.14390)
([pdf](https://arxiv.org/pdf/2607.14390), [html](https://arxiv.org/html/2607.14390)).

The unified flagship paper — *Why Git Is the Memory Solution for the
Agentic Development Lifecycle* (subtitle: git-bound, routed memory for
coding agents), coining the ADLC term. It
supersedes v1 ("The Commit Is the Label", available in git history),
whose guarantee arguments it compresses into §3 and whose retrievability
numbers it absorbs into the seed-stage study (§5). The structure follows
`../07-paper-restructure.md` (ladder body, coverage-at-cost ending).

- `rekal-paper.typ` — Typst source (canonical for edits)
- `refs.bib` — bibliography (BibTeX; the literature-map source set + SPM)
- `rekal-paper.pdf` — compiled output, checked in for convenience
- `arxiv/` — the arXiv LaTeX version (`main.tex` + `main.bbl`; pdflatex-only,
  TikZ figure, zero-warning build). Content mirrors the Typst source — see
  `arxiv/README.md` for build notes. **Live abs:** `2607.14390`
- `rekal-arxiv-submission.zip` — the archive used for the upload
  (`main.tex`, `main.bbl`, `refs.bib` flat at the root)
- `web/` — drop-in site assets for serving the paper at `rekal.dev/paper`:
  SEO landing page (Scholar `citation_*` tags including arXiv id, OpenGraph,
  Twitter card, JSON-LD). See `web/README.md`

## Build

```bash
pip install typst          # bundles the compiler; no LaTeX needed
python3 -c "import typst; typst.compile('rekal-paper.typ', output='rekal-paper.pdf')"
# or, with the typst CLI installed:  typst compile rekal-paper.typ
```

## Data provenance

Empirical values come from two sources. (1) The single-corpus
rung-1 run (Corpus A raw-grep floor, hybrid/BM25 composition, drill cost);
its aggregate manifest is committed under `../runs/` per `DATA-RUN.md` §6.
(2) The consolidated multi-corpus run record (8-corpus retrieval matrix,
B1′ parsed-grep floors, mechanism sweeps, SPM facet results, and the
mode/sufficiency tables with their ablations) — anonymized aggregates
committed at `../runs/consolidated/manifest.json`. Corpora appear
everywhere anonymized by workload class only (Corpus A/B + six small), per
the anonymization rule in `../07-paper-restructure.md`.

**Product vs paper gate.** The paper’s episode-gate numbers are reported on
the max-normalized hybrid score distribution used in the retrieval study.
The shipped skill gate (`scripts/hunt-gate.py`) now uses absolute
`confidence` / raw BM25 `mass` so junk queries cannot clear the bar by
max-normalization alone — see `docs/design/skill-router.md` and
`docs/spec/command/recall.md`. Ranking still max-norms; silence does not.

## Extending the result

Specified but not yet run (§12 of the paper): synthesis on the mined T4
multi-hop gold, a second judge model with agreement, the wild-question
kind-distribution (which upgrades §8.3 to a single expected-cost figure),
and further gate calibration on absolute confidence after the facet term.
All reuse the same public harness at zero further labeling cost.
