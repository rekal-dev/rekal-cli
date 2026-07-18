# smoke msc-valid-999

- questions: 9
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 99.6
- answer_path_tokens mean: 113.8

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.94 |
| q2 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q3 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q4 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q5 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.8 |
| q6 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.79 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.91 |
| q8 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.89 |
| q9 | persona-fact | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.95 |
