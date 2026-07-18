# LME-S FULL TEST — dev-tuned validation (400-conv test split)

Calibration pins: interim=longmemeval-s.json (bm25 .30), tuned=longmemeval-s-tuned.json (bm25 .55, dev-tuned).

| variant | runs | ev@5 | ev@10 | ctx_tok |
|---|---:|---:|---:|---:|
| stock | 400 | 0.9400 | 0.9575 | 385.3 |
| skill interim (bm25 .30) | 400 | 0.9450 | 0.9550 | 380.1 |
| skill DEV-TUNED (bm25 .55) | 400 | 0.9600 | 0.9675 | 382.4 |
