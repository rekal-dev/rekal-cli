# smoke conv-49

- questions: 196
- route: skill
- calibration: locomo-stock-gates-only.json
- evidence@5 (answerable): 0.82
- evidence@10 (answerable): 0.85
- retrieved_context_tokens mean: 185.5
- answer_path_tokens mean: 217.4

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | multi-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.52 |
| q2 | multi-hop | INJECT | 0 | 1 | 7 | deep_rank_lt10 | 0.62 |
| q3 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q4 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.53 |
| q5 | temporal | INJECT | 0 | 0 |  | true_miss | 0.65 |
| q6 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.58 |
| q7 | multi-hop | INJECT | 0 | 0 |  | true_miss | 0.65 |
| q8 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.54 |
| q9 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q10 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q11 | open-domain | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.55 |
| q12 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q13 | temporal | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q14 | temporal | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q15 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.5 |
| q16 | multi-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.62 |
| q17 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.6 |
| q18 | temporal | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q19 | multi-hop | INJECT | 0 | 1 | 5 | deep_rank_lt10 | 0.76 |
| q20 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.56 |
| q21 | open-domain | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.6 |
| q22 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q23 | temporal | INJECT | 0 | 0 |  | true_miss | 0.58 |
| q24 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q25 | temporal | INJECT | 0 | 0 |  | true_miss | 0.72 |
| q26 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q27 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q28 | open-domain | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q29 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q30 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.59 |
| q31 | temporal | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.57 |
| q32 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.8 |
| q33 | multi-hop | INJECT | 0 | 0 |  | true_miss | 0.5 |
| q34 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q35 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q36 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q37 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q38 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q39 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.59 |
| q40 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.58 |
| q41 | temporal | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q42 | temporal | INJECT | 0 | 0 |  | true_miss | 0.58 |
| q43 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.55 |
| q44 | open-domain | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.57 |
| q45 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q46 | temporal | INJECT | 0 | 1 | 5 | deep_rank_lt10 | 0.72 |
| q47 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.63 |
| q48 | temporal | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q49 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.55 |
| q50 | multi-hop | INJECT | 0 | 1 | 7 | deep_rank_lt10 | 0.63 |
| q51 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.57 |
| q52 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.7 |
| q53 | temporal | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.51 |
| q54 | temporal | INJECT | 0 | 0 |  | true_miss | 0.66 |
| q55 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.54 |
| q56 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.56 |
| q57 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q58 | open-domain | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.51 |
| q59 | temporal | INJECT | 1 | 1 | 0 | hit | 0.59 |
| q60 | temporal | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q61 | temporal | INJECT | 0 | 0 |  | true_miss | 0.72 |
| q62 | temporal | INJECT | 1 | 1 | 0 | hit | 0.56 |
| q63 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.6 |
| q64 | temporal | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.58 |
| q65 | multi-hop | INJECT | 0 | 1 | 6 | deep_rank_lt10 | 0.57 |
| q66 | open-domain | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q67 | temporal | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.58 |
| q68 | temporal | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.62 |
| q69 | temporal | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.48 |
| q70 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q71 | temporal | INJECT | 1 | 1 | 0 | hit | 0.56 |
| q72 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.58 |
| q73 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.65 |
| q74 | temporal | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q75 | temporal | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q76 | temporal | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q77 | temporal | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q78 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.74 |
| q79 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q80 | temporal | INJECT | 1 | 1 | 0 | hit | 0.55 |
| q81 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q82 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.59 |
| q83 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.57 |
| q84 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q85 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q86 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.86 |
| q87 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q88 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.59 |
| q89 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q90 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.58 |
| q91 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q92 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q93 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q94 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.5 |
| q95 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q96 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.5 |
| q97 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.58 |
| q98 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.61 |
| q99 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.62 |
| q100 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q101 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q102 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q103 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q104 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q105 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.65 |
| q106 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q107 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q108 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q109 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q110 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.56 |
| q111 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.58 |
| q112 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q113 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.62 |
| q114 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q115 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q116 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q117 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.49 |
| q118 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q119 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.6 |
| q120 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.71 |
| q121 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q122 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q123 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.63 |
| q124 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.59 |
| q125 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q126 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.71 |
| q127 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q128 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.59 |
| q129 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q130 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q131 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q132 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q133 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q134 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q135 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q136 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q137 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.56 |
| q138 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q139 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q140 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.56 |
| q141 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q142 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q143 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q144 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.51 |
| q145 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.57 |
| q146 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q147 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.82 |
| q148 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.57 |
| q149 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.55 |
| q150 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q151 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q152 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.61 |
| q153 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q154 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.64 |
| q155 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.61 |
| q156 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q157 | adversarial | INJECT | 0 | 0 |  | abstention | 0.68 |
| q158 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q159 | adversarial | INJECT | 0 | 0 |  | abstention | 0.86 |
| q160 | adversarial | INJECT | 0 | 0 |  | abstention | 0.59 |
| q161 | adversarial | INJECT | 0 | 0 |  | abstention | 0.85 |
| q162 | adversarial | INJECT | 0 | 0 |  | abstention | 0.58 |
| q163 | adversarial | INJECT | 0 | 0 |  | abstention | 0.77 |
| q164 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.49 |
| q165 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.51 |
| q166 | adversarial | INJECT | 0 | 0 |  | abstention | 0.61 |
| q167 | adversarial | INJECT | 0 | 0 |  | abstention | 0.75 |
| q168 | adversarial | INJECT | 0 | 0 |  | abstention | 0.77 |
| q169 | adversarial | INJECT | 0 | 0 |  | abstention | 0.65 |
| q170 | adversarial | INJECT | 0 | 0 |  | abstention | 0.66 |
| q171 | adversarial | INJECT | 0 | 0 |  | abstention | 0.71 |
| q172 | adversarial | INJECT | 0 | 0 |  | abstention | 0.75 |
| q173 | adversarial | INJECT | 0 | 0 |  | abstention | 0.67 |
| q174 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q175 | adversarial | INJECT | 0 | 0 |  | abstention | 0.79 |
| q176 | adversarial | INJECT | 0 | 0 |  | abstention | 0.83 |
| q177 | adversarial | INJECT | 0 | 0 |  | abstention | 0.58 |
| q178 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q179 | adversarial | INJECT | 0 | 0 |  | abstention | 0.56 |
| q180 | adversarial | INJECT | 0 | 0 |  | abstention | 0.63 |
| q181 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q182 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q183 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q184 | adversarial | INJECT | 0 | 0 |  | abstention | 0.6 |
| q185 | adversarial | INJECT | 0 | 0 |  | abstention | 0.85 |
| q186 | adversarial | INJECT | 0 | 0 |  | abstention | 0.63 |
| q187 | adversarial | INJECT | 0 | 0 |  | abstention | 0.87 |
| q188 | adversarial | INJECT | 0 | 0 |  | abstention | 0.76 |
| q189 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q190 | adversarial | INJECT | 0 | 0 |  | abstention | 0.73 |
| q191 | adversarial | INJECT | 0 | 0 |  | abstention | 0.57 |
| q192 | adversarial | INJECT | 0 | 0 |  | abstention | 0.55 |
| q193 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q194 | adversarial | INJECT | 0 | 0 |  | abstention | 0.61 |
| q195 | adversarial | INJECT | 0 | 0 |  | abstention | 0.67 |
| q196 | adversarial | INJECT | 0 | 0 |  | abstention | 0.65 |
