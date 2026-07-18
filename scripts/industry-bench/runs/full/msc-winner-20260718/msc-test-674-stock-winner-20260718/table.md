# smoke msc-test-674

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 85.8
- answer_path_tokens mean: 103.5

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.95 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.89 |
| q4 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.64 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.96 |
| q7 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.92 |
| q9 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q10 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.95 |
