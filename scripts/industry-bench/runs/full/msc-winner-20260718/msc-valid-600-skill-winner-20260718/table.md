# smoke msc-valid-600

- questions: 9
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 84.4
- answer_path_tokens mean: 99.2

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q2 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.86 |
| q3 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.89 |
| q4 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q5 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q6 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q8 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.58 |
| q9 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.74 |
