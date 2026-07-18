# smoke msc-valid-403

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 124.3
- answer_path_tokens mean: 141.5

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.94 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.86 |
| q3 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.47 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 1 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.6 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q10 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.68 |
