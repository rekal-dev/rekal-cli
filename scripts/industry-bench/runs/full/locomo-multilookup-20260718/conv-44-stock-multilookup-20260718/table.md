# smoke conv-44

- questions: 158
- route: stock
- calibration: stock
- evidence@5 (answerable): 0.85
- evidence@10 (answerable): 0.92
- retrieved_context_tokens mean: 551.7
- answer_path_tokens mean: 584.1

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.47 |
| q2 | temporal | STOCK | 1 | 1 | 0 | hit | 0.91 |
| q3 | multi-hop | STOCK | 0 | 1 | 6 | deep_rank_lt10 | 0.45 |
| q4 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q5 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.61 |
| q6 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.61 |
| q7 | temporal | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q8 | temporal | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q9 | temporal | STOCK | 0 | 0 |  | true_miss | 0.61 |
| q10 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q11 | temporal | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q12 | temporal | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q13 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q14 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.42 |
| q15 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q16 | multi-hop | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.54 |
| q17 | temporal | STOCK | 0 | 1 | 6 | deep_rank_lt10 | 0.5 |
| q18 | multi-hop | STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.46 |
| q19 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q20 | open-domain | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.58 |
| q21 | open-domain | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.42 |
| q22 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q23 | temporal | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.41 |
| q24 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.5 |
| q25 | multi-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.49 |
| q26 | temporal | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q27 | multi-hop | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.48 |
| q28 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.49 |
| q29 | multi-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.42 |
| q30 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.59 |
| q31 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.67 |
| q32 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.56 |
| q33 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q34 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q35 | temporal | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q36 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q37 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.6 |
| q38 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.48 |
| q39 | temporal | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q40 | multi-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.44 |
| q41 | multi-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.69 |
| q42 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.55 |
| q43 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.59 |
| q44 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.5 |
| q45 | open-domain | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q46 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.4 |
| q47 | temporal | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.4 |
| q48 | temporal | STOCK | 1 | 1 | 0 | hit | 0.56 |
| q49 | multi-hop | STOCK | 0 | 0 | 10 | deep_rank_gte10 | 0.56 |
| q50 | multi-hop | STOCK | 0 | 0 | 11 | deep_rank_gte10 | 0.56 |
| q51 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q52 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.46 |
| q53 | open-domain | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.58 |
| q54 | open-domain | STOCK | 0 | 0 | 18 | deep_rank_gte10 | 0.6 |
| q55 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.54 |
| q56 | temporal | STOCK | 0 | 0 |  | true_miss | 0.56 |
| q57 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q58 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.56 |
| q59 | temporal | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q60 | temporal | STOCK | 0 | 0 | 18 | deep_rank_gte10 | 0.49 |
| q61 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.46 |
| q62 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q63 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q64 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q65 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q66 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.56 |
| q67 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.59 |
| q68 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q69 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q70 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q71 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.57 |
| q72 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q73 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q74 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.55 |
| q75 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q76 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q77 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q78 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q79 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.66 |
| q80 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.53 |
| q81 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.53 |
| q82 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q83 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.59 |
| q84 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q85 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.59 |
| q86 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.57 |
| q87 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.5 |
| q88 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.61 |
| q89 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q90 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q91 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q92 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.86 |
| q93 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.6 |
| q94 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.53 |
| q95 | single-hop | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.47 |
| q96 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q97 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.58 |
| q98 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.61 |
| q99 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.48 |
| q100 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.51 |
| q101 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q102 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.62 |
| q103 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q104 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.61 |
| q105 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q106 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.5 |
| q107 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q108 | single-hop | STOCK | 0 | 0 | 16 | deep_rank_gte10 | 0.5 |
| q109 | single-hop | STOCK | 0 | 0 | 17 | deep_rank_gte10 | 0.5 |
| q110 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.49 |
| q111 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q112 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q113 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q114 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q115 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q116 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.53 |
| q117 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.46 |
| q118 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.6 |
| q119 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q120 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q121 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q122 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q123 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q124 | adversarial | STOCK | 0 | 0 |  | abstention | 0.68 |
| q125 | adversarial | STOCK | 0 | 0 |  | abstention | 0.64 |
| q126 | adversarial | STOCK | 0 | 0 |  | abstention | 0.58 |
| q127 | adversarial | STOCK | 0 | 0 |  | abstention | 0.49 |
| q128 | adversarial | STOCK | 0 | 0 |  | abstention | 0.8 |
| q129 | adversarial | STOCK | 0 | 0 |  | abstention | 0.57 |
| q130 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q131 | adversarial | STOCK | 0 | 0 |  | abstention | 0.55 |
| q132 | adversarial | STOCK | 0 | 0 |  | abstention | 0.81 |
| q133 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q134 | adversarial | STOCK | 0 | 0 |  | abstention | 0.66 |
| q135 | adversarial | STOCK | 0 | 0 |  | abstention | 0.59 |
| q136 | adversarial | STOCK | 0 | 0 |  | abstention | 0.78 |
| q137 | adversarial | STOCK | 0 | 0 |  | abstention | 0.59 |
| q138 | adversarial | STOCK | 0 | 0 |  | abstention | 0.55 |
| q139 | adversarial | STOCK | 0 | 0 |  | abstention | 0.64 |
| q140 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q141 | adversarial | STOCK | 0 | 0 |  | abstention | 0.85 |
| q142 | adversarial | STOCK | 0 | 0 |  | abstention | 0.6 |
| q143 | adversarial | STOCK | 0 | 0 |  | abstention | 0.54 |
| q144 | adversarial | STOCK | 0 | 0 |  | abstention | 0.5 |
| q145 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
| q146 | adversarial | STOCK | 0 | 0 |  | abstention | 0.58 |
| q147 | adversarial | STOCK | 0 | 0 |  | abstention | 0.62 |
| q148 | adversarial | STOCK | 0 | 0 |  | abstention | 0.62 |
| q149 | adversarial | STOCK | 0 | 0 |  | abstention | 0.78 |
| q150 | adversarial | STOCK | 0 | 0 |  | abstention | 0.5 |
| q151 | adversarial | STOCK | 0 | 0 |  | abstention | 0.66 |
| q152 | adversarial | STOCK | 0 | 0 |  | abstention | 0.5 |
| q153 | adversarial | STOCK | 0 | 0 |  | abstention | 0.5 |
| q154 | adversarial | STOCK | 0 | 0 |  | abstention | 0.49 |
| q155 | adversarial | STOCK | 0 | 0 |  | abstention | 0.66 |
| q156 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q157 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q158 | adversarial | STOCK | 0 | 0 |  | abstention | 0.66 |
