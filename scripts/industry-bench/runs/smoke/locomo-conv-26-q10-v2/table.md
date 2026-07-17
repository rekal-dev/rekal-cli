# smoke conv-26

- questions: 10
- route: stock
- calibration: stock
- evidence@5 (answerable): 0.80
- evidence@10 (answerable): 1.00
- retrieved_context_tokens mean: 226.5
- answer_path_tokens mean: 255.7

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | temporal | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q2 | temporal | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q3 | open-domain | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.65 |
| q4 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.78 |
| q5 | multi-hop | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.57 |
| q6 | temporal | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q7 | temporal | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q8 | multi-hop | STOCK | 0 | 1 | 8 | deep_rank_lt10 | 0.58 |
| q9 | temporal | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q10 | temporal | STOCK | 1 | 1 | 0 | hit | 0.72 |
