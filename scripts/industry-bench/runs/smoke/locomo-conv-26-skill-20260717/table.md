# smoke conv-26

- questions: 199
- route: skill
- calibration: chat-provisional.json
- evidence@5 (answerable): 0.90
- evidence@10 (answerable): 0.95
- retrieved_context_tokens mean: 172.7
- answer_path_tokens mean: 203.9

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | temporal | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q2 | temporal | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q3 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q4 | multi-hop | INJECT | 0 | 1 | 6 | deep_rank_lt10 | 0.78 |
| q5 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q6 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q7 | temporal | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q8 | multi-hop | INJECT | 0 | 1 | 6 | deep_rank_lt10 | 0.71 |
| q9 | temporal | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q10 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.74 |
| q11 | temporal | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.71 |
| q12 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.61 |
| q13 | temporal | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q14 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q15 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q16 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q17 | temporal | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q18 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q19 | multi-hop | INJECT | 0 | 1 | 5 | deep_rank_lt10 | 0.57 |
| q20 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q21 | temporal | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.72 |
| q22 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q23 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q24 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q25 | multi-hop | INJECT | 0 | 1 | 6 | deep_rank_lt10 | 0.72 |
| q26 | temporal | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q27 | temporal | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.69 |
| q28 | open-domain | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.73 |
| q29 | temporal | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.74 |
| q30 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q31 | open-domain | INJECT | 0 | 0 |  | abstention | 0.68 |
| q32 | temporal | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.62 |
| q33 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.76 |
| q34 | temporal | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q35 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.75 |
| q36 | temporal | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.61 |
| q37 | temporal | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q38 | multi-hop | INJECT | 0 | 1 | 5 | deep_rank_lt10 | 0.67 |
| q39 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.71 |
| q40 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q41 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q42 | temporal | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q43 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.74 |
| q44 | multi-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.7 |
| q45 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q46 | temporal | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q47 | open-domain | INJECT | 0 | 0 |  | abstention | 0.69 |
| q48 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.7 |
| q49 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q50 | temporal | INJECT | 0 | 0 |  | true_miss | 0.73 |
| q51 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.71 |
| q52 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q53 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q54 | temporal | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.76 |
| q55 | temporal | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.69 |
| q56 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.72 |
| q57 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.68 |
| q58 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q59 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.81 |
| q60 | open-domain | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q61 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q62 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q63 | temporal | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.68 |
| q64 | temporal | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q65 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q66 | multi-hop | INJECT | 0 | 1 | 5 | deep_rank_lt10 | 0.73 |
| q67 | multi-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.69 |
| q68 | temporal | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q69 | temporal | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q70 | open-domain | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.72 |
| q71 | multi-hop | INJECT | 0 | 1 | 7 | deep_rank_lt10 | 0.74 |
| q72 | multi-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.73 |
| q73 | temporal | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.67 |
| q74 | temporal | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.72 |
| q75 | temporal | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q76 | multi-hop | INJECT | 0 | 1 | 7 | deep_rank_lt10 | 0.63 |
| q77 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q78 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q79 | multi-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.72 |
| q80 | temporal | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q81 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q82 | open-domain | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.73 |
| q83 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q84 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q85 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q86 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.68 |
| q87 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.79 |
| q88 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.76 |
| q89 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.79 |
| q90 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q91 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q92 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q93 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q94 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q95 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q96 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.69 |
| q97 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.91 |
| q98 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q99 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q100 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q101 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q102 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q103 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q104 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q105 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.76 |
| q106 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.63 |
| q107 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.66 |
| q108 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.65 |
| q109 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.7 |
| q110 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q111 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q112 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q113 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q114 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.74 |
| q115 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.7 |
| q116 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q117 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.72 |
| q118 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q119 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q120 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q121 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q122 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q123 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q124 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.71 |
| q125 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q126 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q127 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.71 |
| q128 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q129 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.65 |
| q130 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.71 |
| q131 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.58 |
| q132 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q133 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q134 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.59 |
| q135 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q136 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.55 |
| q137 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.66 |
| q138 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.73 |
| q139 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.74 |
| q140 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.67 |
| q141 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q142 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.71 |
| q143 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.76 |
| q144 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q145 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q146 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q147 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q148 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.74 |
| q149 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q150 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.68 |
| q151 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.69 |
| q152 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q153 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q154 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q155 | adversarial | INJECT | 0 | 0 |  | abstention | 0.75 |
| q156 | adversarial | INJECT | 0 | 0 |  | abstention | 0.73 |
| q157 | adversarial | INJECT | 0 | 0 |  | abstention | 0.77 |
| q158 | adversarial | INJECT | 0 | 0 |  | abstention | 0.73 |
| q159 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q160 | adversarial | INJECT | 0 | 0 |  | abstention | 0.71 |
| q161 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q162 | adversarial | INJECT | 0 | 0 |  | abstention | 0.71 |
| q163 | adversarial | INJECT | 0 | 0 |  | abstention | 0.71 |
| q164 | adversarial | INJECT | 0 | 0 |  | abstention | 0.92 |
| q165 | adversarial | INJECT | 0 | 0 |  | abstention | 0.76 |
| q166 | adversarial | INJECT | 0 | 0 |  | abstention | 0.77 |
| q167 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q168 | adversarial | INJECT | 0 | 0 |  | abstention | 0.85 |
| q169 | adversarial | INJECT | 0 | 0 |  | abstention | 0.68 |
| q170 | adversarial | INJECT | 0 | 0 |  | abstention | 0.66 |
| q171 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q172 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q173 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q174 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q175 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q176 | adversarial | INJECT | 0 | 0 |  | abstention | 0.73 |
| q177 | adversarial | INJECT | 0 | 0 |  | abstention | 0.65 |
| q178 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q179 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q180 | adversarial | INJECT | 0 | 0 |  | abstention | 0.64 |
| q181 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q182 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q183 | adversarial | INJECT | 0 | 0 |  | abstention | 0.61 |
| q184 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q185 | adversarial | INJECT | 0 | 0 |  | abstention | 0.73 |
| q186 | adversarial | INJECT | 0 | 0 |  | abstention | 0.61 |
| q187 | adversarial | INJECT | 0 | 0 |  | abstention | 0.77 |
| q188 | adversarial | INJECT | 0 | 0 |  | abstention | 0.59 |
| q189 | adversarial | INJECT | 0 | 0 |  | abstention | 0.61 |
| q190 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q191 | adversarial | INJECT | 0 | 0 |  | abstention | 0.63 |
| q192 | adversarial | INJECT | 0 | 0 |  | abstention | 0.73 |
| q193 | adversarial | INJECT | 0 | 0 |  | abstention | 0.73 |
| q194 | adversarial | INJECT | 0 | 0 |  | abstention | 0.76 |
| q195 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q196 | adversarial | INJECT | 0 | 0 |  | abstention | 0.76 |
| q197 | adversarial | INJECT | 0 | 0 |  | abstention | 0.83 |
| q198 | adversarial | INJECT | 0 | 0 |  | abstention | 0.79 |
| q199 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
