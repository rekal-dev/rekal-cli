# smoke conv-49

- questions: 196
- route: stock
- calibration: stock
- evidence@5 (answerable): 0.88
- evidence@10 (answerable): 0.91
- retrieved_context_tokens mean: 235.2
- answer_path_tokens mean: 267.4

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.53 |
| q2 | multi-hop | STOCK | 0 | 1 | 6 | deep_rank_lt10 | 0.62 |
| q3 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.65 |
| q4 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q5 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.66 |
| q6 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.59 |
| q7 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.61 |
| q8 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q9 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q10 | temporal | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q11 | open-domain | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.66 |
| q12 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q13 | temporal | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q14 | temporal | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q15 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.68 |
| q16 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.67 |
| q17 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q18 | temporal | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q19 | multi-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.79 |
| q20 | open-domain | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.78 |
| q21 | open-domain | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q22 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q23 | temporal | STOCK | 0 | 0 |  | true_miss | 0.61 |
| q24 | temporal | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q25 | temporal | STOCK | 0 | 0 |  | true_miss | 0.72 |
| q26 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.69 |
| q27 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q28 | open-domain | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q29 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q30 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q31 | temporal | STOCK | 0 | 0 |  | true_miss | 0.65 |
| q32 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.8 |
| q33 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.59 |
| q34 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q35 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q36 | temporal | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q37 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q38 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q39 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.81 |
| q40 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.61 |
| q41 | temporal | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q42 | temporal | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q43 | multi-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.74 |
| q44 | open-domain | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.61 |
| q45 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q46 | temporal | STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.72 |
| q47 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.71 |
| q48 | temporal | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q49 | multi-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.74 |
| q50 | multi-hop | STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.63 |
| q51 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q52 | open-domain | STOCK | 0 | 1 | 6 | deep_rank_lt10 | 0.72 |
| q53 | temporal | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.68 |
| q54 | temporal | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.73 |
| q55 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q56 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.61 |
| q57 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q58 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.6 |
| q59 | temporal | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q60 | temporal | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q61 | temporal | STOCK | 0 | 0 |  | true_miss | 0.72 |
| q62 | temporal | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q63 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q64 | temporal | STOCK | 1 | 1 | 0 | hit | 0.59 |
| q65 | multi-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.67 |
| q66 | open-domain | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q67 | temporal | STOCK | 1 | 1 | 0 | hit | 0.57 |
| q68 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.72 |
| q69 | temporal | STOCK | 0 | 0 |  | true_miss | 0.6 |
| q70 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q71 | temporal | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q72 | open-domain | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.65 |
| q73 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q74 | temporal | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q75 | temporal | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q76 | temporal | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q77 | temporal | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q78 | multi-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.74 |
| q79 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q80 | temporal | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q81 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q82 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q83 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q84 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q85 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q86 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.86 |
| q87 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q88 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q89 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q90 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q91 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q92 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q93 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q94 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.6 |
| q95 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q96 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q97 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q98 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q99 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q100 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q101 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q102 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q103 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q104 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q105 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q106 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q107 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q108 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q109 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q110 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.65 |
| q111 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.58 |
| q112 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q113 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.7 |
| q114 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q115 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q116 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q117 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.75 |
| q118 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q119 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.6 |
| q120 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.71 |
| q121 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q122 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q123 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.63 |
| q124 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q125 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q126 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.7 |
| q127 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q128 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.7 |
| q129 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q130 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q131 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q132 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q133 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q134 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q135 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q136 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q137 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.77 |
| q138 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q139 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q140 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.7 |
| q141 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q142 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q143 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q144 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q145 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q146 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q147 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q148 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q149 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q150 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q151 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q152 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q153 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q154 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q155 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q156 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q157 | adversarial | STOCK | 0 | 0 |  | abstention | 0.73 |
| q158 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q159 | adversarial | STOCK | 0 | 0 |  | abstention | 0.86 |
| q160 | adversarial | STOCK | 0 | 0 |  | abstention | 0.61 |
| q161 | adversarial | STOCK | 0 | 0 |  | abstention | 0.85 |
| q162 | adversarial | STOCK | 0 | 0 |  | abstention | 0.71 |
| q163 | adversarial | STOCK | 0 | 0 |  | abstention | 0.77 |
| q164 | adversarial | STOCK | 0 | 0 |  | abstention | 0.61 |
| q165 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q166 | adversarial | STOCK | 0 | 0 |  | abstention | 0.61 |
| q167 | adversarial | STOCK | 0 | 0 |  | abstention | 0.75 |
| q168 | adversarial | STOCK | 0 | 0 |  | abstention | 0.77 |
| q169 | adversarial | STOCK | 0 | 0 |  | abstention | 0.65 |
| q170 | adversarial | STOCK | 0 | 0 |  | abstention | 0.66 |
| q171 | adversarial | STOCK | 0 | 0 |  | abstention | 0.71 |
| q172 | adversarial | STOCK | 0 | 0 |  | abstention | 0.75 |
| q173 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q174 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q175 | adversarial | STOCK | 0 | 0 |  | abstention | 0.79 |
| q176 | adversarial | STOCK | 0 | 0 |  | abstention | 0.83 |
| q177 | adversarial | STOCK | 0 | 0 |  | abstention | 0.58 |
| q178 | adversarial | STOCK | 0 | 0 |  | abstention | 0.6 |
| q179 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q180 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q181 | adversarial | STOCK | 0 | 0 |  | abstention | 0.68 |
| q182 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
| q183 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q184 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q185 | adversarial | STOCK | 0 | 0 |  | abstention | 0.85 |
| q186 | adversarial | STOCK | 0 | 0 |  | abstention | 0.65 |
| q187 | adversarial | STOCK | 0 | 0 |  | abstention | 0.87 |
| q188 | adversarial | STOCK | 0 | 0 |  | abstention | 0.76 |
| q189 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q190 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q191 | adversarial | STOCK | 0 | 0 |  | abstention | 0.64 |
| q192 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q193 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
| q194 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q195 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q196 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
