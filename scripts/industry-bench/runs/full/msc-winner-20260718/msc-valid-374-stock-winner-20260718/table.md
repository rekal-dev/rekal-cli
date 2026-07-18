# smoke msc-valid-374

- questions: 8
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 111.0
- answer_path_tokens mean: 126.8

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.72 |
| q2 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.52 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.89 |
| q5 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.81 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.92 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.84 |
