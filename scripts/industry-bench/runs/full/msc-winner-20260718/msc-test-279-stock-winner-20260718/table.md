# smoke msc-test-279

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 98.2
- answer_path_tokens mean: 117.8

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.54 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q4 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q8 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.7 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q10 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.84 |
