# smoke msc-valid-190

- questions: 8
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 78.2
- answer_path_tokens mean: 96.1

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q2 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q3 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q4 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q5 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q6 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q7 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q8 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.87 |
