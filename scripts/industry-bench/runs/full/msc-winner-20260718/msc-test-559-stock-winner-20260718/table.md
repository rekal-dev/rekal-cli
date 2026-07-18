# smoke msc-test-559

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 102.7
- answer_path_tokens mean: 118.6

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.88 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q7 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.66 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q10 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.66 |
