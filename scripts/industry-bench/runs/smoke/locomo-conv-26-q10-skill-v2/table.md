# smoke conv-26

- questions: 10
- route: skill
- calibration: chat-provisional.json
- evidence@5 (answerable): 0.80
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 172.8
- answer_path_tokens mean: 193.3

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | temporal | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q2 | temporal | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q3 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q4 | multi-hop | INJECT | 0 | 1 | 6 | deep_rank_lt10 | 0.78 |
| q5 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q6 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q7 | temporal | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q8 | multi-hop | INJECT | 0 | 1 | 6 | deep_rank_lt10 | 0.71 |
| q9 | temporal | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q10 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.74 |
