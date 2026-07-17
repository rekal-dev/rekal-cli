# smoke conv-44

- questions: 158
- route: skill
- calibration: chat-provisional.json
- evidence@5 (answerable): 0.89
- evidence@10 (answerable): 0.92
- retrieved_context_tokens mean: 183.9
- answer_path_tokens mean: 217.9

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | temporal | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.74 |
| q2 | temporal | INJECT | 1 | 1 | 0 | hit | 0.91 |
| q3 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.66 |
| q4 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q5 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q6 | temporal | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q7 | temporal | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q8 | temporal | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q9 | temporal | INJECT | 0 | 0 |  | true_miss | 0.71 |
| q10 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q11 | temporal | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q12 | temporal | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q13 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q14 | temporal | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q15 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q16 | multi-hop | INJECT | 0 | 1 | 6 | deep_rank_lt10 | 0.73 |
| q17 | temporal | INJECT | 0 | 0 |  | true_miss | 0.73 |
| q18 | multi-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.68 |
| q19 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q20 | open-domain | INJECT | 0 | 1 | 6 | deep_rank_lt10 | 0.75 |
| q21 | open-domain | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.68 |
| q22 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q23 | temporal | INJECT | 0 | 0 |  | true_miss | 0.58 |
| q24 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.63 |
| q25 | multi-hop | INJECT | 0 | 1 | 5 | deep_rank_lt10 | 0.78 |
| q26 | temporal | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q27 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q28 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q29 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.71 |
| q30 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.75 |
| q31 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.7 |
| q32 | multi-hop | INJECT | 0 | 0 |  | true_miss | 0.75 |
| q33 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q34 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q35 | temporal | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q36 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.74 |
| q37 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q38 | temporal | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.62 |
| q39 | temporal | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q40 | multi-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.74 |
| q41 | multi-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.71 |
| q42 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q43 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q44 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.69 |
| q45 | open-domain | INJECT | 0 | 1 | 6 | deep_rank_lt10 | 0.76 |
| q46 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.72 |
| q47 | temporal | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.73 |
| q48 | temporal | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q49 | multi-hop | INJECT | 0 | 0 |  | true_miss | 0.77 |
| q50 | multi-hop | INJECT | 0 | 0 |  | true_miss | 0.75 |
| q51 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.78 |
| q52 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q53 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q54 | open-domain | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.73 |
| q55 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q56 | temporal | INJECT | 0 | 0 |  | true_miss | 0.76 |
| q57 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q58 | temporal | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q59 | temporal | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q60 | temporal | INJECT | 0 | 0 |  | true_miss | 0.71 |
| q61 | multi-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.74 |
| q62 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q63 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q64 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q65 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.73 |
| q66 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.76 |
| q67 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q68 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q69 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q70 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q71 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q72 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.77 |
| q73 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q74 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.73 |
| q75 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.76 |
| q76 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q77 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.73 |
| q78 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q79 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q80 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q81 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.86 |
| q82 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q83 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.72 |
| q84 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q85 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.75 |
| q86 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.74 |
| q87 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q88 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q89 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q90 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.76 |
| q91 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.74 |
| q92 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.86 |
| q93 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.77 |
| q94 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q95 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.75 |
| q96 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q97 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.93 |
| q98 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.89 |
| q99 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q100 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.71 |
| q101 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q102 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q103 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q104 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q105 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.77 |
| q106 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q107 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.73 |
| q108 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.7 |
| q109 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.74 |
| q110 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.8 |
| q111 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q112 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q113 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q114 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q115 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q116 | single-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.62 |
| q117 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q118 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.76 |
| q119 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q120 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q121 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q122 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q123 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q124 | adversarial | INJECT | 0 | 0 |  | abstention | 0.68 |
| q125 | adversarial | INJECT | 0 | 0 |  | abstention | 0.75 |
| q126 | adversarial | INJECT | 0 | 0 |  | abstention | 0.78 |
| q127 | adversarial | INJECT | 0 | 0 |  | abstention | 0.66 |
| q128 | adversarial | INJECT | 0 | 0 |  | abstention | 0.8 |
| q129 | adversarial | INJECT | 0 | 0 |  | abstention | 0.67 |
| q130 | adversarial | INJECT | 0 | 0 |  | abstention | 0.76 |
| q131 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q132 | adversarial | INJECT | 0 | 0 |  | abstention | 0.81 |
| q133 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q134 | adversarial | INJECT | 0 | 0 |  | abstention | 0.66 |
| q135 | adversarial | INJECT | 0 | 0 |  | abstention | 0.71 |
| q136 | adversarial | INJECT | 0 | 0 |  | abstention | 0.78 |
| q137 | adversarial | INJECT | 0 | 0 |  | abstention | 0.76 |
| q138 | adversarial | INJECT | 0 | 0 |  | abstention | 0.73 |
| q139 | adversarial | INJECT | 0 | 0 |  | abstention | 0.75 |
| q140 | adversarial | INJECT | 0 | 0 |  | abstention | 0.67 |
| q141 | adversarial | INJECT | 0 | 0 |  | abstention | 0.85 |
| q142 | adversarial | INJECT | 0 | 0 |  | abstention | 0.73 |
| q143 | adversarial | INJECT | 0 | 0 |  | abstention | 0.75 |
| q144 | adversarial | INJECT | 0 | 0 |  | abstention | 0.75 |
| q145 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q146 | adversarial | INJECT | 0 | 0 |  | abstention | 0.86 |
| q147 | adversarial | INJECT | 0 | 0 |  | abstention | 0.71 |
| q148 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q149 | adversarial | INJECT | 0 | 0 |  | abstention | 0.78 |
| q150 | adversarial | INJECT | 0 | 0 |  | abstention | 0.78 |
| q151 | adversarial | INJECT | 0 | 0 |  | abstention | 0.66 |
| q152 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q153 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q154 | adversarial | INJECT | 0 | 0 |  | abstention | 0.75 |
| q155 | adversarial | INJECT | 0 | 0 |  | abstention | 0.73 |
| q156 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q157 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q158 | adversarial | INJECT | 0 | 0 |  | abstention | 0.79 |
