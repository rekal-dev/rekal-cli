# smoke msc-valid-467

- questions: 10
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 91.9
- answer_path_tokens mean: 106.0

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.95 |
| q2 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.95 |
| q3 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q4 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.95 |
| q5 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.95 |
| q6 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q8 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q9 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.95 |
| q10 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.91 |
