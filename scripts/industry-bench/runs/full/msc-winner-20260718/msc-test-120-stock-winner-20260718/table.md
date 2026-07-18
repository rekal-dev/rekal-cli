# smoke msc-test-120

- questions: 9
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 110.7
- answer_path_tokens mean: 129.7

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.98 |
| q2 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.94 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.88 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.91 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.88 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.96 |
