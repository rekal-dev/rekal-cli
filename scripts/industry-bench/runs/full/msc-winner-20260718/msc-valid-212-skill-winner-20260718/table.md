# smoke msc-valid-212

- questions: 8
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 99.2
- answer_path_tokens mean: 114.4

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q2 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q3 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.52 |
| q4 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q5 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q6 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q8 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.7 |
