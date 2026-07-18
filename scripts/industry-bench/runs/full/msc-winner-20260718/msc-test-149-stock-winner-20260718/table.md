# smoke msc-test-149

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 61.2
- answer_path_tokens mean: 72.0

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q2 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q6 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.94 |
| q7 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.78 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.91 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q10 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
