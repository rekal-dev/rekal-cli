# smoke msc-test-170

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 99.4
- answer_path_tokens mean: 114.7

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q6 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.61 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.98 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.59 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.96 |
| q10 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.83 |
