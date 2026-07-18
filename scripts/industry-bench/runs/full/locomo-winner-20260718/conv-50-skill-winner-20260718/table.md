# smoke conv-50

- questions: 204
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 0.87
- evidence@10 (answerable): 0.89
- retrieved_context_tokens mean: 201.4
- answer_path_tokens mean: 236.5

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | temporal | INJECT | 1 | 1 | 0 | hit | 0.56 |
| q2 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.62 |
| q3 | temporal | INJECT | 1 | 1 | 0 | hit | 0.56 |
| q4 | multi-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.54 |
| q5 | open-domain | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.48 |
| q6 | multi-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.41 |
| q7 | multi-hop | INJECT | 0 | 1 | 5 | deep_rank_lt10 | 0.62 |
| q8 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.6 |
| q9 | temporal | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q10 | temporal | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q11 | temporal | INJECT | 1 | 1 | 0 | hit | 0.58 |
| q12 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.62 |
| q13 | temporal | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.57 |
| q14 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.56 |
| q15 | temporal | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q16 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.62 |
| q17 | temporal | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q18 | temporal | INJECT | 0 | 0 |  | true_miss | 0.59 |
| q19 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q20 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.59 |
| q21 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.58 |
| q22 | open-domain | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.48 |
| q23 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q24 | multi-hop | FALLBACK_STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.43 |
| q25 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.5 |
| q26 | temporal | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q27 | temporal | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q28 | temporal | INJECT | 0 | 0 |  | true_miss | 0.65 |
| q29 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.43 |
| q30 | multi-hop | INJECT | 0 | 1 | 6 | deep_rank_lt10 | 0.63 |
| q31 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.53 |
| q32 | temporal | FALLBACK_STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.42 |
| q33 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q34 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q35 | temporal | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q36 | temporal | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q37 | temporal | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q38 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.52 |
| q39 | temporal | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q40 | open-domain | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.46 |
| q41 | multi-hop | INJECT | 0 | 0 |  | true_miss | 0.57 |
| q42 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.55 |
| q43 | open-domain | INJECT | 0 | 0 |  | abstention | 0.63 |
| q44 | temporal | INJECT | 0 | 0 |  | true_miss | 0.55 |
| q45 | temporal | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q46 | multi-hop | INJECT | 0 | 0 |  | true_miss | 0.68 |
| q47 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.6 |
| q48 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.57 |
| q49 | temporal | INJECT | 0 | 0 |  | true_miss | 0.6 |
| q50 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.52 |
| q51 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q52 | temporal | INJECT | 1 | 1 | 0 | hit | 0.59 |
| q53 | temporal | INJECT | 0 | 0 |  | true_miss | 0.54 |
| q54 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.52 |
| q55 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q56 | temporal | INJECT | 1 | 1 | 0 | hit | 0.91 |
| q57 | multi-hop | INJECT | 0 | 0 |  | true_miss | 0.65 |
| q58 | temporal | INJECT | 0 | 0 |  | true_miss | 0.6 |
| q59 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q60 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.51 |
| q61 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q62 | temporal | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q63 | temporal | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q64 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q65 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.56 |
| q66 | temporal | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q67 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q68 | temporal | INJECT | 0 | 0 |  | true_miss | 0.55 |
| q69 | temporal | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q70 | temporal | INJECT | 1 | 1 | 0 | hit | 0.61 |
| q71 | temporal | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q72 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q73 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q74 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.52 |
| q75 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q76 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.59 |
| q77 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q78 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q79 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.56 |
| q80 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.59 |
| q81 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q82 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q83 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.43 |
| q84 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q85 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q86 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.5 |
| q87 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.72 |
| q88 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.55 |
| q89 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q90 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q91 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.89 |
| q92 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.57 |
| q93 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q94 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.62 |
| q95 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q96 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q97 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.56 |
| q98 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q99 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q100 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q101 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q102 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.59 |
| q103 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.86 |
| q104 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q105 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q106 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q107 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q108 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q109 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q110 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.5 |
| q111 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.57 |
| q112 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q113 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q114 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.6 |
| q115 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q116 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.52 |
| q117 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.61 |
| q118 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.61 |
| q119 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q120 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.4 |
| q121 | single-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.45 |
| q122 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.53 |
| q123 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q124 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q125 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q126 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q127 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q128 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q129 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.58 |
| q130 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q131 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q132 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.61 |
| q133 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.61 |
| q134 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q135 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.88 |
| q136 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q137 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q138 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.55 |
| q139 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q140 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.48 |
| q141 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.68 |
| q142 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.58 |
| q143 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.5 |
| q144 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q145 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.58 |
| q146 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q147 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.59 |
| q148 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q149 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q150 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q151 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q152 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.41 |
| q153 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.45 |
| q154 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.74 |
| q155 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.6 |
| q156 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q157 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q158 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q159 | adversarial | INJECT | 0 | 0 |  | abstention | 0.59 |
| q160 | adversarial | INJECT | 0 | 0 |  | abstention | 0.62 |
| q161 | adversarial | INJECT | 0 | 0 |  | abstention | 0.55 |
| q162 | adversarial | INJECT | 0 | 0 |  | abstention | 0.67 |
| q163 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q164 | adversarial | INJECT | 0 | 0 |  | abstention | 0.58 |
| q165 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.44 |
| q166 | adversarial | INJECT | 0 | 0 |  | abstention | 0.81 |
| q167 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q168 | adversarial | INJECT | 0 | 0 |  | abstention | 0.81 |
| q169 | adversarial | INJECT | 0 | 0 |  | abstention | 0.57 |
| q170 | adversarial | INJECT | 0 | 0 |  | abstention | 0.67 |
| q171 | adversarial | INJECT | 0 | 0 |  | abstention | 0.78 |
| q172 | adversarial | INJECT | 0 | 0 |  | abstention | 0.56 |
| q173 | adversarial | INJECT | 0 | 0 |  | abstention | 0.6 |
| q174 | adversarial | INJECT | 0 | 0 |  | abstention | 0.86 |
| q175 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q176 | adversarial | INJECT | 0 | 0 |  | abstention | 0.85 |
| q177 | adversarial | INJECT | 0 | 0 |  | abstention | 0.71 |
| q178 | adversarial | INJECT | 0 | 0 |  | abstention | 0.77 |
| q179 | adversarial | INJECT | 0 | 0 |  | abstention | 0.78 |
| q180 | adversarial | INJECT | 0 | 0 |  | abstention | 0.79 |
| q181 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q182 | adversarial | INJECT | 0 | 0 |  | abstention | 0.84 |
| q183 | adversarial | INJECT | 0 | 0 |  | abstention | 0.61 |
| q184 | adversarial | INJECT | 0 | 0 |  | abstention | 0.68 |
| q185 | adversarial | INJECT | 0 | 0 |  | abstention | 0.53 |
| q186 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.43 |
| q187 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.49 |
| q188 | adversarial | INJECT | 0 | 0 |  | abstention | 0.75 |
| q189 | adversarial | INJECT | 0 | 0 |  | abstention | 0.61 |
| q190 | adversarial | INJECT | 0 | 0 |  | abstention | 0.77 |
| q191 | adversarial | INJECT | 0 | 0 |  | abstention | 0.65 |
| q192 | adversarial | INJECT | 0 | 0 |  | abstention | 0.51 |
| q193 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.48 |
| q194 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.51 |
| q195 | adversarial | INJECT | 0 | 0 |  | abstention | 0.67 |
| q196 | adversarial | INJECT | 0 | 0 |  | abstention | 0.63 |
| q197 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.47 |
| q198 | adversarial | INJECT | 0 | 0 |  | abstention | 0.79 |
| q199 | adversarial | INJECT | 0 | 0 |  | abstention | 0.6 |
| q200 | adversarial | INJECT | 0 | 0 |  | abstention | 0.65 |
| q201 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.46 |
| q202 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q203 | adversarial | INJECT | 0 | 0 |  | abstention | 0.63 |
| q204 | adversarial | INJECT | 0 | 0 |  | abstention | 0.56 |
