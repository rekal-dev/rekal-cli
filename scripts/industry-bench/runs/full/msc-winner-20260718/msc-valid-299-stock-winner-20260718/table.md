# smoke msc-valid-299

- questions: 8
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 101.1
- answer_path_tokens mean: 115.8

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.88 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.97 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.96 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.96 |
| q7 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.46 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.96 |
