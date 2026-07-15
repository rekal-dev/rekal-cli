# arXiv version

LaTeX conversion of the flagship paper for arXiv submission. The Typst
source (`../rekal-paper.typ`) stays canonical for content; this file is
the arXiv artifact — keep them in sync (content changes land in the Typst
source first, then get mirrored here).

## Build

```bash
pdflatex main && bibtex main && pdflatex main && pdflatex main
```

pdflatex-only (no XeLaTeX/LuaLaTeX needed): lmodern + microtype, natbib
(numeric), booktabs, TikZ for Figure 1. Compiles clean — zero errors,
zero overfull boxes — with TeX Live 2023.

## Upload to arXiv

Submit `main.tex`, `main.bbl`, and `refs.bib` (arXiv compiles from the
`.bbl`; the `.bib` is included for completeness). No external figures —
Figure 1 is inline TikZ. Suggested categories: cs.SE (primary), cs.AI,
cs.IR.

## Before submitting

Same gate as the Typst PDF (see `../README.md`): verify the transcribed
run-record cells against the operator copy, replace the derived
best-config cells (~0.31 A / ~0.24 B) and floor ratios with exact values,
and re-freeze after gate recalibration. Do not submit while the
consolidated run record's status is `TRANSCRIBED_PENDING_VERIFICATION`.
