# smoke conv-26

- questions: 199
- route: stock
- calibration: stock
- evidence@5 (answerable): 0.84
- evidence@10 (answerable): 0.89
- retrieved_context_tokens mean: 178.4
- answer_path_tokens mean: 209.0

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | temporal | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q2 | temporal | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q3 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.5 |
| q4 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.78 |
| q5 | multi-hop | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.53 |
| q6 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q7 | temporal | STOCK | 1 | 1 | 0 | hit | 0.57 |
| q8 | multi-hop | STOCK | 0 | 1 | 9 | deep_rank_lt10 | 0.71 |
| q9 | temporal | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.43 |
| q10 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.52 |
| q11 | temporal | STOCK | 0 | 0 |  | true_miss | 0.5 |
| q12 | multi-hop | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.47 |
| q13 | temporal | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q14 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.48 |
| q15 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q16 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q17 | temporal | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q18 | temporal | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q19 | multi-hop | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.51 |
| q20 | multi-hop | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.61 |
| q21 | temporal | STOCK | 1 | 1 | 0 | hit | 0.41 |
| q22 | temporal | STOCK | 1 | 1 | 0 | hit | 0.39 |
| q23 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q24 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q25 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q26 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q27 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.69 |
| q28 | open-domain | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q29 | temporal | STOCK | 1 | 1 | 0 | hit | 0.55 |
| q30 | temporal | STOCK | 1 | 1 | 0 | hit | 0.52 |
| q31 | open-domain | STOCK | 0 | 0 |  | abstention | 0.54 |
| q32 | temporal | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.4 |
| q33 | multi-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.57 |
| q34 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.52 |
| q35 | multi-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.55 |
| q36 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.4 |
| q37 | temporal | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q38 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.51 |
| q39 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.56 |
| q40 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.55 |
| q41 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.62 |
| q42 | temporal | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q43 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.74 |
| q44 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.46 |
| q45 | temporal | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q46 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.62 |
| q47 | open-domain | STOCK | 0 | 0 |  | abstention | 0.52 |
| q48 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.5 |
| q49 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.61 |
| q50 | temporal | STOCK | 0 | 0 |  | true_miss | 0.49 |
| q51 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.71 |
| q52 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q53 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q54 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.68 |
| q55 | temporal | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.46 |
| q56 | multi-hop | STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.48 |
| q57 | multi-hop | STOCK | 0 | 1 | 6 | deep_rank_lt10 | 0.55 |
| q58 | temporal | STOCK | 0 | 0 |  | true_miss | 0.65 |
| q59 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.81 |
| q60 | open-domain | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q61 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q62 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q63 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.68 |
| q64 | temporal | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q65 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q66 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.58 |
| q67 | multi-hop | STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.62 |
| q68 | temporal | STOCK | 0 | 0 |  | true_miss | 0.55 |
| q69 | temporal | STOCK | 1 | 1 | 0 | hit | 0.52 |
| q70 | open-domain | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q71 | multi-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.55 |
| q72 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q73 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.53 |
| q74 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q75 | temporal | STOCK | 1 | 1 | 0 | hit | 0.47 |
| q76 | multi-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.56 |
| q77 | multi-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.64 |
| q78 | open-domain | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.53 |
| q79 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q80 | temporal | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q81 | temporal | STOCK | 1 | 1 | 0 | hit | 0.41 |
| q82 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q83 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q84 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q85 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q86 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q87 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.6 |
| q88 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q89 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q90 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.56 |
| q91 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q92 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q93 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q94 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q95 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q96 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.47 |
| q97 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q98 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q99 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q100 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q101 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q102 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q103 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q104 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q105 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q106 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q107 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q108 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.51 |
| q109 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q110 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q111 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q112 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q113 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.51 |
| q114 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q115 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.55 |
| q116 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.53 |
| q117 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q118 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q119 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.68 |
| q120 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q121 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.57 |
| q122 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q123 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q124 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q125 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q126 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q127 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q128 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.71 |
| q129 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.55 |
| q130 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q131 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.58 |
| q132 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q133 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q134 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.47 |
| q135 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q136 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.55 |
| q137 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q138 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.62 |
| q139 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.52 |
| q140 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q141 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q142 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.52 |
| q143 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.55 |
| q144 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q145 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q146 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.7 |
| q147 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q148 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.74 |
| q149 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q150 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.57 |
| q151 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.65 |
| q152 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q153 | adversarial | STOCK | 0 | 0 |  | abstention | 0.71 |
| q154 | adversarial | STOCK | 0 | 0 |  | abstention | 0.65 |
| q155 | adversarial | STOCK | 0 | 0 |  | abstention | 0.57 |
| q156 | adversarial | STOCK | 0 | 0 |  | abstention | 0.71 |
| q157 | adversarial | STOCK | 0 | 0 |  | abstention | 0.76 |
| q158 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q159 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q160 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q161 | adversarial | STOCK | 0 | 0 |  | abstention | 0.65 |
| q162 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q163 | adversarial | STOCK | 0 | 0 |  | abstention | 0.46 |
| q164 | adversarial | STOCK | 0 | 0 |  | abstention | 0.92 |
| q165 | adversarial | STOCK | 0 | 0 |  | abstention | 0.76 |
| q166 | adversarial | STOCK | 0 | 0 |  | abstention | 0.77 |
| q167 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
| q168 | adversarial | STOCK | 0 | 0 |  | abstention | 0.85 |
| q169 | adversarial | STOCK | 0 | 0 |  | abstention | 0.68 |
| q170 | adversarial | STOCK | 0 | 0 |  | abstention | 0.51 |
| q171 | adversarial | STOCK | 0 | 0 |  | abstention | 0.61 |
| q172 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q173 | adversarial | STOCK | 0 | 0 |  | abstention | 0.6 |
| q174 | adversarial | STOCK | 0 | 0 |  | abstention | 0.6 |
| q175 | adversarial | STOCK | 0 | 0 |  | abstention | 0.66 |
| q176 | adversarial | STOCK | 0 | 0 |  | abstention | 0.68 |
| q177 | adversarial | STOCK | 0 | 0 |  | abstention | 0.65 |
| q178 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q179 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q180 | adversarial | STOCK | 0 | 0 |  | abstention | 0.64 |
| q181 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q182 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q183 | adversarial | STOCK | 0 | 0 |  | abstention | 0.55 |
| q184 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q185 | adversarial | STOCK | 0 | 0 |  | abstention | 0.73 |
| q186 | adversarial | STOCK | 0 | 0 |  | abstention | 0.56 |
| q187 | adversarial | STOCK | 0 | 0 |  | abstention | 0.77 |
| q188 | adversarial | STOCK | 0 | 0 |  | abstention | 0.52 |
| q189 | adversarial | STOCK | 0 | 0 |  | abstention | 0.58 |
| q190 | adversarial | STOCK | 0 | 0 |  | abstention | 0.62 |
| q191 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q192 | adversarial | STOCK | 0 | 0 |  | abstention | 0.71 |
| q193 | adversarial | STOCK | 0 | 0 |  | abstention | 0.71 |
| q194 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q195 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
| q196 | adversarial | STOCK | 0 | 0 |  | abstention | 0.76 |
| q197 | adversarial | STOCK | 0 | 0 |  | abstention | 0.83 |
| q198 | adversarial | STOCK | 0 | 0 |  | abstention | 0.79 |
| q199 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
