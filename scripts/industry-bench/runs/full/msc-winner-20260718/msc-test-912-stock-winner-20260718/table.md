# smoke msc-test-912

- questions: 9
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 118.0
- answer_path_tokens mean: 133.2

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.6 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q6 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.81 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.89 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.68 |
