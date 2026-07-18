# smoke msc-valid-112

- questions: 8
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 90.4
- answer_path_tokens mean: 104.9

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q2 | persona-fact | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.75 |
| q3 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q4 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.88 |
| q5 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.89 |
| q6 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q8 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.91 |
