# smoke conv-41

- questions: 193
- route: stock
- calibration: stock
- evidence@5 (answerable): 0.86
- evidence@10 (answerable): 0.88
- retrieved_context_tokens mean: 187.5
- answer_path_tokens mean: 219.6

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.76 |
| q2 | temporal | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q3 | multi-hop | STOCK | 0 | 1 | 9 | deep_rank_lt10 | 0.78 |
| q4 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.49 |
| q5 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.53 |
| q6 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q7 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.55 |
| q8 | multi-hop | STOCK | 0 | 1 | 9 | deep_rank_lt10 | 0.63 |
| q9 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.65 |
| q10 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.57 |
| q11 | temporal | STOCK | 0 | 0 |  | true_miss | 0.59 |
| q12 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q13 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q14 | temporal | STOCK | 1 | 1 | 0 | hit | 0.54 |
| q15 | open-domain | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.59 |
| q16 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.45 |
| q17 | temporal | STOCK | 1 | 1 | 0 | hit | 0.47 |
| q18 | open-domain | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.65 |
| q19 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.52 |
| q20 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.73 |
| q21 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.61 |
| q22 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.61 |
| q23 | temporal | STOCK | 0 | 0 |  | true_miss | 0.56 |
| q24 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.45 |
| q25 | temporal | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q26 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.51 |
| q27 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q28 | temporal | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q29 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.63 |
| q30 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q31 | multi-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.52 |
| q32 | temporal | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q33 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.39 |
| q34 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.51 |
| q35 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q36 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.56 |
| q37 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q38 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q39 | temporal | STOCK | 1 | 1 | 0 | hit | 0.45 |
| q40 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.45 |
| q41 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.6 |
| q42 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.46 |
| q43 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q44 | temporal | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q45 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.57 |
| q46 | open-domain | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q47 | temporal | STOCK | 1 | 1 | 0 | hit | 0.58 |
| q48 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.47 |
| q49 | temporal | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q50 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q51 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.56 |
| q52 | temporal | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q53 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q54 | temporal | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q55 | temporal | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q56 | temporal | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.62 |
| q57 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q58 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q59 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.56 |
| q60 | temporal | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q61 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.55 |
| q62 | temporal | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q63 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q64 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.49 |
| q65 | open-domain | STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.39 |
| q66 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q67 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q68 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.88 |
| q69 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q70 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.48 |
| q71 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q72 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.49 |
| q73 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.49 |
| q74 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.53 |
| q75 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.55 |
| q76 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.45 |
| q77 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q78 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q79 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.46 |
| q80 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q81 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.51 |
| q82 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q83 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.46 |
| q84 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q85 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q86 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.48 |
| q87 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q88 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q89 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.46 |
| q90 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.52 |
| q91 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.55 |
| q92 | temporal | STOCK | 0 | 0 |  | true_miss | 0.43 |
| q93 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.52 |
| q94 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q95 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q96 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.51 |
| q97 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q98 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q99 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q100 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q101 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q102 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q103 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q104 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q105 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q106 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q107 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.94 |
| q108 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.44 |
| q109 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.55 |
| q110 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q111 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.48 |
| q112 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q113 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.46 |
| q114 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.54 |
| q115 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.47 |
| q116 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q117 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q118 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q119 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q120 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.6 |
| q121 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.57 |
| q122 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q123 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q124 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.49 |
| q125 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q126 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q127 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q128 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q129 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.57 |
| q130 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q131 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q132 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.74 |
| q133 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q134 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.45 |
| q135 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.57 |
| q136 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q137 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.49 |
| q138 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q139 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.57 |
| q140 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q141 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q142 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.95 |
| q143 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.62 |
| q144 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.56 |
| q145 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.64 |
| q146 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q147 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q148 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q149 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.6 |
| q150 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.52 |
| q151 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q152 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.57 |
| q153 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
| q154 | adversarial | STOCK | 0 | 0 |  | abstention | 0.56 |
| q155 | adversarial | STOCK | 0 | 0 |  | abstention | 0.53 |
| q156 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q157 | adversarial | STOCK | 0 | 0 |  | abstention | 0.82 |
| q158 | adversarial | STOCK | 0 | 0 |  | abstention | 0.46 |
| q159 | adversarial | STOCK | 0 | 0 |  | abstention | 0.46 |
| q160 | adversarial | STOCK | 0 | 0 |  | abstention | 0.51 |
| q161 | adversarial | STOCK | 0 | 0 |  | abstention | 0.73 |
| q162 | adversarial | STOCK | 0 | 0 |  | abstention | 0.51 |
| q163 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q164 | adversarial | STOCK | 0 | 0 |  | abstention | 0.51 |
| q165 | adversarial | STOCK | 0 | 0 |  | abstention | 0.47 |
| q166 | adversarial | STOCK | 0 | 0 |  | abstention | 0.52 |
| q167 | adversarial | STOCK | 0 | 0 |  | abstention | 0.81 |
| q168 | adversarial | STOCK | 0 | 0 |  | abstention | 0.65 |
| q169 | adversarial | STOCK | 0 | 0 |  | abstention | 0.83 |
| q170 | adversarial | STOCK | 0 | 0 |  | abstention | 0.78 |
| q171 | adversarial | STOCK | 0 | 0 |  | abstention | 0.75 |
| q172 | adversarial | STOCK | 0 | 0 |  | abstention | 0.79 |
| q173 | adversarial | STOCK | 0 | 0 |  | abstention | 0.94 |
| q174 | adversarial | STOCK | 0 | 0 |  | abstention | 0.39 |
| q175 | adversarial | STOCK | 0 | 0 |  | abstention | 0.77 |
| q176 | adversarial | STOCK | 0 | 0 |  | abstention | 0.47 |
| q177 | adversarial | STOCK | 0 | 0 |  | abstention | 0.54 |
| q178 | adversarial | STOCK | 0 | 0 |  | abstention | 0.78 |
| q179 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q180 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q181 | adversarial | STOCK | 0 | 0 |  | abstention | 0.78 |
| q182 | adversarial | STOCK | 0 | 0 |  | abstention | 0.59 |
| q183 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q184 | adversarial | STOCK | 0 | 0 |  | abstention | 0.57 |
| q185 | adversarial | STOCK | 0 | 0 |  | abstention | 0.86 |
| q186 | adversarial | STOCK | 0 | 0 |  | abstention | 0.62 |
| q187 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q188 | adversarial | STOCK | 0 | 0 |  | abstention | 0.78 |
| q189 | adversarial | STOCK | 0 | 0 |  | abstention | 0.61 |
| q190 | adversarial | STOCK | 0 | 0 |  | abstention | 0.56 |
| q191 | adversarial | STOCK | 0 | 0 |  | abstention | 0.64 |
| q192 | adversarial | STOCK | 0 | 0 |  | abstention | 0.55 |
| q193 | adversarial | STOCK | 0 | 0 |  | abstention | 0.6 |
