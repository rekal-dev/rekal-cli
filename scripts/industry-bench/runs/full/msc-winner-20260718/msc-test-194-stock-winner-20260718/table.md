# smoke msc-test-194

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 93.8
- answer_path_tokens mean: 113.8

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.97 |
| q4 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.78 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.6 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q7 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.83 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.97 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.97 |
| q10 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.55 |
