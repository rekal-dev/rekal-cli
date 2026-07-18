# smoke conv-41

- questions: 193
- route: skill
- calibration: locomo-multi-hop.json
- evidence@5 (answerable): 0.82
- evidence@10 (answerable): 0.85
- retrieved_context_tokens mean: 181.6
- answer_path_tokens mean: 213.2

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | temporal | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.76 |
| q2 | temporal | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q3 | multi-hop | INJECT | 0 | 1 | 9 | deep_rank_lt10 | 0.78 |
| q4 | multi-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.49 |
| q5 | temporal | INJECT | 1 | 1 | 0 | hit | 0.58 |
| q6 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q7 | multi-hop | INJECT | 0 | 0 |  | true_miss | 0.55 |
| q8 | multi-hop | INJECT | 0 | 1 | 9 | deep_rank_lt10 | 0.63 |
| q9 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.65 |
| q10 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q11 | temporal | INJECT | 0 | 0 |  | true_miss | 0.59 |
| q12 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q13 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q14 | temporal | INJECT | 1 | 1 | 0 | hit | 0.54 |
| q15 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.59 |
| q16 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q17 | temporal | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.56 |
| q18 | open-domain | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q19 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.52 |
| q20 | multi-hop | INJECT | 0 | 1 | 6 | deep_rank_lt10 | 0.73 |
| q21 | temporal | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q22 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.61 |
| q23 | temporal | INJECT | 0 | 0 |  | true_miss | 0.56 |
| q24 | multi-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.45 |
| q25 | temporal | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q26 | multi-hop | INJECT | 0 | 0 |  | true_miss | 0.56 |
| q27 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q28 | temporal | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q29 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q30 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q31 | multi-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.57 |
| q32 | temporal | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q33 | multi-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.47 |
| q34 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.51 |
| q35 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q36 | multi-hop | INJECT | 0 | 0 |  | true_miss | 0.56 |
| q37 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q38 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q39 | temporal | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.47 |
| q40 | open-domain | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.49 |
| q41 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.6 |
| q42 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.54 |
| q43 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q44 | temporal | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q45 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q46 | open-domain | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.7 |
| q47 | temporal | INJECT | 1 | 1 | 0 | hit | 0.58 |
| q48 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.53 |
| q49 | temporal | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q50 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q51 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.56 |
| q52 | temporal | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.75 |
| q53 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q54 | temporal | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q55 | temporal | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q56 | temporal | INJECT | 0 | 0 |  | true_miss | 0.62 |
| q57 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q58 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q59 | temporal | INJECT | 0 | 0 |  | true_miss | 0.56 |
| q60 | temporal | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q61 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.55 |
| q62 | temporal | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q63 | multi-hop | INJECT | 0 | 1 | 5 | deep_rank_lt10 | 0.69 |
| q64 | temporal | INJECT | 1 | 1 | 0 | hit | 0.53 |
| q65 | open-domain | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.49 |
| q66 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q67 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.9 |
| q68 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.88 |
| q69 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q70 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.54 |
| q71 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q72 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.49 |
| q73 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.61 |
| q74 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q75 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.63 |
| q76 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.45 |
| q77 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q78 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q79 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.46 |
| q80 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.61 |
| q81 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q82 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q83 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.56 |
| q84 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q85 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q86 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.48 |
| q87 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q88 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q89 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.46 |
| q90 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q91 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q92 | temporal | INJECT | 0 | 0 |  | true_miss | 0.56 |
| q93 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.52 |
| q94 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q95 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q96 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.51 |
| q97 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q98 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q99 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q100 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q101 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q102 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q103 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q104 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q105 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q106 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q107 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.94 |
| q108 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.47 |
| q109 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q110 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q111 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.48 |
| q112 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q113 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.46 |
| q114 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.55 |
| q115 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q116 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q117 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q118 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q119 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q120 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.6 |
| q121 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q122 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q123 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q124 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.5 |
| q125 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q126 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q127 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q128 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q129 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.57 |
| q130 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q131 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q132 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.74 |
| q133 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.75 |
| q134 | single-hop | FALLBACK_STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.47 |
| q135 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q136 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.93 |
| q137 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.51 |
| q138 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q139 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q140 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.61 |
| q141 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q142 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.95 |
| q143 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.62 |
| q144 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.56 |
| q145 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.68 |
| q146 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q147 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q148 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q149 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q150 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.52 |
| q151 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q152 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.57 |
| q153 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q154 | adversarial | INJECT | 0 | 0 |  | abstention | 0.56 |
| q155 | adversarial | INJECT | 0 | 0 |  | abstention | 0.64 |
| q156 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q157 | adversarial | INJECT | 0 | 0 |  | abstention | 0.82 |
| q158 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.46 |
| q159 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.46 |
| q160 | adversarial | INJECT | 0 | 0 |  | abstention | 0.56 |
| q161 | adversarial | INJECT | 0 | 0 |  | abstention | 0.73 |
| q162 | adversarial | INJECT | 0 | 0 |  | abstention | 0.56 |
| q163 | adversarial | INJECT | 0 | 0 |  | abstention | 0.71 |
| q164 | adversarial | INJECT | 0 | 0 |  | abstention | 0.64 |
| q165 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.47 |
| q166 | adversarial | INJECT | 0 | 0 |  | abstention | 0.52 |
| q167 | adversarial | INJECT | 0 | 0 |  | abstention | 0.81 |
| q168 | adversarial | INJECT | 0 | 0 |  | abstention | 0.68 |
| q169 | adversarial | INJECT | 0 | 0 |  | abstention | 0.83 |
| q170 | adversarial | INJECT | 0 | 0 |  | abstention | 0.78 |
| q171 | adversarial | INJECT | 0 | 0 |  | abstention | 0.75 |
| q172 | adversarial | INJECT | 0 | 0 |  | abstention | 0.79 |
| q173 | adversarial | INJECT | 0 | 0 |  | abstention | 0.94 |
| q174 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.4 |
| q175 | adversarial | INJECT | 0 | 0 |  | abstention | 0.77 |
| q176 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.47 |
| q177 | adversarial | INJECT | 0 | 0 |  | abstention | 0.55 |
| q178 | adversarial | INJECT | 0 | 0 |  | abstention | 0.78 |
| q179 | adversarial | INJECT | 0 | 0 |  | abstention | 0.85 |
| q180 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q181 | adversarial | INJECT | 0 | 0 |  | abstention | 0.78 |
| q182 | adversarial | INJECT | 0 | 0 |  | abstention | 0.59 |
| q183 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q184 | adversarial | INJECT | 0 | 0 |  | abstention | 0.57 |
| q185 | adversarial | INJECT | 0 | 0 |  | abstention | 0.86 |
| q186 | adversarial | INJECT | 0 | 0 |  | abstention | 0.62 |
| q187 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q188 | adversarial | INJECT | 0 | 0 |  | abstention | 0.78 |
| q189 | adversarial | INJECT | 0 | 0 |  | abstention | 0.61 |
| q190 | adversarial | INJECT | 0 | 0 |  | abstention | 0.56 |
| q191 | adversarial | INJECT | 0 | 0 |  | abstention | 0.64 |
| q192 | adversarial | INJECT | 0 | 0 |  | abstention | 0.55 |
| q193 | adversarial | INJECT | 0 | 0 |  | abstention | 0.67 |
