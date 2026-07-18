# smoke msc-test-375

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 98.9
- answer_path_tokens mean: 114.0

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.54 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.92 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.91 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q9 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q10 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.81 |
