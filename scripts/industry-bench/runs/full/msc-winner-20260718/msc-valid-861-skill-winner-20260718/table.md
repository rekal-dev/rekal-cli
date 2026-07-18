# smoke msc-valid-861

- questions: 9
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 76.0
- answer_path_tokens mean: 92.1

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.86 |
| q2 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q3 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q4 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q5 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q6 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.86 |
| q7 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q8 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q9 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.83 |
