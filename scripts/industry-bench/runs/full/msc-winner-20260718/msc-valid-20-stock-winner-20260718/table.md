# smoke msc-valid-20

- questions: 8
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 97.5
- answer_path_tokens mean: 112.5

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q3 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.97 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.86 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.88 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q8 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
