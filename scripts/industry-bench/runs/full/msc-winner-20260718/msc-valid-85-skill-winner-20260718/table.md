# smoke msc-valid-85

- questions: 10
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 106.9
- answer_path_tokens mean: 122.5

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q2 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.92 |
| q3 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q4 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q5 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q6 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.92 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.88 |
| q8 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.86 |
| q9 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q10 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.92 |
