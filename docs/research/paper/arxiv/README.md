# arXiv version

**Live:** [arXiv:2607.14390](https://arxiv.org/abs/2607.14390) —
*Why Git Is the Memory Solution for the Agentic Development Lifecycle*.

LaTeX conversion of the flagship paper. The Typst source
(`../rekal-paper.typ`) stays canonical for content edits; this tree is the
arXiv artifact — keep them in sync (content changes land in Typst first,
then get mirrored here for any revision).

## Build

```bash
pdflatex main && bibtex main && pdflatex main && pdflatex main
```

pdflatex-only (no XeLaTeX/LuaLaTeX needed): lmodern + microtype, natbib
(numeric), booktabs, TikZ for Figure 1. Compiles clean — zero errors,
zero overfull boxes — with TeX Live 2023.

## Upload / revise

Submit or replace `main.tex`, `main.bbl`, and `refs.bib` (arXiv compiles
from the `.bbl`; the `.bib` is included for completeness). No external
figures — Figure 1 is inline TikZ. Categories: cs.SE (primary), cs.AI,
cs.IR.

Regenerate the zip for a revision:

```bash
cd arxiv && zip -j ../rekal-arxiv-submission.zip main.tex main.bbl refs.bib
```
