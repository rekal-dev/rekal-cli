# smoke msc-valid-373

- questions: 9
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 112.1
- answer_path_tokens mean: 131.0

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q2 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q3 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q4 | persona-fact | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.7 |
| q5 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q6 | persona-fact | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.7 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q8 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q9 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.65 |
