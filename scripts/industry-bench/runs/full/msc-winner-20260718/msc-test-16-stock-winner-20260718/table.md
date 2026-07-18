# smoke msc-test-16

- questions: 11
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 96.6
- answer_path_tokens mean: 109.2

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.88 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.88 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q5 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.62 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.88 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.88 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.86 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.88 |
| q10 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.88 |
| q11 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.92 |
