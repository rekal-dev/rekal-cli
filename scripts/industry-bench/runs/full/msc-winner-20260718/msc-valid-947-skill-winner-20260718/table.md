# smoke msc-valid-947

- questions: 9
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 115.8
- answer_path_tokens mean: 130.0

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q2 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q3 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q4 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.78 |
| q5 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.74 |
| q6 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q8 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q9 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.74 |
