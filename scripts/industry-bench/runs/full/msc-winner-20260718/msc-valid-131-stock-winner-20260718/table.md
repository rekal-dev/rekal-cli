# smoke msc-valid-131

- questions: 8
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 95.6
- answer_path_tokens mean: 110.0

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.91 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q8 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.7 |
