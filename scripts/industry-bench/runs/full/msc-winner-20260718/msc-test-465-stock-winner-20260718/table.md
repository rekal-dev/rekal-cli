# smoke msc-test-465

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 106.7
- answer_path_tokens mean: 125.7

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.91 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.92 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.59 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q8 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.76 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q10 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.59 |
