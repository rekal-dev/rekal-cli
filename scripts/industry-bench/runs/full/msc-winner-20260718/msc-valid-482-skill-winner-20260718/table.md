# smoke msc-valid-482

- questions: 9
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 108.9
- answer_path_tokens mean: 128.9

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q2 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.74 |
| q3 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.94 |
| q4 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q5 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.93 |
| q6 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q8 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q9 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.87 |
