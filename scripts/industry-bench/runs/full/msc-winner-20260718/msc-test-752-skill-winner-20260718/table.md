# smoke msc-test-752

- questions: 10
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 88.2
- answer_path_tokens mean: 103.2

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q2 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.94 |
| q3 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.89 |
| q4 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.94 |
| q5 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q6 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.62 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.89 |
| q8 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.94 |
| q9 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q10 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.94 |
