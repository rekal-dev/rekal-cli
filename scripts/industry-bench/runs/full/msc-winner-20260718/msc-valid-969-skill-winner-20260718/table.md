# smoke msc-valid-969

- questions: 9
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 90.3
- answer_path_tokens mean: 104.3

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.92 |
| q2 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.97 |
| q3 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.6 |
| q4 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.94 |
| q5 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q6 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q8 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q9 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.92 |
