# smoke conv-44

- questions: 158
- route: stock
- calibration: stock
- evidence@5 (answerable): 0.87
- evidence@10 (answerable): 0.92
- retrieved_context_tokens mean: 238.1
- answer_path_tokens mean: 271.0

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.71 |
| q2 | temporal | STOCK | 1 | 1 | 0 | hit | 0.91 |
| q3 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q4 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q5 | temporal | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q6 | temporal | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q7 | temporal | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q8 | temporal | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q9 | temporal | STOCK | 0 | 0 |  | true_miss | 0.69 |
| q10 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q11 | temporal | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q12 | temporal | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q13 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q14 | temporal | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q15 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q16 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q17 | temporal | STOCK | 0 | 0 |  | true_miss | 0.73 |
| q18 | multi-hop | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.68 |
| q19 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q20 | open-domain | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.75 |
| q21 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q22 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q23 | temporal | STOCK | 0 | 0 |  | true_miss | 0.56 |
| q24 | multi-hop | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.63 |
| q25 | multi-hop | STOCK | 0 | 1 | 6 | deep_rank_lt10 | 0.74 |
| q26 | temporal | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q27 | multi-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.69 |
| q28 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q29 | multi-hop | STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.67 |
| q30 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q31 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q32 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.73 |
| q33 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q34 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q35 | temporal | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q36 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q37 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q38 | temporal | STOCK | 1 | 1 | 0 | hit | 0.6 |
| q39 | temporal | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q40 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.73 |
| q41 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q42 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q43 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q44 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.67 |
| q45 | open-domain | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q46 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.68 |
| q47 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q48 | temporal | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q49 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.74 |
| q50 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.75 |
| q51 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q52 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q53 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q54 | open-domain | STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.72 |
| q55 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q56 | temporal | STOCK | 0 | 0 |  | true_miss | 0.74 |
| q57 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q58 | temporal | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q59 | temporal | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q60 | temporal | STOCK | 0 | 0 |  | true_miss | 0.71 |
| q61 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q62 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q63 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q64 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q65 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.73 |
| q66 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q67 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q68 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q69 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q70 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q71 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q72 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q73 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q74 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.73 |
| q75 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q76 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q77 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q78 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q79 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q80 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q81 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.86 |
| q82 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q83 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.69 |
| q84 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q85 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.75 |
| q86 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q87 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q88 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q89 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q90 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q91 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q92 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.86 |
| q93 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q94 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q95 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.7 |
| q96 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q97 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q98 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.89 |
| q99 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q100 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q101 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q102 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q103 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q104 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q105 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q106 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q107 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.64 |
| q108 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q109 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q110 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.77 |
| q111 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q112 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q113 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q114 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q115 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q116 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.57 |
| q117 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q118 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.73 |
| q119 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q120 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q121 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q122 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q123 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q124 | adversarial | STOCK | 0 | 0 |  | abstention | 0.68 |
| q125 | adversarial | STOCK | 0 | 0 |  | abstention | 0.75 |
| q126 | adversarial | STOCK | 0 | 0 |  | abstention | 0.78 |
| q127 | adversarial | STOCK | 0 | 0 |  | abstention | 0.66 |
| q128 | adversarial | STOCK | 0 | 0 |  | abstention | 0.8 |
| q129 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q130 | adversarial | STOCK | 0 | 0 |  | abstention | 0.73 |
| q131 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q132 | adversarial | STOCK | 0 | 0 |  | abstention | 0.81 |
| q133 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
| q134 | adversarial | STOCK | 0 | 0 |  | abstention | 0.66 |
| q135 | adversarial | STOCK | 0 | 0 |  | abstention | 0.68 |
| q136 | adversarial | STOCK | 0 | 0 |  | abstention | 0.78 |
| q137 | adversarial | STOCK | 0 | 0 |  | abstention | 0.76 |
| q138 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q139 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
| q140 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q141 | adversarial | STOCK | 0 | 0 |  | abstention | 0.85 |
| q142 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q143 | adversarial | STOCK | 0 | 0 |  | abstention | 0.75 |
| q144 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q145 | adversarial | STOCK | 0 | 0 |  | abstention | 0.66 |
| q146 | adversarial | STOCK | 0 | 0 |  | abstention | 0.86 |
| q147 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q148 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q149 | adversarial | STOCK | 0 | 0 |  | abstention | 0.78 |
| q150 | adversarial | STOCK | 0 | 0 |  | abstention | 0.78 |
| q151 | adversarial | STOCK | 0 | 0 |  | abstention | 0.6 |
| q152 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q153 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
| q154 | adversarial | STOCK | 0 | 0 |  | abstention | 0.75 |
| q155 | adversarial | STOCK | 0 | 0 |  | abstention | 0.73 |
| q156 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q157 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q158 | adversarial | STOCK | 0 | 0 |  | abstention | 0.79 |
