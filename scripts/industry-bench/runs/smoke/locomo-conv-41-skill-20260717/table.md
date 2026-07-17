# smoke conv-41

- questions: 193
- route: skill
- calibration: chat-provisional.json
- evidence@5 (answerable): 0.91
- evidence@10 (answerable): 0.93
- retrieved_context_tokens mean: 177.1
- answer_path_tokens mean: 208.7

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | temporal | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.76 |
| q2 | temporal | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q3 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.78 |
| q4 | multi-hop | INJECT | 0 | 0 |  | true_miss | 0.77 |
| q5 | temporal | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q6 | temporal | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.63 |
| q7 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q8 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.63 |
| q9 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.65 |
| q10 | multi-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.69 |
| q11 | temporal | INJECT | 0 | 0 |  | true_miss | 0.6 |
| q12 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q13 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.64 |
| q14 | temporal | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q15 | open-domain | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.64 |
| q16 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q17 | temporal | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q18 | open-domain | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q19 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q20 | multi-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.73 |
| q21 | temporal | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q22 | multi-hop | INJECT | 0 | 1 | 5 | deep_rank_lt10 | 0.63 |
| q23 | temporal | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q24 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.57 |
| q25 | temporal | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q26 | multi-hop | INJECT | 0 | 1 | 5 | deep_rank_lt10 | 0.59 |
| q27 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q28 | temporal | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q29 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.69 |
| q30 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q31 | multi-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.72 |
| q32 | temporal | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q33 | multi-hop | INJECT | 0 | 0 |  | true_miss | 0.71 |
| q34 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.69 |
| q35 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q36 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q37 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q38 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q39 | temporal | INJECT | 1 | 1 | 0 | hit | 0.59 |
| q40 | open-domain | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.61 |
| q41 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.6 |
| q42 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.64 |
| q43 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q44 | temporal | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q45 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q46 | open-domain | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.7 |
| q47 | temporal | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q48 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q49 | temporal | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q50 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q51 | open-domain | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.57 |
| q52 | temporal | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.75 |
| q53 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q54 | temporal | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q55 | temporal | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q56 | temporal | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q57 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q58 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q59 | temporal | INJECT | 1 | 1 | 0 | hit | 0.59 |
| q60 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.68 |
| q61 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q62 | temporal | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q63 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q64 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.53 |
| q65 | open-domain | INJECT | 0 | 1 | 6 | deep_rank_lt10 | 0.67 |
| q66 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q67 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.9 |
| q68 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.88 |
| q69 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q70 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q71 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q72 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q73 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q74 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.76 |
| q75 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.66 |
| q76 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.63 |
| q77 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q78 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q79 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.58 |
| q80 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.61 |
| q81 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q82 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q83 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q84 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q85 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.65 |
| q86 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.68 |
| q87 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q88 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q89 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q90 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q91 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q92 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q93 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q94 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q95 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q96 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q97 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q98 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q99 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q100 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q101 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q102 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q103 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q104 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q105 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q106 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q107 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.94 |
| q108 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.6 |
| q109 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q110 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q111 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.61 |
| q112 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q113 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.58 |
| q114 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.61 |
| q115 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q116 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q117 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q118 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q119 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q120 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.69 |
| q121 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.58 |
| q122 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q123 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q124 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.57 |
| q125 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q126 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q127 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q128 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q129 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.64 |
| q130 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q131 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q132 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.74 |
| q133 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.75 |
| q134 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.58 |
| q135 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q136 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.93 |
| q137 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q138 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q139 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.65 |
| q140 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.72 |
| q141 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q142 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.95 |
| q143 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.67 |
| q144 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q145 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.68 |
| q146 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q147 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q148 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.68 |
| q149 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q150 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.67 |
| q151 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q152 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.62 |
| q153 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q154 | adversarial | INJECT | 0 | 0 |  | abstention | 0.57 |
| q155 | adversarial | INJECT | 0 | 0 |  | abstention | 0.76 |
| q156 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q157 | adversarial | INJECT | 0 | 0 |  | abstention | 0.82 |
| q158 | adversarial | INJECT | 0 | 0 |  | abstention | 0.62 |
| q159 | adversarial | INJECT | 0 | 0 |  | abstention | 0.59 |
| q160 | adversarial | INJECT | 0 | 0 |  | abstention | 0.65 |
| q161 | adversarial | INJECT | 0 | 0 |  | abstention | 0.73 |
| q162 | adversarial | INJECT | 0 | 0 |  | abstention | 0.64 |
| q163 | adversarial | INJECT | 0 | 0 |  | abstention | 0.71 |
| q164 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q165 | adversarial | INJECT | 0 | 0 |  | abstention | 0.67 |
| q166 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q167 | adversarial | INJECT | 0 | 0 |  | abstention | 0.81 |
| q168 | adversarial | INJECT | 0 | 0 |  | abstention | 0.68 |
| q169 | adversarial | INJECT | 0 | 0 |  | abstention | 0.83 |
| q170 | adversarial | INJECT | 0 | 0 |  | abstention | 0.78 |
| q171 | adversarial | INJECT | 0 | 0 |  | abstention | 0.75 |
| q172 | adversarial | INJECT | 0 | 0 |  | abstention | 0.79 |
| q173 | adversarial | INJECT | 0 | 0 |  | abstention | 0.94 |
| q174 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q175 | adversarial | INJECT | 0 | 0 |  | abstention | 0.77 |
| q176 | adversarial | INJECT | 0 | 0 |  | abstention | 0.6 |
| q177 | adversarial | INJECT | 0 | 0 |  | abstention | 0.63 |
| q178 | adversarial | INJECT | 0 | 0 |  | abstention | 0.78 |
| q179 | adversarial | INJECT | 0 | 0 |  | abstention | 0.85 |
| q180 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q181 | adversarial | INJECT | 0 | 0 |  | abstention | 0.78 |
| q182 | adversarial | INJECT | 0 | 0 |  | abstention | 0.59 |
| q183 | adversarial | INJECT | 0 | 0 |  | abstention | 0.75 |
| q184 | adversarial | INJECT | 0 | 0 |  | abstention | 0.71 |
| q185 | adversarial | INJECT | 0 | 0 |  | abstention | 0.86 |
| q186 | adversarial | INJECT | 0 | 0 |  | abstention | 0.62 |
| q187 | adversarial | INJECT | 0 | 0 |  | abstention | 0.71 |
| q188 | adversarial | INJECT | 0 | 0 |  | abstention | 0.78 |
| q189 | adversarial | INJECT | 0 | 0 |  | abstention | 0.66 |
| q190 | adversarial | INJECT | 0 | 0 |  | abstention | 0.61 |
| q191 | adversarial | INJECT | 0 | 0 |  | abstention | 0.76 |
| q192 | adversarial | INJECT | 0 | 0 |  | abstention | 0.62 |
| q193 | adversarial | INJECT | 0 | 0 |  | abstention | 0.67 |
