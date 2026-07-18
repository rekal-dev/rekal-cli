# smoke msc-valid-447

- questions: 8
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 99.9
- answer_path_tokens mean: 117.2

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q2 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.81 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q4 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.82 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.91 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.58 |
| q7 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.64 |
