# smoke conv-48

- questions: 239
- route: stock
- calibration: stock
- evidence@5 (answerable): 0.92
- evidence@10 (answerable): 0.93
- retrieved_context_tokens mean: 194.3
- answer_path_tokens mean: 224.8

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | temporal | STOCK | 0 | 0 |  | true_miss | 0.66 |
| q2 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q3 | temporal | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q4 | temporal | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q5 | temporal | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q6 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.86 |
| q7 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.86 |
| q8 | temporal | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q9 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q10 | temporal | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q11 | temporal | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q12 | open-domain | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q13 | temporal | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q14 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q15 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q16 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q17 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.58 |
| q18 | temporal | STOCK | 1 | 1 | 0 | hit | 0.57 |
| q19 | open-domain | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.57 |
| q20 | multi-hop | STOCK | 0 | 1 | 6 | deep_rank_lt10 | 0.62 |
| q21 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.66 |
| q22 | temporal | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q23 | temporal | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q24 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.71 |
| q25 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q26 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q27 | temporal | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.66 |
| q28 | temporal | STOCK | 1 | 1 | 0 | hit | 0.52 |
| q29 | open-domain | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.73 |
| q30 | temporal | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q31 | temporal | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q32 | temporal | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q33 | temporal | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q34 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.62 |
| q35 | temporal | STOCK | 0 | 0 |  | true_miss | 0.62 |
| q36 | temporal | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q37 | open-domain | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.62 |
| q38 | temporal | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q39 | temporal | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q40 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q41 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.64 |
| q42 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q43 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q44 | temporal | STOCK | 0 | 0 |  | true_miss | 0.63 |
| q45 | temporal | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q46 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.78 |
| q47 | temporal | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q48 | temporal | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q49 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q50 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q51 | temporal | STOCK | 0 | 0 |  | true_miss | 0.75 |
| q52 | temporal | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q53 | temporal | STOCK | 0 | 0 |  | true_miss | 0.66 |
| q54 | temporal | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q55 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q56 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q57 | temporal | STOCK | 0 | 0 |  | true_miss | 0.7 |
| q58 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q59 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q60 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q61 | temporal | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q62 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q63 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q64 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.58 |
| q65 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.59 |
| q66 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q67 | temporal | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q68 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q69 | temporal | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q70 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q71 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q72 | temporal | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q73 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q74 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q75 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q76 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q77 | temporal | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q78 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q79 | temporal | STOCK | 0 | 0 |  | true_miss | 0.63 |
| q80 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.6 |
| q81 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q82 | temporal | STOCK | 1 | 1 | 0 | hit | 0.6 |
| q83 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.7 |
| q84 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q85 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q86 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.67 |
| q87 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.7 |
| q88 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q89 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.7 |
| q90 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.58 |
| q91 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q92 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q93 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q94 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q95 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q96 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.58 |
| q97 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q98 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q99 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q100 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q101 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q102 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q103 | single-hop | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.68 |
| q104 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q105 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.76 |
| q106 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.63 |
| q107 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q108 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q109 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q110 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q111 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.86 |
| q112 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q113 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q114 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q115 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.7 |
| q116 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q117 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q118 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q119 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q120 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.69 |
| q121 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q122 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.7 |
| q123 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q124 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q125 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q126 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q127 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q128 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q129 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q130 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q131 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q132 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q133 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q134 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.63 |
| q135 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q136 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q137 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q138 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q139 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q140 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q141 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.74 |
| q142 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q143 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q144 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q145 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q146 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.74 |
| q147 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.73 |
| q148 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q149 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.91 |
| q150 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q151 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q152 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q153 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q154 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q155 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q156 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q157 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q158 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q159 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q160 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.86 |
| q161 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q162 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q163 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.86 |
| q164 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q165 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.47 |
| q166 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q167 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q168 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.62 |
| q169 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.6 |
| q170 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q171 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q172 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q173 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q174 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q175 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.67 |
| q176 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q177 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q178 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q179 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q180 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q181 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q182 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.58 |
| q183 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.73 |
| q184 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.64 |
| q185 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q186 | single-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.68 |
| q187 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q188 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q189 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.77 |
| q190 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q191 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q192 | adversarial | STOCK | 0 | 0 |  | abstention | 0.6 |
| q193 | adversarial | STOCK | 0 | 0 |  | abstention | 0.65 |
| q194 | adversarial | STOCK | 0 | 0 |  | abstention | 0.6 |
| q195 | adversarial | STOCK | 0 | 0 |  | abstention | 0.56 |
| q196 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q197 | adversarial | STOCK | 0 | 0 |  | abstention | 0.59 |
| q198 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q199 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q200 | adversarial | STOCK | 0 | 0 |  | abstention | 0.84 |
| q201 | adversarial | STOCK | 0 | 0 |  | abstention | 0.9 |
| q202 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q203 | adversarial | STOCK | 0 | 0 |  | abstention | 0.71 |
| q204 | adversarial | STOCK | 0 | 0 |  | abstention | 0.65 |
| q205 | adversarial | STOCK | 0 | 0 |  | abstention | 0.92 |
| q206 | adversarial | STOCK | 0 | 0 |  | abstention | 0.84 |
| q207 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q208 | adversarial | STOCK | 0 | 0 |  | abstention | 0.73 |
| q209 | adversarial | STOCK | 0 | 0 |  | abstention | 0.66 |
| q210 | adversarial | STOCK | 0 | 0 |  | abstention | 0.66 |
| q211 | adversarial | STOCK | 0 | 0 |  | abstention | 0.71 |
| q212 | adversarial | STOCK | 0 | 0 |  | abstention | 0.55 |
| q213 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q214 | adversarial | STOCK | 0 | 0 |  | abstention | 0.77 |
| q215 | adversarial | STOCK | 0 | 0 |  | abstention | 0.62 |
| q216 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q217 | adversarial | STOCK | 0 | 0 |  | abstention | 0.85 |
| q218 | adversarial | STOCK | 0 | 0 |  | abstention | 0.9 |
| q219 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q220 | adversarial | STOCK | 0 | 0 |  | abstention | 0.84 |
| q221 | adversarial | STOCK | 0 | 0 |  | abstention | 0.77 |
| q222 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q223 | adversarial | STOCK | 0 | 0 |  | abstention | 0.85 |
| q224 | adversarial | STOCK | 0 | 0 |  | abstention | 0.66 |
| q225 | adversarial | STOCK | 0 | 0 |  | abstention | 0.8 |
| q226 | adversarial | STOCK | 0 | 0 |  | abstention | 0.87 |
| q227 | adversarial | STOCK | 0 | 0 |  | abstention | 0.62 |
| q228 | adversarial | STOCK | 0 | 0 |  | abstention | 0.9 |
| q229 | adversarial | STOCK | 0 | 0 |  | abstention | 0.6 |
| q230 | adversarial | STOCK | 0 | 0 |  | abstention | 0.77 |
| q231 | adversarial | STOCK | 0 | 0 |  | abstention | 0.82 |
| q232 | adversarial | STOCK | 0 | 0 |  | abstention | 0.79 |
| q233 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q234 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q235 | adversarial | STOCK | 0 | 0 |  | abstention | 0.59 |
| q236 | adversarial | STOCK | 0 | 0 |  | abstention | 0.64 |
| q237 | adversarial | STOCK | 0 | 0 |  | abstention | 0.71 |
| q238 | adversarial | STOCK | 0 | 0 |  | abstention | 0.86 |
| q239 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
