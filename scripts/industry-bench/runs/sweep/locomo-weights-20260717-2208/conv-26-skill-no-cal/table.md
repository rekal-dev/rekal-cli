# smoke conv-26

- questions: 199
- route: skill
- calibration: stock
- evidence@5 (answerable): 0.86
- evidence@10 (answerable): 0.89
- retrieved_context_tokens mean: 178.4
- answer_path_tokens mean: 208.9

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | temporal | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q2 | temporal | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q3 | open-domain | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.5 |
| q4 | multi-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.78 |
| q5 | multi-hop | FALLBACK_STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.53 |
| q6 | temporal | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q7 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.57 |
| q8 | multi-hop | FALLBACK_STOCK | 0 | 1 | 9 | deep_rank_lt10 | 0.71 |
| q9 | temporal | FALLBACK_STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.47 |
| q10 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.62 |
| q11 | temporal | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.53 |
| q12 | multi-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.58 |
| q13 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.7 |
| q14 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.51 |
| q15 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q16 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.77 |
| q17 | temporal | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q18 | temporal | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q19 | multi-hop | FALLBACK_STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.51 |
| q20 | multi-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.61 |
| q21 | temporal | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q22 | temporal | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.71 |
| q23 | open-domain | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.68 |
| q24 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.69 |
| q25 | multi-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q26 | temporal | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q27 | temporal | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.69 |
| q28 | open-domain | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q29 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.55 |
| q30 | temporal | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.6 |
| q31 | open-domain | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.54 |
| q32 | temporal | FALLBACK_STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.44 |
| q33 | multi-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.57 |
| q34 | temporal | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.61 |
| q35 | multi-hop | FALLBACK_STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.55 |
| q36 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.44 |
| q37 | temporal | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q38 | multi-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.54 |
| q39 | multi-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.56 |
| q40 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.59 |
| q41 | multi-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.62 |
| q42 | temporal | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q43 | open-domain | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.74 |
| q44 | multi-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.53 |
| q45 | temporal | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q46 | temporal | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.62 |
| q47 | open-domain | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.52 |
| q48 | multi-hop | FALLBACK_STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.5 |
| q49 | multi-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.61 |
| q50 | temporal | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.49 |
| q51 | open-domain | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.71 |
| q52 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.68 |
| q53 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q54 | temporal | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.68 |
| q55 | temporal | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.46 |
| q56 | multi-hop | FALLBACK_STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.49 |
| q57 | multi-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.55 |
| q58 | temporal | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.65 |
| q59 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.81 |
| q60 | open-domain | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.71 |
| q61 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q62 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.71 |
| q63 | temporal | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.68 |
| q64 | temporal | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q65 | open-domain | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.67 |
| q66 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.58 |
| q67 | multi-hop | FALLBACK_STOCK | 0 | 1 | 5 | deep_rank_lt10 | 0.64 |
| q68 | temporal | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.55 |
| q69 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.52 |
| q70 | open-domain | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q71 | multi-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.55 |
| q72 | multi-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q73 | temporal | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.53 |
| q74 | temporal | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q75 | temporal | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.53 |
| q76 | multi-hop | FALLBACK_STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.56 |
| q77 | multi-hop | FALLBACK_STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.64 |
| q78 | open-domain | FALLBACK_STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.53 |
| q79 | multi-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.72 |
| q80 | temporal | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q81 | temporal | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q82 | open-domain | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q83 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q84 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q85 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q86 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.64 |
| q87 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.6 |
| q88 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q89 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q90 | single-hop | FALLBACK_STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.59 |
| q91 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.68 |
| q92 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q93 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q94 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q95 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q96 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.47 |
| q97 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.91 |
| q98 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.71 |
| q99 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.69 |
| q100 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q101 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q102 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q103 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q104 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q105 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q106 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.61 |
| q107 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q108 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.53 |
| q109 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.68 |
| q110 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.64 |
| q111 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q112 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q113 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.59 |
| q114 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q115 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.55 |
| q116 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.53 |
| q117 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.61 |
| q118 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q119 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.68 |
| q120 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.66 |
| q121 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.57 |
| q122 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q123 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q124 | single-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.71 |
| q125 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.65 |
| q126 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q127 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q128 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.71 |
| q129 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.55 |
| q130 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q131 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.58 |
| q132 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q133 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.66 |
| q134 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.59 |
| q135 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q136 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.55 |
| q137 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.63 |
| q138 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.62 |
| q139 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.52 |
| q140 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q141 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q142 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.52 |
| q143 | single-hop | FALLBACK_STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.55 |
| q144 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q145 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q146 | single-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.7 |
| q147 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q148 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.74 |
| q149 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q150 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.57 |
| q151 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.66 |
| q152 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q153 | adversarial | INJECT | 0 | 0 |  | abstention | 0.71 |
| q154 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.65 |
| q155 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.59 |
| q156 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.71 |
| q157 | adversarial | INJECT | 0 | 0 |  | abstention | 0.76 |
| q158 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.72 |
| q159 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q160 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.65 |
| q161 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.65 |
| q162 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q163 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.46 |
| q164 | adversarial | INJECT | 0 | 0 |  | abstention | 0.92 |
| q165 | adversarial | INJECT | 0 | 0 |  | abstention | 0.76 |
| q166 | adversarial | INJECT | 0 | 0 |  | abstention | 0.77 |
| q167 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q168 | adversarial | INJECT | 0 | 0 |  | abstention | 0.85 |
| q169 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.68 |
| q170 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.53 |
| q171 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.68 |
| q172 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q173 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.6 |
| q174 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.6 |
| q175 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.66 |
| q176 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.68 |
| q177 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.65 |
| q178 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q179 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.72 |
| q180 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.64 |
| q181 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.72 |
| q182 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.72 |
| q183 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.55 |
| q184 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q185 | adversarial | INJECT | 0 | 0 |  | abstention | 0.73 |
| q186 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.58 |
| q187 | adversarial | INJECT | 0 | 0 |  | abstention | 0.77 |
| q188 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.59 |
| q189 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.6 |
| q190 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.62 |
| q191 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.63 |
| q192 | adversarial | INJECT | 0 | 0 |  | abstention | 0.71 |
| q193 | adversarial | INJECT | 0 | 0 |  | abstention | 0.71 |
| q194 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.7 |
| q195 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q196 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.76 |
| q197 | adversarial | INJECT | 0 | 0 |  | abstention | 0.83 |
| q198 | adversarial | INJECT | 0 | 0 |  | abstention | 0.79 |
| q199 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.74 |
