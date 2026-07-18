# smoke msc-test-107

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 113.1
- answer_path_tokens mean: 126.2

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q2 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q3 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q4 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.91 |
| q5 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q6 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.94 |
| q7 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.91 |
| q8 | persona-fact | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q9 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q10 | persona-fact | STOCK | 1 | 1 | 0 | hit | 0.87 |
