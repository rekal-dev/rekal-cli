# smoke msc-test-730

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 96.5
- answer_path_tokens mean: 111.0

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.97 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 1 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.99 |
| q10 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.74 |
