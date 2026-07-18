# smoke msc-valid-981

- questions: 9
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 73.0
- answer_path_tokens mean: 87.6

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q2 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q3 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q4 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.6 |
| q5 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q6 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q8 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q9 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.77 |
