# smoke conv-47

- questions: 190
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 0.93
- evidence@10 (answerable): 0.95
- retrieved_context_tokens mean: 146.9
- answer_path_tokens mean: 173.6

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | open-domain | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.5 |
| q2 | temporal | INJECT | 0 | 0 |  | true_miss | 0.58 |
| q3 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.63 |
| q4 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.48 |
| q5 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q6 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.55 |
| q7 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.65 |
| q8 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q9 | multi-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.48 |
| q10 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.54 |
| q11 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q12 | temporal | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.42 |
| q13 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.54 |
| q14 | temporal | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q15 | temporal | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.55 |
| q16 | temporal | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q17 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q18 | open-domain | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.63 |
| q19 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q20 | open-domain | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.53 |
| q21 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q22 | temporal | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q23 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q24 | temporal | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q25 | multi-hop | FALLBACK_STOCK | 0 | 1 | 6 | deep_rank_lt10 | 0.52 |
| q26 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q27 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q28 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.45 |
| q29 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q30 | temporal | INJECT | 0 | 0 |  | true_miss | 0.62 |
| q31 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q32 | temporal | INJECT | 0 | 0 |  | true_miss | 0.64 |
| q33 | temporal | INJECT | 1 | 1 | 0 | hit | 0.55 |
| q34 | open-domain | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.74 |
| q35 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.76 |
| q36 | open-domain | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.76 |
| q37 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.58 |
| q38 | temporal | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q39 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.56 |
| q40 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q41 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.54 |
| q42 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.53 |
| q43 | temporal | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q44 | temporal | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q45 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.55 |
| q46 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q47 | temporal | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.72 |
| q48 | temporal | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q49 | temporal | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q50 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q51 | temporal | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q52 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.64 |
| q53 | temporal | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q54 | temporal | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q55 | multi-hop | INJECT | 0 | 0 |  | true_miss | 0.6 |
| q56 | temporal | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.72 |
| q57 | multi-hop | INJECT | 0 | 1 | 5 | deep_rank_lt10 | 0.65 |
| q58 | temporal | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q59 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.76 |
| q60 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.78 |
| q61 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q62 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q63 | temporal | INJECT | 1 | 1 | 0 | hit | 0.88 |
| q64 | temporal | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q65 | temporal | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q66 | temporal | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q67 | temporal | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.56 |
| q68 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q69 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q70 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q71 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q72 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.58 |
| q73 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.61 |
| q74 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q75 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q76 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q77 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q78 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.89 |
| q79 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q80 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q81 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q82 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q83 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q84 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.83 |
| q85 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q86 | single-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.52 |
| q87 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q88 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q89 | single-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.46 |
| q90 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q91 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q92 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q93 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q94 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q95 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.72 |
| q96 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q97 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q98 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q99 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.43 |
| q100 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q101 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q102 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.61 |
| q103 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q104 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q105 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q106 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q107 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q108 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q109 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q110 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q111 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q112 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q113 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.58 |
| q114 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q115 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q116 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q117 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q118 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q119 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q120 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q121 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q122 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q123 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q124 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q125 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q126 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q127 | single-hop | FALLBACK_STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.53 |
| q128 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.56 |
| q129 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.51 |
| q130 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q131 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q132 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q133 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q134 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.61 |
| q135 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q136 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q137 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q138 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q139 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.68 |
| q140 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q141 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q142 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q143 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.59 |
| q144 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q145 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q146 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q147 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q148 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q149 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.56 |
| q150 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.55 |
| q151 | adversarial | INJECT | 0 | 0 |  | abstention | 0.73 |
| q152 | adversarial | INJECT | 0 | 0 |  | abstention | 0.68 |
| q153 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q154 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q155 | adversarial | INJECT | 0 | 0 |  | abstention | 0.77 |
| q156 | adversarial | INJECT | 0 | 0 |  | abstention | 0.79 |
| q157 | adversarial | INJECT | 0 | 0 |  | abstention | 0.9 |
| q158 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q159 | adversarial | INJECT | 0 | 0 |  | abstention | 0.78 |
| q160 | adversarial | INJECT | 0 | 0 |  | abstention | 0.6 |
| q161 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.47 |
| q162 | adversarial | INJECT | 0 | 0 |  | abstention | 0.71 |
| q163 | adversarial | INJECT | 0 | 0 |  | abstention | 0.64 |
| q164 | adversarial | INJECT | 0 | 0 |  | abstention | 0.65 |
| q165 | adversarial | INJECT | 0 | 0 |  | abstention | 0.75 |
| q166 | adversarial | INJECT | 0 | 0 |  | abstention | 0.66 |
| q167 | adversarial | INJECT | 0 | 0 |  | abstention | 0.68 |
| q168 | adversarial | INJECT | 0 | 0 |  | abstention | 0.62 |
| q169 | adversarial | INJECT | 0 | 0 |  | abstention | 0.51 |
| q170 | adversarial | INJECT | 0 | 0 |  | abstention | 0.68 |
| q171 | adversarial | INJECT | 0 | 0 |  | abstention | 0.63 |
| q172 | adversarial | INJECT | 0 | 0 |  | abstention | 0.58 |
| q173 | adversarial | INJECT | 0 | 0 |  | abstention | 0.8 |
| q174 | adversarial | INJECT | 0 | 0 |  | abstention | 0.63 |
| q175 | adversarial | INJECT | 0 | 0 |  | abstention | 0.83 |
| q176 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.52 |
| q177 | adversarial | INJECT | 0 | 0 |  | abstention | 0.56 |
| q178 | adversarial | INJECT | 0 | 0 |  | abstention | 0.71 |
| q179 | adversarial | INJECT | 0 | 0 |  | abstention | 0.89 |
| q180 | adversarial | INJECT | 0 | 0 |  | abstention | 0.79 |
| q181 | adversarial | INJECT | 0 | 0 |  | abstention | 0.84 |
| q182 | adversarial | INJECT | 0 | 0 |  | abstention | 0.62 |
| q183 | adversarial | INJECT | 0 | 0 |  | abstention | 0.66 |
| q184 | adversarial | INJECT | 0 | 0 |  | abstention | 0.59 |
| q185 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q186 | adversarial | INJECT | 0 | 0 |  | abstention | 0.75 |
| q187 | adversarial | INJECT | 0 | 0 |  | abstention | 0.84 |
| q188 | adversarial | INJECT | 0 | 0 |  | abstention | 0.66 |
| q189 | adversarial | INJECT | 0 | 0 |  | abstention | 0.57 |
| q190 | adversarial | INJECT | 0 | 0 |  | abstention | 0.55 |
