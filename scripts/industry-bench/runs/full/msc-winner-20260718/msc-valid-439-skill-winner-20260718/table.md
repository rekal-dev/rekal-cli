# smoke msc-valid-439

- questions: 9
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 99.1
- answer_path_tokens mean: 114.9

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q2 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q3 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q4 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q5 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q6 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.91 |
| q8 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q9 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.68 |
