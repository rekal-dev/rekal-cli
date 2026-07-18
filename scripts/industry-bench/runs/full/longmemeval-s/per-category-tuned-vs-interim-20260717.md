# LME-S per-category: dev-tuned (bm25 .55) vs interim (bm25 .30)

| category | n | interim@5 | tuned@5 | Δ@5 | interim@10 | tuned@10 | Δ@10 |
|---|--:|--:|--:|--:|--:|--:|--:|
| knowledge-update | 62 | 1.000 | 1.000 | +0.000 | 1.000 | 1.000 | +0.000 |
| multi-session | 108 | 0.981 | 0.991 | +0.009 | 0.991 | 0.991 | +0.000 |
| single-session-assistant | 50 | 1.000 | 1.000 | +0.000 | 1.000 | 1.000 | +0.000 |
| single-session-preference | 22 | 0.636 | 0.727 | +0.091 | 0.636 | 0.727 | +0.091 |
| single-session-user | 53 | 0.943 | 0.962 | +0.019 | 0.943 | 0.962 | +0.019 |
| temporal-reasoning | 105 | 0.914 | 0.933 | +0.019 | 0.943 | 0.962 | +0.019 |
