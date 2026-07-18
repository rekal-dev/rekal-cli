# smoke msc-test-241

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 110.0
- answer_path_tokens mean: 129.1

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q3 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.79 |
| q4 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.84 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q6 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.79 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.37 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.53 |
| q10 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.83 |
