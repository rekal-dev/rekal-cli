# smoke msc-test-459

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 93.6
- answer_path_tokens mean: 109.7

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.66 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.55 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.49 |
| q5 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.66 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q7 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.66 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.91 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q10 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.96 |
