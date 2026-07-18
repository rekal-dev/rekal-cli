# smoke msc-test-706

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 99.8
- answer_path_tokens mean: 119.5

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.92 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q3 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.46 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.89 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.89 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q10 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.76 |
