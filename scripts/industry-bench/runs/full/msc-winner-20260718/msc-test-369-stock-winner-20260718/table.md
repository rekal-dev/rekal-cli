# smoke msc-test-369

- questions: 9
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 86.4
- answer_path_tokens mean: 108.0

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.92 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q4 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.58 |
| q5 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.93 |
| q6 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.82 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.95 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.61 |
