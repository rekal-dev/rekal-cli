# smoke msc-valid-663

- questions: 9
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 70.2
- answer_path_tokens mean: 86.2

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.55 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q6 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.73 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.72 |
