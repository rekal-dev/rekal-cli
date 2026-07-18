# smoke msc-test-735

- questions: 10
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 87.2
- answer_path_tokens mean: 105.9

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q2 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.74 |
| q3 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q4 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q5 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q6 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q8 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.83 |
| q9 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.87 |
| q10 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
