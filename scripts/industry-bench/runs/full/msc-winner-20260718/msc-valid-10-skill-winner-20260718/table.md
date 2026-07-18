# smoke msc-valid-10

- questions: 10
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 106.4
- answer_path_tokens mean: 121.7

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q2 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.92 |
| q3 | persona-fact | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.53 |
| q4 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.91 |
| q5 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q6 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q7 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.61 |
| q8 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q9 | persona-fact | INJECT | 1 | 1 | 0 | hit | 0.61 |
| q10 | persona-fact | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.53 |
