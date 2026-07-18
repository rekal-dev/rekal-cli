# LoCoMo weight sweep — conclusion (2026-07-17)

Winner: **locomo-bm25-push** (bm25 .55 / lsa .10 / semantic .35, facet .05).

| profile | evidence@5 | evidence@10 |
|---|---:|---:|
| skill/locomo-bm25-push | 0.8927 | 0.9201 |
| stock/no-cal | 0.8742 | 0.8968 |

Delta vs stock: +1.85pt @5, +2.33pt @10 across 9 convs.
LoCoMo is keyword-heavy chat QA → BM25-weighted retrieval wins.
Scope: apply to LoCoMo calibration profile only; do NOT overwrite repo
recall weights (cross-domain overfit risk). LME-S/BEAM keep own calibration.
