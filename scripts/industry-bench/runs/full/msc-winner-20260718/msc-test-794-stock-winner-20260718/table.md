# smoke msc-test-794

- questions: 9
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 99.4
- answer_path_tokens mean: 117.0

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.91 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q7 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q8 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.71 |
