# smoke msc-test-676

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 100.6
- answer_path_tokens mean: 116.5

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q6 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.8 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q10 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.84 |
