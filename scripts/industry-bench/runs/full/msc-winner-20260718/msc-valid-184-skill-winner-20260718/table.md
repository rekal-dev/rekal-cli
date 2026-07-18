# smoke msc-valid-184

- questions: 8
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 97.4
- answer_path_tokens mean: 110.9

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.85 |
| q2 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q3 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q4 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q5 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.58 |
| q6 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q8 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.81 |
