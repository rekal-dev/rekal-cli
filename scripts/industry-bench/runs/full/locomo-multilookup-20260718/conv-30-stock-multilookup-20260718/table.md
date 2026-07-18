# smoke conv-30

- questions: 105
- route: stock
- calibration: stock
- evidence@5 (answerable): 0.90
- evidence@10 (answerable): 0.98
- retrieved_context_tokens mean: 451.1
- answer_path_tokens mean: 479.5

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | temporal | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q2 | temporal | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q3 | single-hop | STOCK | 0 | 0 | 11 | deep_rank_gte10 | 0.49 |
| q4 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.51 |
| q5 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.54 |
| q6 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q7 | temporal | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q8 | temporal | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q9 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q10 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.44 |
| q11 | temporal | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q12 | temporal | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.63 |
| q13 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.56 |
| q14 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.71 |
| q15 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q16 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.82 |
| q17 | temporal | STOCK | 0 | 1 | 8 | deep_rank_lt10 | 0.52 |
| q18 | multi-hop | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.6 |
| q19 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.52 |
| q20 | temporal | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q21 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q22 | temporal | STOCK | 0 | 0 | 10 | deep_rank_gte10 | 0.56 |
| q23 | temporal | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q24 | multi-hop | STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.58 |
| q25 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q26 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.62 |
| q27 | temporal | STOCK | 1 | 1 | 0 | hit | 0.46 |
| q28 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.57 |
| q29 | temporal | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q30 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q31 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.58 |
| q32 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q33 | temporal | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q34 | temporal | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q35 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.61 |
| q36 | temporal | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q37 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.6 |
| q38 | temporal | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q39 | temporal | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q40 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.59 |
| q41 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.6 |
| q42 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.52 |
| q43 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q44 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.54 |
| q45 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.64 |
| q46 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.57 |
| q47 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.57 |
| q48 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.52 |
| q49 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q50 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q51 | single-hop | STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.73 |
| q52 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q53 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q54 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q55 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.51 |
| q56 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.63 |
| q57 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q58 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q59 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q60 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.62 |
| q61 | single-hop | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.52 |
| q62 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.42 |
| q63 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q64 | single-hop | STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.47 |
| q65 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q66 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q67 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.6 |
| q68 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q69 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.53 |
| q70 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q71 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q72 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q73 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q74 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q75 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q76 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q77 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q78 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q79 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q80 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q81 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q82 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.62 |
| q83 | adversarial | STOCK | 0 | 0 |  | abstention | 0.55 |
| q84 | adversarial | STOCK | 0 | 0 |  | abstention | 0.5 |
| q85 | adversarial | STOCK | 0 | 0 |  | abstention | 0.79 |
| q86 | adversarial | STOCK | 0 | 0 |  | abstention | 0.59 |
| q87 | adversarial | STOCK | 0 | 0 |  | abstention | 0.57 |
| q88 | adversarial | STOCK | 0 | 0 |  | abstention | 0.52 |
| q89 | adversarial | STOCK | 0 | 0 |  | abstention | 0.59 |
| q90 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q91 | adversarial | STOCK | 0 | 0 |  | abstention | 0.81 |
| q92 | adversarial | STOCK | 0 | 0 |  | abstention | 0.51 |
| q93 | adversarial | STOCK | 0 | 0 |  | abstention | 0.59 |
| q94 | adversarial | STOCK | 0 | 0 |  | abstention | 0.85 |
| q95 | adversarial | STOCK | 0 | 0 |  | abstention | 0.66 |
| q96 | adversarial | STOCK | 0 | 0 |  | abstention | 0.42 |
| q97 | adversarial | STOCK | 0 | 0 |  | abstention | 0.47 |
| q98 | adversarial | STOCK | 0 | 0 |  | abstention | 0.62 |
| q99 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q100 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q101 | adversarial | STOCK | 0 | 0 |  | abstention | 0.8 |
| q102 | adversarial | STOCK | 0 | 0 |  | abstention | 0.49 |
| q103 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
| q104 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q105 | adversarial | STOCK | 0 | 0 |  | abstention | 0.64 |
