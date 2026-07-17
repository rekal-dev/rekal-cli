# smoke conv-50

- questions: 204
- route: stock
- calibration: stock
- evidence@5 (answerable): 0.87
- evidence@10 (answerable): 0.89
- retrieved_context_tokens mean: 252.2
- answer_path_tokens mean: 286.5

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | temporal | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q2 | multi-hop | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.63 |
| q3 | temporal | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q4 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.62 |
| q5 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.75 |
| q6 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.62 |
| q7 | multi-hop | STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.68 |
| q8 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.6 |
| q9 | temporal | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q10 | temporal | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q11 | temporal | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q12 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q13 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.74 |
| q14 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q15 | temporal | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q16 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q17 | temporal | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q18 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.66 |
| q19 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q20 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q21 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q22 | open-domain | STOCK | 0 | 1 | 9 | deep_rank_lt10 | 0.65 |
| q23 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q24 | multi-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.64 |
| q25 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q26 | temporal | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q27 | temporal | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q28 | temporal | STOCK | 0 | 0 |  | true_miss | 0.65 |
| q29 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q30 | multi-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.7 |
| q31 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q32 | temporal | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.62 |
| q33 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q34 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q35 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q36 | temporal | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q37 | temporal | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q38 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q39 | temporal | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q40 | open-domain | STOCK | 0 | 0 |  | abstention | 0.54 |
| q41 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.67 |
| q42 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q43 | open-domain | STOCK | 0 | 0 |  | abstention | 0.73 |
| q44 | temporal | STOCK | 0 | 0 |  | true_miss | 0.64 |
| q45 | temporal | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q46 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.71 |
| q47 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.63 |
| q48 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q49 | temporal | STOCK | 0 | 0 |  | true_miss | 0.61 |
| q50 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q51 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q52 | temporal | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q53 | temporal | STOCK | 0 | 0 |  | true_miss | 0.71 |
| q54 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q55 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q56 | temporal | STOCK | 1 | 1 | 0 | hit | 0.91 |
| q57 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.66 |
| q58 | temporal | STOCK | 0 | 0 |  | true_miss | 0.63 |
| q59 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q60 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q61 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q62 | temporal | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q63 | temporal | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q64 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q65 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q66 | temporal | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q67 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.76 |
| q68 | temporal | STOCK | 0 | 0 |  | true_miss | 0.57 |
| q69 | temporal | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q70 | temporal | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q71 | temporal | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q72 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q73 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q74 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.65 |
| q75 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q76 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.58 |
| q77 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q78 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q79 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q80 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q81 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q82 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q83 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q84 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q85 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q86 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q87 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.72 |
| q88 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.79 |
| q89 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q90 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q91 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.89 |
| q92 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q93 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q94 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.71 |
| q95 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q96 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q97 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q98 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q99 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q100 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q101 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q102 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q103 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.86 |
| q104 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q105 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q106 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q107 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q108 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q109 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q110 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q111 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.78 |
| q112 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q113 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q114 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q115 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q116 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q117 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q118 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.81 |
| q119 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q120 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.74 |
| q121 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.75 |
| q122 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.68 |
| q123 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q124 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q125 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.77 |
| q126 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.84 |
| q127 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q128 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q129 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.73 |
| q130 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q131 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q132 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q133 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q134 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q135 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.88 |
| q136 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q137 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q138 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.76 |
| q139 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q140 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q141 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.75 |
| q142 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q143 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.77 |
| q144 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q145 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q146 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q147 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.69 |
| q148 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q149 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q150 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q151 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q152 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.71 |
| q153 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.57 |
| q154 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q155 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.73 |
| q156 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q157 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q158 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q159 | adversarial | STOCK | 0 | 0 |  | abstention | 0.61 |
| q160 | adversarial | STOCK | 0 | 0 |  | abstention | 0.71 |
| q161 | adversarial | STOCK | 0 | 0 |  | abstention | 0.64 |
| q162 | adversarial | STOCK | 0 | 0 |  | abstention | 0.75 |
| q163 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q164 | adversarial | STOCK | 0 | 0 |  | abstention | 0.58 |
| q165 | adversarial | STOCK | 0 | 0 |  | abstention | 0.64 |
| q166 | adversarial | STOCK | 0 | 0 |  | abstention | 0.81 |
| q167 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q168 | adversarial | STOCK | 0 | 0 |  | abstention | 0.81 |
| q169 | adversarial | STOCK | 0 | 0 |  | abstention | 0.64 |
| q170 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q171 | adversarial | STOCK | 0 | 0 |  | abstention | 0.82 |
| q172 | adversarial | STOCK | 0 | 0 |  | abstention | 0.79 |
| q173 | adversarial | STOCK | 0 | 0 |  | abstention | 0.64 |
| q174 | adversarial | STOCK | 0 | 0 |  | abstention | 0.86 |
| q175 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q176 | adversarial | STOCK | 0 | 0 |  | abstention | 0.85 |
| q177 | adversarial | STOCK | 0 | 0 |  | abstention | 0.71 |
| q178 | adversarial | STOCK | 0 | 0 |  | abstention | 0.77 |
| q179 | adversarial | STOCK | 0 | 0 |  | abstention | 0.8 |
| q180 | adversarial | STOCK | 0 | 0 |  | abstention | 0.79 |
| q181 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q182 | adversarial | STOCK | 0 | 0 |  | abstention | 0.84 |
| q183 | adversarial | STOCK | 0 | 0 |  | abstention | 0.71 |
| q184 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
| q185 | adversarial | STOCK | 0 | 0 |  | abstention | 0.6 |
| q186 | adversarial | STOCK | 0 | 0 |  | abstention | 0.64 |
| q187 | adversarial | STOCK | 0 | 0 |  | abstention | 0.65 |
| q188 | adversarial | STOCK | 0 | 0 |  | abstention | 0.75 |
| q189 | adversarial | STOCK | 0 | 0 |  | abstention | 0.61 |
| q190 | adversarial | STOCK | 0 | 0 |  | abstention | 0.77 |
| q191 | adversarial | STOCK | 0 | 0 |  | abstention | 0.64 |
| q192 | adversarial | STOCK | 0 | 0 |  | abstention | 0.71 |
| q193 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q194 | adversarial | STOCK | 0 | 0 |  | abstention | 0.61 |
| q195 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q196 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q197 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q198 | adversarial | STOCK | 0 | 0 |  | abstention | 0.79 |
| q199 | adversarial | STOCK | 0 | 0 |  | abstention | 0.65 |
| q200 | adversarial | STOCK | 0 | 0 |  | abstention | 0.65 |
| q201 | adversarial | STOCK | 0 | 0 |  | abstention | 0.62 |
| q202 | adversarial | STOCK | 0 | 0 |  | abstention | 0.77 |
| q203 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q204 | adversarial | STOCK | 0 | 0 |  | abstention | 0.73 |
