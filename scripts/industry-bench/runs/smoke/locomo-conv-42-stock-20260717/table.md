# smoke conv-42

- questions: 260
- route: stock
- calibration: stock
- evidence@5 (answerable): 0.90
- evidence@10 (answerable): 0.91
- retrieved_context_tokens mean: 214.0
- answer_path_tokens mean: 241.7

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.69 |
| q2 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q3 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.58 |
| q4 | temporal | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q5 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.65 |
| q6 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q7 | temporal | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q8 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q9 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.68 |
| q10 | temporal | STOCK | 0 | 0 |  | true_miss | 0.62 |
| q11 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.76 |
| q12 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q13 | open-domain | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.6 |
| q14 | temporal | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q15 | open-domain | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.68 |
| q16 | temporal | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q17 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.54 |
| q18 | temporal | STOCK | 0 | 0 |  | true_miss | 0.58 |
| q19 | temporal | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q20 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q21 | temporal | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q22 | temporal | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q23 | temporal | STOCK | 0 | 0 |  | true_miss | 0.66 |
| q24 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q25 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.81 |
| q26 | temporal | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q27 | temporal | STOCK | 0 | 0 |  | true_miss | 0.67 |
| q28 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q29 | temporal | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q30 | temporal | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q31 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q32 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.72 |
| q33 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.54 |
| q34 | temporal | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q35 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q36 | temporal | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q37 | temporal | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q38 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q39 | temporal | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q40 | temporal | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q41 | temporal | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q42 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.8 |
| q43 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q44 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q45 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q46 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.68 |
| q47 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q48 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q49 | temporal | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q50 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.6 |
| q51 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.76 |
| q52 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q53 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.66 |
| q54 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q55 | temporal | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q56 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.62 |
| q57 | multi-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.73 |
| q58 | temporal | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q59 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q60 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.63 |
| q61 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.54 |
| q62 | multi-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.74 |
| q63 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q64 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q65 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.63 |
| q66 | temporal | STOCK | 0 | 0 |  | true_miss | 0.61 |
| q67 | open-domain | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.64 |
| q68 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q69 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q70 | multi-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.67 |
| q71 | multi-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.71 |
| q72 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q73 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q74 | open-domain | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.54 |
| q75 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q76 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q77 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q78 | temporal | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q79 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q80 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q81 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q82 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q83 | multi-hop | STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.73 |
| q84 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q85 | open-domain | STOCK | 0 | 1 | 6 | deep_rank_lt10 | 0.73 |
| q86 | open-domain | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.68 |
| q87 | temporal | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q88 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.56 |
| q89 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q90 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q91 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q92 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q93 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q94 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q95 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.63 |
| q96 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.77 |
| q97 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q98 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.54 |
| q99 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.63 |
| q100 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q101 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q102 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.81 |
| q103 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q104 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q105 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.59 |
| q106 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q107 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q108 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q109 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q110 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q111 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q112 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.74 |
| q113 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q114 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q115 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q116 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q117 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q118 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q119 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q120 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q121 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q122 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q123 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.68 |
| q124 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.77 |
| q125 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.55 |
| q126 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.75 |
| q127 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q128 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q129 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.56 |
| q130 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q131 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.86 |
| q132 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q133 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q134 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q135 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.67 |
| q136 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q137 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.77 |
| q138 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q139 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.59 |
| q140 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q141 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.64 |
| q142 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q143 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q144 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q145 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q146 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q147 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.84 |
| q148 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.79 |
| q149 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q150 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q151 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q152 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q153 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q154 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q155 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q156 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.89 |
| q157 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q158 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.96 |
| q159 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q160 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.94 |
| q161 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q162 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q163 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q164 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q165 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q166 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q167 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.78 |
| q168 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q169 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.89 |
| q170 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.72 |
| q171 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q172 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.86 |
| q173 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q174 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q175 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q176 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q177 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q178 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q179 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q180 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q181 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.71 |
| q182 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.74 |
| q183 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.67 |
| q184 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.66 |
| q185 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q186 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.67 |
| q187 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.68 |
| q188 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.61 |
| q189 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q190 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q191 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q192 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q193 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q194 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q195 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.65 |
| q196 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.7 |
| q197 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q198 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q199 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.64 |
| q200 | adversarial | STOCK | 0 | 0 |  | abstention | 0.9 |
| q201 | adversarial | STOCK | 0 | 0 |  | abstention | 0.62 |
| q202 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q203 | adversarial | STOCK | 0 | 0 |  | abstention | 0.64 |
| q204 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q205 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q206 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q207 | adversarial | STOCK | 0 | 0 |  | abstention | 0.6 |
| q208 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q209 | adversarial | STOCK | 0 | 0 |  | abstention | 0.62 |
| q210 | adversarial | STOCK | 0 | 0 |  | abstention | 0.73 |
| q211 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q212 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q213 | adversarial | STOCK | 0 | 0 |  | abstention | 0.9 |
| q214 | adversarial | STOCK | 0 | 0 |  | abstention | 0.71 |
| q215 | adversarial | STOCK | 0 | 0 |  | abstention | 0.75 |
| q216 | adversarial | STOCK | 0 | 0 |  | abstention | 0.59 |
| q217 | adversarial | STOCK | 0 | 0 |  | abstention | 0.66 |
| q218 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q219 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q220 | adversarial | STOCK | 0 | 0 |  | abstention | 0.51 |
| q221 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
| q222 | adversarial | STOCK | 0 | 0 |  | abstention | 0.83 |
| q223 | adversarial | STOCK | 0 | 0 |  | abstention | 0.82 |
| q224 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q225 | adversarial | STOCK | 0 | 0 |  | abstention | 0.55 |
| q226 | adversarial | STOCK | 0 | 0 |  | abstention | 0.68 |
| q227 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
| q228 | adversarial | STOCK | 0 | 0 |  | abstention | 0.62 |
| q229 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q230 | adversarial | STOCK | 0 | 0 |  | abstention | 0.77 |
| q231 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q232 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q233 | adversarial | STOCK | 0 | 0 |  | abstention | 0.8 |
| q234 | adversarial | STOCK | 0 | 0 |  | abstention | 0.89 |
| q235 | adversarial | STOCK | 0 | 0 |  | abstention | 0.96 |
| q236 | adversarial | STOCK | 0 | 0 |  | abstention | 0.93 |
| q237 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q238 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q239 | adversarial | STOCK | 0 | 0 |  | abstention | 0.89 |
| q240 | adversarial | STOCK | 0 | 0 |  | abstention | 0.78 |
| q241 | adversarial | STOCK | 0 | 0 |  | abstention | 0.88 |
| q242 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q243 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
| q244 | adversarial | STOCK | 0 | 0 |  | abstention | 0.85 |
| q245 | adversarial | STOCK | 0 | 0 |  | abstention | 0.6 |
| q246 | adversarial | STOCK | 0 | 0 |  | abstention | 0.75 |
| q247 | adversarial | STOCK | 0 | 0 |  | abstention | 0.65 |
| q248 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q249 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q250 | adversarial | STOCK | 0 | 0 |  | abstention | 0.62 |
| q251 | adversarial | STOCK | 0 | 0 |  | abstention | 0.76 |
| q252 | adversarial | STOCK | 0 | 0 |  | abstention | 0.6 |
| q253 | adversarial | STOCK | 0 | 0 |  | abstention | 0.64 |
| q254 | adversarial | STOCK | 0 | 0 |  | abstention | 0.59 |
| q255 | adversarial | STOCK | 0 | 0 |  | abstention | 0.83 |
| q256 | adversarial | STOCK | 0 | 0 |  | abstention | 0.65 |
| q257 | adversarial | STOCK | 0 | 0 |  | abstention | 0.82 |
| q258 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q259 | adversarial | STOCK | 0 | 0 |  | abstention | 0.61 |
| q260 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
