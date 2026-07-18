# smoke msc-valid-642

- questions: 9
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 103.9
- answer_path_tokens mean: 121.3

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q2 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.93 |
| q3 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q4 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.79 |
| q5 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.93 |
| q6 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q7 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.61 |
| q8 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.93 |
| q9 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.65 |
