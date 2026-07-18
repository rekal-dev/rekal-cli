# smoke msc-test-392

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 111.9
- answer_path_tokens mean: 128.2

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.95 |
| q4 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.74 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q6 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.85 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q10 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.83 |
