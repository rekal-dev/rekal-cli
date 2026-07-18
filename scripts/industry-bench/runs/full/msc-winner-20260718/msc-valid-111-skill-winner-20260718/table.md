# smoke msc-valid-111

- questions: 9
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 108.8
- answer_path_tokens mean: 124.6

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.92 |
| q2 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q3 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.93 |
| q4 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.92 |
| q5 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.92 |
| q6 | persona-fact | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.77 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.94 |
| q8 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.92 |
| q9 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.91 |
