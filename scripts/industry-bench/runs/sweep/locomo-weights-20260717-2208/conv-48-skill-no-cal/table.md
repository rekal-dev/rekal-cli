# smoke conv-48

- questions: 239
- route: skill
- calibration: stock
- evidence@5 (answerable): 0.91
- evidence@10 (answerable): 0.92
- retrieved_context_tokens mean: 150.8
- answer_path_tokens mean: 181.2

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | temporal | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.56 |
| q2 | multi-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.58 |
| q3 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.79 |
| q4 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.79 |
| q5 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.85 |
| q6 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.86 |
| q7 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.86 |
| q8 | temporal | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q9 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.67 |
| q10 | temporal | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.77 |
| q11 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.65 |
| q12 | open-domain | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q13 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.62 |
| q14 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.68 |
| q15 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q16 | single-hop | FALLBACK_STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.56 |
| q17 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q18 | temporal | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q19 | open-domain | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.59 |
| q20 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.78 |
| q21 | multi-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q22 | temporal | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q23 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.67 |
| q24 | open-domain | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.51 |
| q25 | multi-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.59 |
| q26 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.59 |
| q27 | temporal | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.66 |
| q28 | temporal | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.47 |
| q29 | open-domain | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.73 |
| q30 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.58 |
| q31 | temporal | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q32 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.81 |
| q33 | temporal | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.49 |
| q34 | single-hop | FALLBACK_STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.49 |
| q35 | temporal | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.49 |
| q36 | temporal | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q37 | open-domain | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q38 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.6 |
| q39 | temporal | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q40 | multi-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q41 | open-domain | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.48 |
| q42 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q43 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q44 | temporal | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.56 |
| q45 | temporal | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q46 | temporal | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.6 |
| q47 | temporal | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q48 | temporal | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.42 |
| q49 | open-domain | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.57 |
| q50 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q51 | temporal | FALLBACK_STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.52 |
| q52 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.64 |
| q53 | temporal | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.66 |
| q54 | temporal | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q55 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q56 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q57 | temporal | FALLBACK_STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.67 |
| q58 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q59 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.62 |
| q60 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.56 |
| q61 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.58 |
| q62 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.43 |
| q63 | single-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.66 |
| q64 | single-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.51 |
| q65 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q66 | temporal | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.53 |
| q67 | temporal | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q68 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q69 | temporal | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.53 |
| q70 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q71 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q72 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.62 |
| q73 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.62 |
| q74 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q75 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q76 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q77 | temporal | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q78 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q79 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.46 |
| q80 | multi-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.56 |
| q81 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.57 |
| q82 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.56 |
| q83 | multi-hop | FALLBACK_STOCK | 0 | 1 | 9 | deep_rank_lt10 | 0.51 |
| q84 | multi-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.56 |
| q85 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.67 |
| q86 | multi-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.52 |
| q87 | multi-hop | FALLBACK_STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.7 |
| q88 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.71 |
| q89 | multi-hop | FALLBACK_STOCK | 0 | 1 | 6 | deep_rank_lt10 | 0.53 |
| q90 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q91 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.59 |
| q92 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q93 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.62 |
| q94 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.43 |
| q95 | single-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.66 |
| q96 | single-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.51 |
| q97 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q98 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q99 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.62 |
| q100 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q101 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q102 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.58 |
| q103 | single-hop | FALLBACK_STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.59 |
| q104 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.79 |
| q105 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q106 | single-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.54 |
| q107 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.59 |
| q108 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.62 |
| q109 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q110 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q111 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.86 |
| q112 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q113 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.61 |
| q114 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q115 | single-hop | FALLBACK_STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.56 |
| q116 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.93 |
| q117 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.83 |
| q118 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.67 |
| q119 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q120 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q121 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q122 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.44 |
| q123 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q124 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q125 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.67 |
| q126 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q127 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q128 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.66 |
| q129 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q130 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.55 |
| q131 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q132 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.61 |
| q133 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.61 |
| q134 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.46 |
| q135 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q136 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.61 |
| q137 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.69 |
| q138 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.65 |
| q139 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q140 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q141 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.74 |
| q142 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.51 |
| q143 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q144 | single-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.65 |
| q145 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.68 |
| q146 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.74 |
| q147 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.56 |
| q148 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q149 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.91 |
| q150 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q151 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q152 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q153 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q154 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q155 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q156 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q157 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q158 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.59 |
| q159 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.54 |
| q160 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.86 |
| q161 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.56 |
| q162 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q163 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.86 |
| q164 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q165 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.5 |
| q166 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q167 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q168 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.47 |
| q169 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.48 |
| q170 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.51 |
| q171 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.53 |
| q172 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q173 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.93 |
| q174 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q175 | single-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.67 |
| q176 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q177 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q178 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q179 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.54 |
| q180 | single-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.52 |
| q181 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.62 |
| q182 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.61 |
| q183 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.53 |
| q184 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.55 |
| q185 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.56 |
| q186 | single-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.8 |
| q187 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.57 |
| q188 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q189 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.59 |
| q190 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q191 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q192 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.67 |
| q193 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.62 |
| q194 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.48 |
| q195 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.51 |
| q196 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.63 |
| q197 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.58 |
| q198 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.65 |
| q199 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.62 |
| q200 | adversarial | INJECT | 0 | 0 |  | abstention | 0.84 |
| q201 | adversarial | INJECT | 0 | 0 |  | abstention | 0.9 |
| q202 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.59 |
| q203 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.66 |
| q204 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.56 |
| q205 | adversarial | INJECT | 0 | 0 |  | abstention | 0.92 |
| q206 | adversarial | INJECT | 0 | 0 |  | abstention | 0.84 |
| q207 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.52 |
| q208 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.54 |
| q209 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.55 |
| q210 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.48 |
| q211 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.59 |
| q212 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.47 |
| q213 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.6 |
| q214 | adversarial | INJECT | 0 | 0 |  | abstention | 0.68 |
| q215 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.62 |
| q216 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.68 |
| q217 | adversarial | INJECT | 0 | 0 |  | abstention | 0.85 |
| q218 | adversarial | INJECT | 0 | 0 |  | abstention | 0.9 |
| q219 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.6 |
| q220 | adversarial | INJECT | 0 | 0 |  | abstention | 0.84 |
| q221 | adversarial | INJECT | 0 | 0 |  | abstention | 0.77 |
| q222 | adversarial | INJECT | 0 | 0 |  | abstention | 0.73 |
| q223 | adversarial | INJECT | 0 | 0 |  | abstention | 0.85 |
| q224 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.52 |
| q225 | adversarial | INJECT | 0 | 0 |  | abstention | 0.8 |
| q226 | adversarial | INJECT | 0 | 0 |  | abstention | 0.87 |
| q227 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.7 |
| q228 | adversarial | INJECT | 0 | 0 |  | abstention | 0.9 |
| q229 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.46 |
| q230 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.53 |
| q231 | adversarial | INJECT | 0 | 0 |  | abstention | 0.82 |
| q232 | adversarial | INJECT | 0 | 0 |  | abstention | 0.79 |
| q233 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.69 |
| q234 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.65 |
| q235 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.61 |
| q236 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.57 |
| q237 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.57 |
| q238 | adversarial | INJECT | 0 | 0 |  | abstention | 0.86 |
| q239 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
