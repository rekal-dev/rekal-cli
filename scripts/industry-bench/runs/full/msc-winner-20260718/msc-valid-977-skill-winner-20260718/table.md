# smoke msc-valid-977

- questions: 8
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 92.1
- answer_path_tokens mean: 111.5

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q2 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.78 |
| q3 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q4 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.79 |
| q5 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q6 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q8 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.79 |
