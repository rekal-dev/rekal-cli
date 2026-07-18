# smoke msc-valid-974

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 83.0
- answer_path_tokens mean: 101.6

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q4 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.85 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q7 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.82 |
| q8 | persona-fact | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.71 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.58 |
| q10 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.76 |
