# smoke msc-valid-60

- questions: 9
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 90.4
- answer_path_tokens mean: 106.3

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.96 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.96 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.94 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q7 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.49 |
| q8 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.49 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.67 |
