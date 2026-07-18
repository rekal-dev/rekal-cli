# smoke msc-test-975

- questions: 9
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 101.1
- answer_path_tokens mean: 119.8

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.94 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.97 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.97 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.97 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q7 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q9 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.61 |
