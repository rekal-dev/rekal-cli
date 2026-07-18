# smoke msc-valid-649

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 92.9
- answer_path_tokens mean: 112.5

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.48 |
| q3 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.9 |
| q4 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q6 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.88 |
| q8 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.81 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q10 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.92 |
