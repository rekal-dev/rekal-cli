# smoke msc-valid-166

- questions: 9
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 94.6
- answer_path_tokens mean: 113.1

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q4 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q8 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.77 |
