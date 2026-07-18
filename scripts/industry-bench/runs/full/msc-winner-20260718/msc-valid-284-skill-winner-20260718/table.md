# smoke msc-valid-284

- questions: 10
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 83.8
- answer_path_tokens mean: 100.2

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q2 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q3 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q4 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q5 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q6 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q8 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q9 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q10 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.82 |
