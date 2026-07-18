# smoke conv-42

- questions: 260
- route: stock
- calibration: stock
- evidence@5 (answerable): 0.88
- evidence@10 (answerable): 0.89
- retrieved_context_tokens mean: 158.5
- answer_path_tokens mean: 186.0

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | open-domain | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q2 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.51 |
| q3 | temporal | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.51 |
| q4 | temporal | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q5 | open-domain | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.5 |
| q6 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q7 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.58 |
| q8 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.62 |
| q9 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.52 |
| q10 | temporal | STOCK | 0 | 0 |  | true_miss | 0.5 |
| q11 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.76 |
| q12 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q13 | open-domain | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.6 |
| q14 | temporal | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q15 | open-domain | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.54 |
| q16 | temporal | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q17 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.43 |
| q18 | temporal | STOCK | 0 | 0 |  | true_miss | 0.57 |
| q19 | temporal | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q20 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q21 | temporal | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q22 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q23 | temporal | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.66 |
| q24 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.86 |
| q25 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.81 |
| q26 | temporal | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q27 | temporal | STOCK | 0 | 0 |  | true_miss | 0.43 |
| q28 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q29 | temporal | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q30 | temporal | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q31 | multi-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.45 |
| q32 | temporal | STOCK | 0 | 0 |  | true_miss | 0.66 |
| q33 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.49 |
| q34 | temporal | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q35 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.65 |
| q36 | temporal | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q37 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.49 |
| q38 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.64 |
| q39 | temporal | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q40 | temporal | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q41 | temporal | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q42 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.8 |
| q43 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q44 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.52 |
| q45 | temporal | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q46 | temporal | STOCK | 1 | 1 | 0 | hit | 0.51 |
| q47 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q48 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.57 |
| q49 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q50 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.42 |
| q51 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q52 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.53 |
| q53 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.55 |
| q54 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.57 |
| q55 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.58 |
| q56 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.39 |
| q57 | multi-hop | STOCK | 0 | 1 | 6 | deep_rank_lt10 | 0.61 |
| q58 | temporal | STOCK | 1 | 1 | 0 | hit | 0.48 |
| q59 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.56 |
| q60 | multi-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.62 |
| q61 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.43 |
| q62 | multi-hop | STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.51 |
| q63 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.56 |
| q64 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q65 | temporal | STOCK | 0 | 0 |  | true_miss | 0.48 |
| q66 | temporal | STOCK | 0 | 0 |  | true_miss | 0.54 |
| q67 | open-domain | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.53 |
| q68 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.46 |
| q69 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q70 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.47 |
| q71 | multi-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.47 |
| q72 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.55 |
| q73 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.56 |
| q74 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.57 |
| q75 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q76 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.57 |
| q77 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q78 | temporal | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q79 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.56 |
| q80 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.48 |
| q81 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.54 |
| q82 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.56 |
| q83 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.55 |
| q84 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.48 |
| q85 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.73 |
| q86 | open-domain | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.54 |
| q87 | temporal | STOCK | 1 | 1 | 0 | hit | 0.56 |
| q88 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.57 |
| q89 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q90 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q91 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.52 |
| q92 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.62 |
| q93 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.48 |
| q94 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q95 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.39 |
| q96 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.6 |
| q97 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q98 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.54 |
| q99 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.46 |
| q100 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q101 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.42 |
| q102 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.76 |
| q103 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.6 |
| q104 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q105 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.59 |
| q106 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q107 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.46 |
| q108 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.56 |
| q109 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.53 |
| q110 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.53 |
| q111 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q112 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.69 |
| q113 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.52 |
| q114 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.74 |
| q115 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.48 |
| q116 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.55 |
| q117 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.48 |
| q118 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q119 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q120 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q121 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q122 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q123 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.54 |
| q124 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.68 |
| q125 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.57 |
| q126 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q127 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q128 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q129 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.61 |
| q130 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q131 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.86 |
| q132 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q133 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q134 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q135 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.63 |
| q136 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.48 |
| q137 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.7 |
| q138 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q139 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.57 |
| q140 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q141 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.49 |
| q142 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q143 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q144 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q145 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q146 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.58 |
| q147 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.84 |
| q148 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.74 |
| q149 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q150 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.5 |
| q151 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q152 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q153 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q154 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q155 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q156 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.89 |
| q157 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q158 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.96 |
| q159 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q160 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.65 |
| q161 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.69 |
| q162 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.52 |
| q163 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.51 |
| q164 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q165 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.51 |
| q166 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.6 |
| q167 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.78 |
| q168 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.47 |
| q169 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.89 |
| q170 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.6 |
| q171 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q172 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.86 |
| q173 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q174 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q175 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.57 |
| q176 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q177 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.47 |
| q178 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q179 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q180 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q181 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.59 |
| q182 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.74 |
| q183 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.52 |
| q184 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.63 |
| q185 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.56 |
| q186 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.55 |
| q187 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.51 |
| q188 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.58 |
| q189 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.52 |
| q190 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q191 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q192 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q193 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.62 |
| q194 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q195 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.47 |
| q196 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.62 |
| q197 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.57 |
| q198 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q199 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.47 |
| q200 | adversarial | STOCK | 0 | 0 |  | abstention | 0.9 |
| q201 | adversarial | STOCK | 0 | 0 |  | abstention | 0.51 |
| q202 | adversarial | STOCK | 0 | 0 |  | abstention | 0.64 |
| q203 | adversarial | STOCK | 0 | 0 |  | abstention | 0.66 |
| q204 | adversarial | STOCK | 0 | 0 |  | abstention | 0.61 |
| q205 | adversarial | STOCK | 0 | 0 |  | abstention | 0.38 |
| q206 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q207 | adversarial | STOCK | 0 | 0 |  | abstention | 0.48 |
| q208 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q209 | adversarial | STOCK | 0 | 0 |  | abstention | 0.5 |
| q210 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q211 | adversarial | STOCK | 0 | 0 |  | abstention | 0.51 |
| q212 | adversarial | STOCK | 0 | 0 |  | abstention | 0.52 |
| q213 | adversarial | STOCK | 0 | 0 |  | abstention | 0.9 |
| q214 | adversarial | STOCK | 0 | 0 |  | abstention | 0.71 |
| q215 | adversarial | STOCK | 0 | 0 |  | abstention | 0.52 |
| q216 | adversarial | STOCK | 0 | 0 |  | abstention | 0.55 |
| q217 | adversarial | STOCK | 0 | 0 |  | abstention | 0.61 |
| q218 | adversarial | STOCK | 0 | 0 |  | abstention | 0.62 |
| q219 | adversarial | STOCK | 0 | 0 |  | abstention | 0.66 |
| q220 | adversarial | STOCK | 0 | 0 |  | abstention | 0.61 |
| q221 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
| q222 | adversarial | STOCK | 0 | 0 |  | abstention | 0.83 |
| q223 | adversarial | STOCK | 0 | 0 |  | abstention | 0.82 |
| q224 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q225 | adversarial | STOCK | 0 | 0 |  | abstention | 0.57 |
| q226 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q227 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
| q228 | adversarial | STOCK | 0 | 0 |  | abstention | 0.58 |
| q229 | adversarial | STOCK | 0 | 0 |  | abstention | 0.51 |
| q230 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q231 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q232 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q233 | adversarial | STOCK | 0 | 0 |  | abstention | 0.8 |
| q234 | adversarial | STOCK | 0 | 0 |  | abstention | 0.89 |
| q235 | adversarial | STOCK | 0 | 0 |  | abstention | 0.96 |
| q236 | adversarial | STOCK | 0 | 0 |  | abstention | 0.66 |
| q237 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q238 | adversarial | STOCK | 0 | 0 |  | abstention | 0.55 |
| q239 | adversarial | STOCK | 0 | 0 |  | abstention | 0.49 |
| q240 | adversarial | STOCK | 0 | 0 |  | abstention | 0.78 |
| q241 | adversarial | STOCK | 0 | 0 |  | abstention | 0.48 |
| q242 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q243 | adversarial | STOCK | 0 | 0 |  | abstention | 0.62 |
| q244 | adversarial | STOCK | 0 | 0 |  | abstention | 0.85 |
| q245 | adversarial | STOCK | 0 | 0 |  | abstention | 0.56 |
| q246 | adversarial | STOCK | 0 | 0 |  | abstention | 0.75 |
| q247 | adversarial | STOCK | 0 | 0 |  | abstention | 0.64 |
| q248 | adversarial | STOCK | 0 | 0 |  | abstention | 0.59 |
| q249 | adversarial | STOCK | 0 | 0 |  | abstention | 0.52 |
| q250 | adversarial | STOCK | 0 | 0 |  | abstention | 0.62 |
| q251 | adversarial | STOCK | 0 | 0 |  | abstention | 0.51 |
| q252 | adversarial | STOCK | 0 | 0 |  | abstention | 0.56 |
| q253 | adversarial | STOCK | 0 | 0 |  | abstention | 0.57 |
| q254 | adversarial | STOCK | 0 | 0 |  | abstention | 0.51 |
| q255 | adversarial | STOCK | 0 | 0 |  | abstention | 0.83 |
| q256 | adversarial | STOCK | 0 | 0 |  | abstention | 0.53 |
| q257 | adversarial | STOCK | 0 | 0 |  | abstention | 0.61 |
| q258 | adversarial | STOCK | 0 | 0 |  | abstention | 0.47 |
| q259 | adversarial | STOCK | 0 | 0 |  | abstention | 0.47 |
| q260 | adversarial | STOCK | 0 | 0 |  | abstention | 0.61 |
