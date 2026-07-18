# smoke conv-50

- questions: 204
- route: stock
- calibration: stock
- evidence@5 (answerable): 0.85
- evidence@10 (answerable): 0.87
- retrieved_context_tokens mean: 207.9
- answer_path_tokens mean: 243.0

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | temporal | STOCK | 1 | 1 | 0 | hit | 0.56 |
| q2 | multi-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.62 |
| q3 | temporal | STOCK | 1 | 1 | 0 | hit | 0.56 |
| q4 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.52 |
| q5 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.45 |
| q6 | multi-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.36 |
| q7 | multi-hop | STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.62 |
| q8 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.6 |
| q9 | temporal | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q10 | temporal | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q11 | temporal | STOCK | 1 | 1 | 0 | hit | 0.58 |
| q12 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.62 |
| q13 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.57 |
| q14 | open-domain | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.44 |
| q15 | temporal | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q16 | multi-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.56 |
| q17 | temporal | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q18 | temporal | STOCK | 0 | 0 |  | true_miss | 0.59 |
| q19 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q20 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.59 |
| q21 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.58 |
| q22 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.44 |
| q23 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q24 | multi-hop | STOCK | 0 | 1 | 6 | deep_rank_lt10 | 0.41 |
| q25 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.5 |
| q26 | temporal | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q27 | temporal | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q28 | temporal | STOCK | 0 | 0 |  | true_miss | 0.65 |
| q29 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.43 |
| q30 | multi-hop | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.63 |
| q31 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.49 |
| q32 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.39 |
| q33 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q34 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q35 | temporal | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q36 | temporal | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q37 | temporal | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q38 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.52 |
| q39 | temporal | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q40 | open-domain | STOCK | 0 | 0 |  | abstention | 0.46 |
| q41 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.57 |
| q42 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.55 |
| q43 | open-domain | STOCK | 0 | 0 |  | abstention | 0.57 |
| q44 | temporal | STOCK | 0 | 0 |  | true_miss | 0.55 |
| q45 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.57 |
| q46 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.68 |
| q47 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.4 |
| q48 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.57 |
| q49 | temporal | STOCK | 0 | 0 |  | true_miss | 0.6 |
| q50 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.37 |
| q51 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q52 | temporal | STOCK | 1 | 1 | 0 | hit | 0.59 |
| q53 | temporal | STOCK | 0 | 0 |  | true_miss | 0.54 |
| q54 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.49 |
| q55 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q56 | temporal | STOCK | 1 | 1 | 0 | hit | 0.91 |
| q57 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.65 |
| q58 | temporal | STOCK | 0 | 0 |  | true_miss | 0.6 |
| q59 | multi-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.59 |
| q60 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.41 |
| q61 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.51 |
| q62 | temporal | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q63 | temporal | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q64 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q65 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.56 |
| q66 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q67 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.63 |
| q68 | temporal | STOCK | 0 | 0 |  | true_miss | 0.47 |
| q69 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.54 |
| q70 | temporal | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q71 | temporal | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.52 |
| q72 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q73 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.55 |
| q74 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.49 |
| q75 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q76 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.59 |
| q77 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q78 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q79 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.38 |
| q80 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.59 |
| q81 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q82 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.48 |
| q83 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.42 |
| q84 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q85 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q86 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.5 |
| q87 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.62 |
| q88 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.55 |
| q89 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q90 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.6 |
| q91 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.89 |
| q92 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.54 |
| q93 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q94 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.55 |
| q95 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q96 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q97 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.56 |
| q98 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q99 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q100 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q101 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q102 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.59 |
| q103 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.86 |
| q104 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q105 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q106 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.58 |
| q107 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q108 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q109 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q110 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.5 |
| q111 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.57 |
| q112 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q113 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.65 |
| q114 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.6 |
| q115 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q116 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.49 |
| q117 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q118 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.6 |
| q119 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.63 |
| q120 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.39 |
| q121 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.45 |
| q122 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.53 |
| q123 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.46 |
| q124 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.53 |
| q125 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.63 |
| q126 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.72 |
| q127 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.46 |
| q128 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q129 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.55 |
| q130 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q131 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q132 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.52 |
| q133 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q134 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.47 |
| q135 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.88 |
| q136 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.53 |
| q137 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.64 |
| q138 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.55 |
| q139 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q140 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.48 |
| q141 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.68 |
| q142 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.58 |
| q143 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.49 |
| q144 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.5 |
| q145 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.58 |
| q146 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q147 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.59 |
| q148 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q149 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q150 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q151 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.41 |
| q152 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.41 |
| q153 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.45 |
| q154 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.74 |
| q155 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.6 |
| q156 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.64 |
| q157 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.43 |
| q158 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.64 |
| q159 | adversarial | STOCK | 0 | 0 |  | abstention | 0.46 |
| q160 | adversarial | STOCK | 0 | 0 |  | abstention | 0.62 |
| q161 | adversarial | STOCK | 0 | 0 |  | abstention | 0.55 |
| q162 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q163 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q164 | adversarial | STOCK | 0 | 0 |  | abstention | 0.39 |
| q165 | adversarial | STOCK | 0 | 0 |  | abstention | 0.41 |
| q166 | adversarial | STOCK | 0 | 0 |  | abstention | 0.81 |
| q167 | adversarial | STOCK | 0 | 0 |  | abstention | 0.62 |
| q168 | adversarial | STOCK | 0 | 0 |  | abstention | 0.81 |
| q169 | adversarial | STOCK | 0 | 0 |  | abstention | 0.57 |
| q170 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q171 | adversarial | STOCK | 0 | 0 |  | abstention | 0.78 |
| q172 | adversarial | STOCK | 0 | 0 |  | abstention | 0.55 |
| q173 | adversarial | STOCK | 0 | 0 |  | abstention | 0.6 |
| q174 | adversarial | STOCK | 0 | 0 |  | abstention | 0.86 |
| q175 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q176 | adversarial | STOCK | 0 | 0 |  | abstention | 0.85 |
| q177 | adversarial | STOCK | 0 | 0 |  | abstention | 0.58 |
| q178 | adversarial | STOCK | 0 | 0 |  | abstention | 0.77 |
| q179 | adversarial | STOCK | 0 | 0 |  | abstention | 0.78 |
| q180 | adversarial | STOCK | 0 | 0 |  | abstention | 0.79 |
| q181 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q182 | adversarial | STOCK | 0 | 0 |  | abstention | 0.84 |
| q183 | adversarial | STOCK | 0 | 0 |  | abstention | 0.61 |
| q184 | adversarial | STOCK | 0 | 0 |  | abstention | 0.68 |
| q185 | adversarial | STOCK | 0 | 0 |  | abstention | 0.49 |
| q186 | adversarial | STOCK | 0 | 0 |  | abstention | 0.41 |
| q187 | adversarial | STOCK | 0 | 0 |  | abstention | 0.49 |
| q188 | adversarial | STOCK | 0 | 0 |  | abstention | 0.47 |
| q189 | adversarial | STOCK | 0 | 0 |  | abstention | 0.46 |
| q190 | adversarial | STOCK | 0 | 0 |  | abstention | 0.65 |
| q191 | adversarial | STOCK | 0 | 0 |  | abstention | 0.65 |
| q192 | adversarial | STOCK | 0 | 0 |  | abstention | 0.51 |
| q193 | adversarial | STOCK | 0 | 0 |  | abstention | 0.48 |
| q194 | adversarial | STOCK | 0 | 0 |  | abstention | 0.51 |
| q195 | adversarial | STOCK | 0 | 0 |  | abstention | 0.5 |
| q196 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q197 | adversarial | STOCK | 0 | 0 |  | abstention | 0.47 |
| q198 | adversarial | STOCK | 0 | 0 |  | abstention | 0.79 |
| q199 | adversarial | STOCK | 0 | 0 |  | abstention | 0.6 |
| q200 | adversarial | STOCK | 0 | 0 |  | abstention | 0.41 |
| q201 | adversarial | STOCK | 0 | 0 |  | abstention | 0.46 |
| q202 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
| q203 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q204 | adversarial | STOCK | 0 | 0 |  | abstention | 0.44 |
