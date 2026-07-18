# smoke msc-valid-42

- questions: 8
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 83.4
- answer_path_tokens mean: 100.5

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q2 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.61 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q8 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.61 |
