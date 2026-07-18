# smoke conv-42

- questions: 260
- route: skill-multi
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 0.90
- evidence@10 (answerable): 0.96
- retrieved_context_tokens mean: 521.9
- answer_path_tokens mean: 549.5

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | open-domain | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.67 |
| q2 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.51 |
| q3 | temporal | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.51 |
| q4 | temporal | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q5 | open-domain | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.5 |
| q6 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q7 | temporal | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.59 |
| q8 | temporal | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.62 |
| q9 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.56 |
| q10 | temporal | FALLBACK_STOCK | 0 | 0 | 15 | gate_blocked | 0.51 |
| q11 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.76 |
| q12 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q13 | open-domain | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q14 | temporal | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q15 | open-domain | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.54 |
| q16 | temporal | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q17 | temporal | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q18 | temporal | INJECT | 0 | 0 | 15 | deep_rank_gte10 | 0.57 |
| q19 | temporal | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q20 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q21 | temporal | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q22 | temporal | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q23 | temporal | INJECT | 0 | 1 | 5 | deep_rank_lt10 | 0.66 |
| q24 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.86 |
| q25 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.83 |
| q26 | temporal | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q27 | temporal | FALLBACK_STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.44 |
| q28 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q29 | temporal | INJECT | 1 | 1 | 0 | hit | 0.88 |
| q30 | temporal | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q31 | multi-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.49 |
| q32 | temporal | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.66 |
| q33 | temporal | FALLBACK_STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.51 |
| q34 | temporal | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q35 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q36 | temporal | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q37 | temporal | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q38 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q39 | temporal | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q40 | temporal | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q41 | temporal | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q42 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.8 |
| q43 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q44 | temporal | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.52 |
| q45 | temporal | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q46 | temporal | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.52 |
| q47 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q48 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q49 | temporal | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q50 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.47 |
| q51 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.76 |
| q52 | multi-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.54 |
| q53 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.57 |
| q54 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.59 |
| q55 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q56 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.39 |
| q57 | multi-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.61 |
| q58 | temporal | FALLBACK_STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.52 |
| q59 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.56 |
| q60 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.62 |
| q61 | open-domain | FALLBACK_STOCK | 0 | 1 | 9 | deep_rank_lt10 | 0.45 |
| q62 | multi-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.54 |
| q63 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.56 |
| q64 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q65 | temporal | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.54 |
| q66 | temporal | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.54 |
| q67 | open-domain | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.53 |
| q68 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.46 |
| q69 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q70 | multi-hop | FALLBACK_STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.47 |
| q71 | multi-hop | FALLBACK_STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.47 |
| q72 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q73 | temporal | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.56 |
| q74 | open-domain | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.57 |
| q75 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q76 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q77 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q78 | temporal | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q79 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q80 | multi-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.48 |
| q81 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.54 |
| q82 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.56 |
| q83 | multi-hop | INJECT | 0 | 1 | 8 | deep_rank_lt10 | 0.55 |
| q84 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.5 |
| q85 | open-domain | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.72 |
| q86 | open-domain | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.68 |
| q87 | temporal | INJECT | 1 | 1 | 0 | hit | 0.56 |
| q88 | open-domain | INJECT | 0 | 1 | 8 | deep_rank_lt10 | 0.57 |
| q89 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q90 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q91 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.53 |
| q92 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.62 |
| q93 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q94 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q95 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.41 |
| q96 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.61 |
| q97 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q98 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.54 |
| q99 | single-hop | FALLBACK_STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.46 |
| q100 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q101 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.42 |
| q102 | single-hop | INJECT | 0 | 1 | 5 | deep_rank_lt10 | 0.81 |
| q103 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q104 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q105 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.59 |
| q106 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q107 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.46 |
| q108 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.56 |
| q109 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.58 |
| q110 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.57 |
| q111 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q112 | single-hop | INJECT | 0 | 1 | 8 | deep_rank_lt10 | 0.69 |
| q113 | single-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.52 |
| q114 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.74 |
| q115 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.6 |
| q116 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.55 |
| q117 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.48 |
| q118 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q119 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q120 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q121 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q122 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q123 | single-hop | FALLBACK_STOCK | 0 | 1 | 9 | deep_rank_lt10 | 0.54 |
| q124 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.68 |
| q125 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.57 |
| q126 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q127 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q128 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q129 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.61 |
| q130 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q131 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.86 |
| q132 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q133 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q134 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q135 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.63 |
| q136 | single-hop | INJECT | 0 | 1 | 6 | deep_rank_lt10 | 0.5 |
| q137 | single-hop | INJECT | 0 | 1 | 6 | deep_rank_lt10 | 0.77 |
| q138 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q139 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q140 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q141 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.51 |
| q142 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q143 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.55 |
| q144 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q145 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q146 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.58 |
| q147 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.84 |
| q148 | single-hop | INJECT | 0 | 0 | 11 | deep_rank_gte10 | 0.74 |
| q149 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q150 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.59 |
| q151 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q152 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
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
| q163 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q164 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q165 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q166 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q167 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.78 |
| q168 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q169 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.89 |
| q170 | single-hop | INJECT | 0 | 0 | 15 | deep_rank_gte10 | 0.6 |
| q171 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q172 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.86 |
| q173 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q174 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q175 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.68 |
| q176 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q177 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.86 |
| q178 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q179 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q180 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q181 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.59 |
| q182 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.74 |
| q183 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q184 | single-hop | INJECT | 0 | 0 | 10 | deep_rank_gte10 | 0.51 |
| q185 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q186 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.55 |
| q187 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.51 |
| q188 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q189 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.55 |
| q190 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q191 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q192 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q193 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q194 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q195 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q196 | single-hop | INJECT | 0 | 0 | 15 | deep_rank_gte10 | 0.68 |
| q197 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.61 |
| q198 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.61 |
| q199 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.51 |
| q200 | adversarial | INJECT | 0 | 0 |  | abstention | 0.9 |
| q201 | adversarial | INJECT | 0 | 0 |  | abstention | 0.51 |
| q202 | adversarial | INJECT | 0 | 0 |  | abstention | 0.64 |
| q203 | adversarial | INJECT | 0 | 0 |  | abstention | 0.66 |
| q204 | adversarial | INJECT | 0 | 0 |  | abstention | 0.66 |
| q205 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.41 |
| q206 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q207 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.48 |
| q208 | adversarial | INJECT | 0 | 0 |  | abstention | 0.67 |
| q209 | adversarial | INJECT | 0 | 0 |  | abstention | 0.5 |
| q210 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q211 | adversarial | INJECT | 0 | 0 |  | abstention | 0.51 |
| q212 | adversarial | INJECT | 0 | 0 |  | abstention | 0.57 |
| q213 | adversarial | INJECT | 0 | 0 |  | abstention | 0.9 |
| q214 | adversarial | INJECT | 0 | 0 |  | abstention | 0.71 |
| q215 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.52 |
| q216 | adversarial | INJECT | 0 | 0 |  | abstention | 0.55 |
| q217 | adversarial | INJECT | 0 | 0 |  | abstention | 0.61 |
| q218 | adversarial | INJECT | 0 | 0 |  | abstention | 0.75 |
| q219 | adversarial | INJECT | 0 | 0 |  | abstention | 0.66 |
| q220 | adversarial | INJECT | 0 | 0 |  | abstention | 0.61 |
| q221 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q222 | adversarial | INJECT | 0 | 0 |  | abstention | 0.83 |
| q223 | adversarial | INJECT | 0 | 0 |  | abstention | 0.82 |
| q224 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q225 | adversarial | INJECT | 0 | 0 |  | abstention | 0.47 |
| q226 | adversarial | INJECT | 0 | 0 |  | abstention | 0.63 |
| q227 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q228 | adversarial | INJECT | 0 | 0 |  | abstention | 0.58 |
| q229 | adversarial | INJECT | 0 | 0 |  | abstention | 0.58 |
| q230 | adversarial | INJECT | 0 | 0 |  | abstention | 0.67 |
| q231 | adversarial | INJECT | 0 | 0 |  | abstention | 0.63 |
| q232 | adversarial | INJECT | 0 | 0 |  | abstention | 0.67 |
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
| q243 | adversarial | INJECT | 0 | 0 |  | abstention | 0.62 |
| q244 | adversarial | INJECT | 0 | 0 |  | abstention | 0.85 |
| q245 | adversarial | INJECT | 0 | 0 |  | abstention | 0.67 |
| q246 | adversarial | INJECT | 0 | 0 |  | abstention | 0.75 |
| q247 | adversarial | INJECT | 0 | 0 |  | abstention | 0.64 |
| q248 | adversarial | INJECT | 0 | 0 |  | abstention | 0.59 |
| q249 | adversarial | INJECT | 0 | 0 |  | abstention | 0.73 |
| q250 | adversarial | INJECT | 0 | 0 |  | abstention | 0.49 |
| q251 | adversarial | INJECT | 0 | 0 |  | abstention | 0.76 |
| q252 | adversarial | INJECT | 0 | 0 |  | abstention | 0.62 |
| q253 | adversarial | INJECT | 0 | 0 |  | abstention | 0.63 |
| q254 | adversarial | INJECT | 0 | 0 |  | abstention | 0.54 |
| q255 | adversarial | INJECT | 0 | 0 |  | abstention | 0.83 |
| q256 | adversarial | INJECT | 0 | 0 |  | abstention | 0.53 |
| q257 | adversarial | INJECT | 0 | 0 |  | abstention | 0.64 |
| q258 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.49 |
| q259 | adversarial | INJECT | 0 | 0 |  | abstention | 0.61 |
| q260 | adversarial | INJECT | 0 | 0 |  | abstention | 0.61 |
