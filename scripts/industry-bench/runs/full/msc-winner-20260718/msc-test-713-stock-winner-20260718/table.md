# smoke msc-test-713

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 88.1
- answer_path_tokens mean: 102.2

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.99 |
| q6 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q10 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.87 |
