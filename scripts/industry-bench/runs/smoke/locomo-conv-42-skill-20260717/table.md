# smoke conv-42

- questions: 260
- route: skill
- calibration: chat-provisional.json
- evidence@5 (answerable): 0.88
- evidence@10 (answerable): 0.91
- retrieved_context_tokens mean: 158.0
- answer_path_tokens mean: 187.6

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.7 |
| q2 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q3 | temporal | INJECT | 1 | 1 | 0 | hit | 0.6 |
| q4 | temporal | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q5 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.65 |
| q6 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q7 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q8 | temporal | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q9 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q10 | temporal | INJECT | 0 | 0 |  | true_miss | 0.62 |
| q11 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.76 |
| q12 | multi-hop | INJECT | 0 | 1 | 5 | deep_rank_lt10 | 0.63 |
| q13 | open-domain | INJECT | 0 | 1 | 8 | deep_rank_lt10 | 0.6 |
| q14 | temporal | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q15 | open-domain | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.69 |
| q16 | temporal | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q17 | temporal | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q18 | temporal | INJECT | 0 | 0 |  | true_miss | 0.59 |
| q19 | temporal | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q20 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q21 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q22 | temporal | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q23 | temporal | INJECT | 0 | 0 |  | true_miss | 0.66 |
| q24 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.86 |
| q25 | temporal | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.81 |
| q26 | temporal | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.68 |
| q27 | temporal | INJECT | 0 | 0 |  | true_miss | 0.67 |
| q28 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q29 | temporal | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q30 | temporal | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q31 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q32 | temporal | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q33 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.58 |
| q34 | temporal | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q35 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q36 | temporal | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q37 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q38 | temporal | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q39 | temporal | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q40 | temporal | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q41 | temporal | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q42 | temporal | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.8 |
| q43 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.75 |
| q44 | temporal | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.69 |
| q45 | temporal | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.72 |
| q46 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.68 |
| q47 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q48 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q49 | temporal | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q50 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q51 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.76 |
| q52 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q53 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.66 |
| q54 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q55 | temporal | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q56 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.63 |
| q57 | multi-hop | INJECT | 0 | 1 | 6 | deep_rank_lt10 | 0.75 |
| q58 | temporal | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.75 |
| q59 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q60 | multi-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.67 |
| q61 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.66 |
| q62 | multi-hop | INJECT | 0 | 1 | 7 | deep_rank_lt10 | 0.74 |
| q63 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q64 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q65 | temporal | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q66 | temporal | INJECT | 0 | 0 |  | true_miss | 0.61 |
| q67 | open-domain | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q68 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q69 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q70 | multi-hop | INJECT | 0 | 1 | 6 | deep_rank_lt10 | 0.7 |
| q71 | multi-hop | INJECT | 0 | 1 | 7 | deep_rank_lt10 | 0.75 |
| q72 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.75 |
| q73 | temporal | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.77 |
| q74 | open-domain | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.58 |
| q75 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.74 |
| q76 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q77 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q78 | temporal | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q79 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q80 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.64 |
| q81 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q82 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q83 | multi-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.74 |
| q84 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.7 |
| q85 | open-domain | INJECT | 0 | 1 | 9 | deep_rank_lt10 | 0.73 |
| q86 | open-domain | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.68 |
| q87 | temporal | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q88 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.66 |
| q89 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q90 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q91 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.72 |
| q92 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q93 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q94 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q95 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.66 |
| q96 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.77 |
| q97 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q98 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.59 |
| q99 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.66 |
| q100 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q101 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q102 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.81 |
| q103 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q104 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.72 |
| q105 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.61 |
| q106 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q107 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q108 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.77 |
| q109 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q110 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q111 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q112 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.74 |
| q113 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q114 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q115 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.75 |
| q116 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.68 |
| q117 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q118 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.65 |
| q119 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q120 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q121 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q122 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q123 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.69 |
| q124 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.77 |
| q125 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.6 |
| q126 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.75 |
| q127 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.72 |
| q128 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q129 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q130 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q131 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.86 |
| q132 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q133 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q134 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q135 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q136 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q137 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.77 |
| q138 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q139 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.59 |
| q140 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q141 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q142 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.73 |
| q143 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q144 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q145 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q146 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q147 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.84 |
| q148 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.79 |
| q149 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q150 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.7 |
| q151 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q152 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q153 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q154 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q155 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q156 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.89 |
| q157 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.93 |
| q158 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.96 |
| q159 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q160 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.94 |
| q161 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q162 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q163 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q164 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q165 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q166 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q167 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.78 |
| q168 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q169 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.89 |
| q170 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.72 |
| q171 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q172 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.86 |
| q173 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q174 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.74 |
| q175 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q176 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q177 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q178 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q179 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q180 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q181 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.71 |
| q182 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.74 |
| q183 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.7 |
| q184 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.66 |
| q185 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q186 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.67 |
| q187 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.68 |
| q188 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.66 |
| q189 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q190 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q191 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q192 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q193 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q194 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q195 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.71 |
| q196 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.7 |
| q197 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q198 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q199 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q200 | adversarial | INJECT | 0 | 0 |  | abstention | 0.9 |
| q201 | adversarial | INJECT | 0 | 0 |  | abstention | 0.66 |
| q202 | adversarial | INJECT | 0 | 0 |  | abstention | 0.67 |
| q203 | adversarial | INJECT | 0 | 0 |  | abstention | 0.66 |
| q204 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q205 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q206 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q207 | adversarial | INJECT | 0 | 0 |  | abstention | 0.63 |
| q208 | adversarial | INJECT | 0 | 0 |  | abstention | 0.67 |
| q209 | adversarial | INJECT | 0 | 0 |  | abstention | 0.62 |
| q210 | adversarial | INJECT | 0 | 0 |  | abstention | 0.73 |
| q211 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q212 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q213 | adversarial | INJECT | 0 | 0 |  | abstention | 0.9 |
| q214 | adversarial | INJECT | 0 | 0 |  | abstention | 0.71 |
| q215 | adversarial | INJECT | 0 | 0 |  | abstention | 0.75 |
| q216 | adversarial | INJECT | 0 | 0 |  | abstention | 0.63 |
| q217 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q218 | adversarial | INJECT | 0 | 0 |  | abstention | 0.75 |
| q219 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q220 | adversarial | INJECT | 0 | 0 |  | abstention | 0.65 |
| q221 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q222 | adversarial | INJECT | 0 | 0 |  | abstention | 0.83 |
| q223 | adversarial | INJECT | 0 | 0 |  | abstention | 0.82 |
| q224 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q225 | adversarial | INJECT | 0 | 0 |  | abstention | 0.57 |
| q226 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q227 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q228 | adversarial | INJECT | 0 | 0 |  | abstention | 0.62 |
| q229 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q230 | adversarial | INJECT | 0 | 0 |  | abstention | 0.77 |
| q231 | adversarial | INJECT | 0 | 0 |  | abstention | 0.67 |
| q232 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q233 | adversarial | INJECT | 0 | 0 |  | abstention | 0.8 |
| q234 | adversarial | INJECT | 0 | 0 |  | abstention | 0.89 |
| q235 | adversarial | INJECT | 0 | 0 |  | abstention | 0.96 |
| q236 | adversarial | INJECT | 0 | 0 |  | abstention | 0.93 |
| q237 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q238 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q239 | adversarial | INJECT | 0 | 0 |  | abstention | 0.89 |
| q240 | adversarial | INJECT | 0 | 0 |  | abstention | 0.78 |
| q241 | adversarial | INJECT | 0 | 0 |  | abstention | 0.88 |
| q242 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q243 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q244 | adversarial | INJECT | 0 | 0 |  | abstention | 0.85 |
| q245 | adversarial | INJECT | 0 | 0 |  | abstention | 0.6 |
| q246 | adversarial | INJECT | 0 | 0 |  | abstention | 0.75 |
| q247 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q248 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q249 | adversarial | INJECT | 0 | 0 |  | abstention | 0.73 |
| q250 | adversarial | INJECT | 0 | 0 |  | abstention | 0.62 |
| q251 | adversarial | INJECT | 0 | 0 |  | abstention | 0.76 |
| q252 | adversarial | INJECT | 0 | 0 |  | abstention | 0.62 |
| q253 | adversarial | INJECT | 0 | 0 |  | abstention | 0.66 |
| q254 | adversarial | INJECT | 0 | 0 |  | abstention | 0.6 |
| q255 | adversarial | INJECT | 0 | 0 |  | abstention | 0.83 |
| q256 | adversarial | INJECT | 0 | 0 |  | abstention | 0.65 |
| q257 | adversarial | INJECT | 0 | 0 |  | abstention | 0.82 |
| q258 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q259 | adversarial | INJECT | 0 | 0 |  | abstention | 0.61 |
| q260 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
