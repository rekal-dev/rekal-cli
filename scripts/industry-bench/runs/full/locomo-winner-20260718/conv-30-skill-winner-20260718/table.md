# smoke conv-30

- questions: 105
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 0.95
- evidence@10 (answerable): 0.96
- retrieved_context_tokens mean: 152.1
- answer_path_tokens mean: 180.0

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | temporal | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q2 | temporal | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q3 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.51 |
| q4 | multi-hop | FALLBACK_STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.51 |
| q5 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.56 |
| q6 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q7 | temporal | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q8 | temporal | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q9 | temporal | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.59 |
| q10 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.44 |
| q11 | temporal | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q12 | temporal | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.63 |
| q13 | temporal | INJECT | 1 | 1 | 0 | hit | 0.59 |
| q14 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q15 | temporal | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q16 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.82 |
| q17 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.52 |
| q18 | multi-hop | INJECT | 0 | 1 | 5 | deep_rank_lt10 | 0.6 |
| q19 | multi-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.52 |
| q20 | temporal | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q21 | temporal | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q22 | temporal | INJECT | 1 | 1 | 0 | hit | 0.86 |
| q23 | temporal | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q24 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.58 |
| q25 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q26 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.62 |
| q27 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q28 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q29 | temporal | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q30 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.59 |
| q31 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q32 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q33 | temporal | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q34 | temporal | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q35 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.61 |
| q36 | temporal | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q37 | temporal | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q38 | temporal | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q39 | temporal | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q40 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.59 |
| q41 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.6 |
| q42 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.61 |
| q43 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q44 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.54 |
| q45 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.64 |
| q46 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.57 |
| q47 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q48 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.52 |
| q49 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q50 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q51 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.73 |
| q52 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q53 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q54 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q55 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.51 |
| q56 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.63 |
| q57 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q58 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q59 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q60 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q61 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.57 |
| q62 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.61 |
| q63 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q64 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.47 |
| q65 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q66 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q67 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.6 |
| q68 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q69 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.53 |
| q70 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q71 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q72 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q73 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q74 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q75 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q76 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q77 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q78 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q79 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q80 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q81 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q82 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q83 | adversarial | INJECT | 0 | 0 |  | abstention | 0.58 |
| q84 | adversarial | INJECT | 0 | 0 |  | abstention | 0.6 |
| q85 | adversarial | INJECT | 0 | 0 |  | abstention | 0.79 |
| q86 | adversarial | INJECT | 0 | 0 |  | abstention | 0.59 |
| q87 | adversarial | INJECT | 0 | 0 |  | abstention | 0.57 |
| q88 | adversarial | INJECT | 0 | 0 |  | abstention | 0.52 |
| q89 | adversarial | INJECT | 0 | 0 |  | abstention | 0.59 |
| q90 | adversarial | INJECT | 0 | 0 |  | abstention | 0.67 |
| q91 | adversarial | INJECT | 0 | 0 |  | abstention | 0.81 |
| q92 | adversarial | INJECT | 0 | 0 |  | abstention | 0.51 |
| q93 | adversarial | INJECT | 0 | 0 |  | abstention | 0.59 |
| q94 | adversarial | INJECT | 0 | 0 |  | abstention | 0.85 |
| q95 | adversarial | INJECT | 0 | 0 |  | abstention | 0.68 |
| q96 | adversarial | INJECT | 0 | 0 |  | abstention | 0.61 |
| q97 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.47 |
| q98 | adversarial | INJECT | 0 | 0 |  | abstention | 0.62 |
| q99 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q100 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q101 | adversarial | INJECT | 0 | 0 |  | abstention | 0.8 |
| q102 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.5 |
| q103 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q104 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q105 | adversarial | INJECT | 0 | 0 |  | abstention | 0.66 |
