# smoke msc-test-763

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 76.3
- answer_path_tokens mean: 93.5

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.52 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q5 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.52 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q10 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.84 |
