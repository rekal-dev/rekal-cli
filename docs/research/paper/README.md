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

## Filling in the data

Every pending value renders as a red `⟨…⟩` marker, produced by the `#tbd[…]`
function at the top of the source. Workflow:

1. Run the corpus extraction and benchmark (`../04-data-plan.md`,
   `../03-benchmark.md`; runnable harness: `../../../scripts/bench/README.md`).
   The rung-1 flow fills Table 3; `run_rung3.py`'s pooled table fills the
   drill-strategy table (`tab-drill`) directly — its three columns (tokens,
   coverage, coverage/1k) match one-to-one.
2. Grep the source for `tbd[` and replace each with the measured value
   (abstract numbers, corpus card in §5.1, Tables 3–4, the drill-strategy
   table, §6.3 curves — add figures as `#image(...)` once the plots exist).
3. Delete the DRAFT banner in the title block and this section's caveat
   when no `tbd[` remains: `grep -c 'tbd\[' rekal-paper.typ` → the residual
   count is the to-do list.
4. Recompile; commit source + PDF together.

Section §3 (design), §4 (benchmark), and Related Work are final and do not
depend on the data.
