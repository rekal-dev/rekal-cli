# smoke msc-valid-693

- questions: 8
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 110.8
- answer_path_tokens mean: 135.1

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q7 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.87 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.63 |
