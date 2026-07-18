# smoke msc-test-65

- questions: 10
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 104.0
- answer_path_tokens mean: 125.5

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.91 |
| q2 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q3 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.89 |
| q4 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q5 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.88 |
| q6 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.91 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.89 |
| q8 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q9 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q10 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.91 |
