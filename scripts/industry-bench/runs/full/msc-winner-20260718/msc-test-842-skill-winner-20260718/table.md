# smoke msc-test-842

- questions: 10
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 90.5
- answer_path_tokens mean: 108.0

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.94 |
| q2 | persona-fact | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.44 |
| q3 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.94 |
| q4 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q5 | persona-fact | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.84 |
| q6 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.88 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q8 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.91 |
| q9 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.94 |
| q10 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.94 |
