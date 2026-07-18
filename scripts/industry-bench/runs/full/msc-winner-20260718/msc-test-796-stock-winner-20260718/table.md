# smoke msc-test-796

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 79.8
- answer_path_tokens mean: 90.7

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q2 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.56 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.92 |
| q8 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.59 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q10 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.9 |
