# smoke msc-test-604

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 104.5
- answer_path_tokens mean: 123.5

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q3 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.5 |
| q4 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.78 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.59 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q10 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.57 |
