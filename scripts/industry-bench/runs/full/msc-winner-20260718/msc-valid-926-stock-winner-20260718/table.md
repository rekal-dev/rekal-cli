# smoke msc-valid-926

- questions: 8
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 78.8
- answer_path_tokens mean: 90.9

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q6 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.87 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.76 |
