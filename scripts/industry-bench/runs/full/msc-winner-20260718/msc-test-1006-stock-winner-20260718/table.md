# smoke msc-test-1006

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 97.5
- answer_path_tokens mean: 111.0

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.98 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.89 |
| q6 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.85 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.88 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q10 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.8 |
