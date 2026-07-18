# smoke msc-valid-452

- questions: 8
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 127.1
- answer_path_tokens mean: 141.5

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q2 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q3 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q4 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.58 |
| q5 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q6 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.94 |
| q8 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
