# smoke msc-valid-79

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 107.8
- answer_path_tokens mean: 127.0

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q5 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.81 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q8 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.56 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q10 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.88 |
