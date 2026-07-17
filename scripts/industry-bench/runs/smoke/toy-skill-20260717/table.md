# smoke toy-001

- questions: 4
- route: skill
- calibration: chat-provisional.json
- evidence@5 (answerable): 1.00
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 61.2
- answer_path_tokens mean: 77.5

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q2 | temporal | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q3 | knowledge-update | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q4 | abstention | SILENCE | 0 | 0 |  | abstention | None |
