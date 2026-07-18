# smoke msc-valid-797

- questions: 9
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 111.1
- answer_path_tokens mean: 127.0

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q5 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.72 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.8 |
