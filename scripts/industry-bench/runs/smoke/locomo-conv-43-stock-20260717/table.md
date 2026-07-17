# smoke conv-43

- questions: 242
- route: stock
- calibration: stock
- evidence@5 (answerable): 0.93
- evidence@10 (answerable): 0.96
- retrieved_context_tokens mean: 225.9
- answer_path_tokens mean: 257.4

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q2 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q3 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q4 | open-domain | STOCK | 0 | 0 |  | true_miss | 0.69 |
| q5 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q6 | open-domain | STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.72 |
| q7 | temporal | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q8 | multi-hop | STOCK | 0 | 0 |  | true_miss | 0.65 |
| q9 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q10 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.58 |
| q11 | temporal | STOCK | 1 | 1 | 0 | hit | 0.58 |
| q12 | multi-hop | STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.7 |
| q13 | temporal | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q14 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q15 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q16 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.46 |
| q17 | temporal | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q18 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q19 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q20 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.6 |
| q21 | temporal | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.57 |
| q22 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q23 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.62 |
| q24 | temporal | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q25 | temporal | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.5 |
| q26 | multi-hop | STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.64 |
| q27 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q28 | open-domain | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q29 | open-domain | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q30 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.89 |
| q31 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q32 | temporal | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q33 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q34 | temporal | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q35 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q36 | multi-hop | STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.62 |
| q37 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q38 | temporal | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q39 | multi-hop | STOCK | 0 | 1 | 8 | deep_rank_lt10 | 0.73 |
| q40 | temporal | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q41 | temporal | STOCK | 0 | 0 |  | true_miss | 0.59 |
| q42 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.78 |
| q43 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.53 |
| q44 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q45 | temporal | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q46 | temporal | STOCK | 0 | 0 |  | true_miss | 0.65 |
| q47 | temporal | STOCK | 1 | 1 | 0 | hit | 0.61 |
| q48 | temporal | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q49 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.88 |
| q50 | multi-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q51 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q52 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q53 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q54 | open-domain | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q55 | temporal | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q56 | temporal | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q57 | temporal | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q58 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q59 | temporal | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q60 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.56 |
| q61 | temporal | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q62 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q63 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q64 | temporal | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q65 | temporal | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q66 | multi-hop | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q67 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q68 | open-domain | STOCK | 0 | 1 | 8 | deep_rank_lt10 | 0.73 |
| q69 | temporal | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q70 | temporal | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q71 | open-domain | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q72 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q73 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q74 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.95 |
| q75 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.93 |
| q76 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q77 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q78 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q79 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q80 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q81 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.62 |
| q82 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q83 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.58 |
| q84 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.89 |
| q85 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q86 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q87 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.63 |
| q88 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.62 |
| q89 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.71 |
| q90 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q91 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.6 |
| q92 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q93 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q94 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q95 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.7 |
| q96 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.68 |
| q97 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q98 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.94 |
| q99 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.76 |
| q100 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q101 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q102 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q103 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q104 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q105 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q106 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q107 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q108 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q109 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q110 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q111 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q112 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q113 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.65 |
| q114 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.59 |
| q115 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q116 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.66 |
| q117 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q118 | single-hop | STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.67 |
| q119 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q120 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q121 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q122 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q123 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q124 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.79 |
| q125 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q126 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q127 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q128 | single-hop | STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.6 |
| q129 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q130 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.9 |
| q131 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.73 |
| q132 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.85 |
| q133 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q134 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q135 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q136 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.89 |
| q137 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.88 |
| q138 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.74 |
| q139 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q140 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.75 |
| q141 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q142 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.92 |
| q143 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.66 |
| q144 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q145 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q146 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.57 |
| q147 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.92 |
| q148 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.82 |
| q149 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q150 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q151 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q152 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.61 |
| q153 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q154 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.84 |
| q155 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q156 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q157 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q158 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q159 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.87 |
| q160 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.63 |
| q161 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q162 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.77 |
| q163 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.81 |
| q164 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.64 |
| q165 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q166 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.8 |
| q167 | single-hop | STOCK | 0 | 0 |  | true_miss | 0.62 |
| q168 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q169 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q170 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.79 |
| q171 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q172 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.67 |
| q173 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.72 |
| q174 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.83 |
| q175 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.71 |
| q176 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.68 |
| q177 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.78 |
| q178 | single-hop | STOCK | 1 | 1 | 0 | hit | 0.69 |
| q179 | adversarial | STOCK | 0 | 0 |  | abstention | 0.68 |
| q180 | adversarial | STOCK | 0 | 0 |  | abstention | 0.68 |
| q181 | adversarial | STOCK | 0 | 0 |  | abstention | 0.95 |
| q182 | adversarial | STOCK | 0 | 0 |  | abstention | 0.57 |
| q183 | adversarial | STOCK | 0 | 0 |  | abstention | 0.79 |
| q184 | adversarial | STOCK | 0 | 0 |  | abstention | 0.65 |
| q185 | adversarial | STOCK | 0 | 0 |  | abstention | 0.83 |
| q186 | adversarial | STOCK | 0 | 0 |  | abstention | 0.59 |
| q187 | adversarial | STOCK | 0 | 0 |  | abstention | 0.89 |
| q188 | adversarial | STOCK | 0 | 0 |  | abstention | 0.68 |
| q189 | adversarial | STOCK | 0 | 0 |  | abstention | 0.62 |
| q190 | adversarial | STOCK | 0 | 0 |  | abstention | 0.73 |
| q191 | adversarial | STOCK | 0 | 0 |  | abstention | 0.68 |
| q192 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q193 | adversarial | STOCK | 0 | 0 |  | abstention | 0.76 |
| q194 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q195 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q196 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q197 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q198 | adversarial | STOCK | 0 | 0 |  | abstention | 0.77 |
| q199 | adversarial | STOCK | 0 | 0 |  | abstention | 0.58 |
| q200 | adversarial | STOCK | 0 | 0 |  | abstention | 0.83 |
| q201 | adversarial | STOCK | 0 | 0 |  | abstention | 0.72 |
| q202 | adversarial | STOCK | 0 | 0 |  | abstention | 0.84 |
| q203 | adversarial | STOCK | 0 | 0 |  | abstention | 0.65 |
| q204 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
| q205 | adversarial | STOCK | 0 | 0 |  | abstention | 0.73 |
| q206 | adversarial | STOCK | 0 | 0 |  | abstention | 0.61 |
| q207 | adversarial | STOCK | 0 | 0 |  | abstention | 0.7 |
| q208 | adversarial | STOCK | 0 | 0 |  | abstention | 0.79 |
| q209 | adversarial | STOCK | 0 | 0 |  | abstention | 0.8 |
| q210 | adversarial | STOCK | 0 | 0 |  | abstention | 0.83 |
| q211 | adversarial | STOCK | 0 | 0 |  | abstention | 0.74 |
| q212 | adversarial | STOCK | 0 | 0 |  | abstention | 0.79 |
| q213 | adversarial | STOCK | 0 | 0 |  | abstention | 0.71 |
| q214 | adversarial | STOCK | 0 | 0 |  | abstention | 0.82 |
| q215 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q216 | adversarial | STOCK | 0 | 0 |  | abstention | 0.91 |
| q217 | adversarial | STOCK | 0 | 0 |  | abstention | 0.9 |
| q218 | adversarial | STOCK | 0 | 0 |  | abstention | 0.55 |
| q219 | adversarial | STOCK | 0 | 0 |  | abstention | 0.64 |
| q220 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q221 | adversarial | STOCK | 0 | 0 |  | abstention | 0.71 |
| q222 | adversarial | STOCK | 0 | 0 |  | abstention | 0.66 |
| q223 | adversarial | STOCK | 0 | 0 |  | abstention | 0.76 |
| q224 | adversarial | STOCK | 0 | 0 |  | abstention | 0.91 |
| q225 | adversarial | STOCK | 0 | 0 |  | abstention | 0.83 |
| q226 | adversarial | STOCK | 0 | 0 |  | abstention | 0.79 |
| q227 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q228 | adversarial | STOCK | 0 | 0 |  | abstention | 0.77 |
| q229 | adversarial | STOCK | 0 | 0 |  | abstention | 0.83 |
| q230 | adversarial | STOCK | 0 | 0 |  | abstention | 0.69 |
| q231 | adversarial | STOCK | 0 | 0 |  | abstention | 0.88 |
| q232 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q233 | adversarial | STOCK | 0 | 0 |  | abstention | 0.76 |
| q234 | adversarial | STOCK | 0 | 0 |  | abstention | 0.79 |
| q235 | adversarial | STOCK | 0 | 0 |  | abstention | 0.63 |
| q236 | adversarial | STOCK | 0 | 0 |  | abstention | 0.78 |
| q237 | adversarial | STOCK | 0 | 0 |  | abstention | 0.67 |
| q238 | adversarial | STOCK | 0 | 0 |  | abstention | 0.78 |
| q239 | adversarial | STOCK | 0 | 0 |  | abstention | 0.53 |
| q240 | adversarial | STOCK | 0 | 0 |  | abstention | 0.83 |
| q241 | adversarial | STOCK | 0 | 0 |  | abstention | 0.73 |
| q242 | adversarial | STOCK | 0 | 0 |  | abstention | 0.79 |
