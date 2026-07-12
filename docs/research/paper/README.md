# Paper: "The Commit Is the Label"

- `rekal-paper.typ` — Typst source (canonical)
- `refs.bib` — bibliography (BibTeX; the 18-paper source set + LoCoMo)
- `rekal-paper.pdf` — compiled output, checked in for convenience

## Build

```bash
pip install typst          # bundles the compiler; no LaTeX needed
python3 -c "import typst; typst.compile('rekal-paper.typ', output='rekal-paper.pdf')"
# or, with the typst CLI installed:  typst compile rekal-paper.typ
```

## Data provenance

All empirical values come from the 2026-07-12 corpus run on a real
1,433-session store; its aggregate manifest is committed under `../runs/`
per `DATA-RUN.md` §6 (see `../../../scripts/bench/README.md` for the
harness). The paper reports the rung-1 retrievability result and the bounded
drill-cost figure; the source carries no placeholders.

## Extending the result

The natural next rungs — judged answer quality against the gold turn, the
scale sweep against an agentic-grep baseline, and the embedding-substitution
test — reuse the same public harness and the already-mined query set. When a
run produces those numbers, add the corresponding table and cite the run
directory; the paper's structure (§7 Results, §9 Future Work) is written to
accommodate them without disturbing the retrievability result.
