# smoke conv-47

- questions: 190
- route: stock
- calibration: stock
- evidence@5 (answerable): 0.91
- evidence@10 (answerable): 0.95
- retrieved_context_tokens mean: 460.7
- answer_path_tokens mean: 488.2

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.45 |
| q2 | temporal | STOCK | 0 | 0 |  | true_miss | 0.58 |
| q3 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.63 |
| q4 | multi-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.46 |
| q5 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q6 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.55 |
| q7 | open-domain | STOCK | 0 | 1 | 6 | deep_rank_lt10 | 0.65 |
| q8 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q9 | multi-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.44 |
| q10 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.54 |
| q11 | temporal | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q12 | temporal | STOCK | 0 | 0 | 10 | deep_rank_gte10 | 0.65 |
| q13 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.54 |
| q14 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.6 |
| q15 | temporal | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.53 |
| q16 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.61 |
| q17 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q18 | open-domain | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.57 |
| q19 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q20 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.53 |
| q21 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q22 | temporal | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q23 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q24 | temporal | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q25 | multi-hop | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.52 |
| q26 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q27 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q28 | multi-hop | STOCK | 0 | 1 | 6 | deep_rank_lt10 | 0.43 |
| q29 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q30 | temporal | STOCK | 0 | 0 | 14 | deep_rank_gte10 | 0.62 |
| q31 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q32 | temporal | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.55 |
| q33 | temporal | STOCK | 0 | 1 | 8 | deep_rank_lt10 | 0.49 |
| q34 | open-domain | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.74 |
| q35 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q36 | open-domain | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.73 |
| q37 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.58 |
| q38 | temporal | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.64 |
| q39 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.55 |
| q40 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q41 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.54 |
| q42 | temporal | STOCK | 1 | 1 | 0 | hit | 0.53 |
| q43 | temporal | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q44 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q45 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.55 |
| q46 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q47 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q48 | temporal | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q49 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q50 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q51 | temporal | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q52 | multi-hop | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.64 |
| q53 | temporal | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q54 | temporal | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q55 | multi-hop | STOCK | 0 | 0 | 13 | deep_rank_gte10 | 0.53 |
| q56 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.72 |
| q57 | multi-hop | STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.65 |
| q58 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.53 |
| q59 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.76 |
| q60 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.78 |
| q61 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q62 | temporal | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q63 | temporal | STOCK | 1 | 1 | 0 | hit | 0.88 |
| q64 | temporal | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q65 | temporal | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q66 | temporal | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q67 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.55 |
| q68 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.48 |
| q69 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q70 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q71 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q72 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.58 |
| q73 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.42 |
| q74 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q75 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q76 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q77 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q78 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.89 |
| q79 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q80 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q81 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q82 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q83 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q84 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.83 |
| q85 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q86 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.47 |
| q87 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q88 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q89 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.46 |
| q90 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q91 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q92 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q93 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q94 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q95 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.72 |
| q96 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q97 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q98 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q99 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.43 |
| q100 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q101 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q102 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.59 |
| q103 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q104 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q105 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q106 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q107 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.6 |
| q108 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q109 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q110 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.48 |
| q111 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q112 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.62 |
| q113 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.58 |
| q114 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q115 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q116 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q117 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.55 |
| q118 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q119 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q120 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q121 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q122 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q123 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q124 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q125 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q126 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q127 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.51 |
| q128 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.56 |
| q129 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.48 |
| q130 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.49 |
| q131 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q132 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q133 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q134 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.61 |
| q135 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q136 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q137 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q138 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q139 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q140 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q141 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q142 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q143 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.59 |
| q144 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q145 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q146 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q147 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q148 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q149 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.55 |
| q150 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.55 |
| q151 | adversarial | STOCK | 0 | 0 |  | abstention | 0.73 |
| q152 | adversarial | STOCK | 0 | 0 |  | abstention | 0.68 |
| q153 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q154 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
| q155 | adversarial | STOCK | 0 | 0 |  | abstention | 0.77 |
| q156 | adversarial | STOCK | 0 | 0 |  | abstention | 0.79 |
| q157 | adversarial | STOCK | 0 | 0 |  | abstention | 0.9 |
| q158 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q159 | adversarial | STOCK | 0 | 0 |  | abstention | 0.78 |
| q160 | adversarial | STOCK | 0 | 0 |  | abstention | 0.6 |
| q161 | adversarial | STOCK | 0 | 0 |  | abstention | 0.47 |
| q162 | adversarial | STOCK | 0 | 0 |  | abstention | 0.71 |
| q163 | adversarial | STOCK | 0 | 0 |  | abstention | 0.59 |
| q164 | adversarial | STOCK | 0 | 0 |  | abstention | 0.65 |
| q165 | adversarial | STOCK | 0 | 0 |  | abstention | 0.75 |
| q166 | adversarial | STOCK | 0 | 0 |  | abstention | 0.66 |
| q167 | adversarial | STOCK | 0 | 0 |  | abstention | 0.68 |
| q168 | adversarial | STOCK | 0 | 0 |  | abstention | 0.48 |
| q169 | adversarial | STOCK | 0 | 0 |  | abstention | 0.51 |
| q170 | adversarial | STOCK | 0 | 0 |  | abstention | 0.68 |
| q171 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q172 | adversarial | STOCK | 0 | 0 |  | abstention | 0.58 |
| q173 | adversarial | STOCK | 0 | 0 |  | abstention | 0.8 |
| q174 | adversarial | STOCK | 0 | 0 |  | abstention | 0.62 |
| q175 | adversarial | STOCK | 0 | 0 |  | abstention | 0.83 |
| q176 | adversarial | STOCK | 0 | 0 |  | abstention | 0.48 |
| q177 | adversarial | STOCK | 0 | 0 |  | abstention | 0.56 |
| q178 | adversarial | STOCK | 0 | 0 |  | abstention | 0.71 |
| q179 | adversarial | STOCK | 0 | 0 |  | abstention | 0.89 |
| q180 | adversarial | STOCK | 0 | 0 |  | abstention | 0.79 |
| q181 | adversarial | STOCK | 0 | 0 |  | abstention | 0.84 |
| q182 | adversarial | STOCK | 0 | 0 |  | abstention | 0.6 |
| q183 | adversarial | STOCK | 0 | 0 |  | abstention | 0.66 |
| q184 | adversarial | STOCK | 0 | 0 |  | abstention | 0.59 |
| q185 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q186 | adversarial | STOCK | 0 | 0 |  | abstention | 0.75 |
| q187 | adversarial | STOCK | 0 | 0 |  | abstention | 0.84 |
| q188 | adversarial | STOCK | 0 | 0 |  | abstention | 0.66 |
| q189 | adversarial | STOCK | 0 | 0 |  | abstention | 0.53 |
| q190 | adversarial | STOCK | 0 | 0 |  | abstention | 0.55 |
