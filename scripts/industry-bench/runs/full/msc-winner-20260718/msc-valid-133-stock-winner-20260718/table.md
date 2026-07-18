# smoke msc-valid-133

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 104.0
- answer_path_tokens mean: 121.6

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.89 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q5 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.65 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.89 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q10 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.91 |
