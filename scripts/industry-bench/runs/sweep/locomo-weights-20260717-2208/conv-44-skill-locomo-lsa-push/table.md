# smoke conv-44

- questions: 158
- route: skill
- calibration: locomo-lsa-push.json
- evidence@5 (answerable): 0.83
- evidence@10 (answerable): 0.86
- retrieved_context_tokens mean: 192.4
- answer_path_tokens mean: 224.2

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | temporal | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.48 |
| q2 | temporal | INJECT | 1 | 1 | 0 | hit | 0.91 |
| q3 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.53 |
| q4 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.65 |
| q5 | temporal | INJECT | 0 | 0 |  | true_miss | 0.61 |
| q6 | temporal | INJECT | 0 | 0 |  | true_miss | 0.61 |
| q7 | temporal | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q8 | temporal | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q9 | temporal | INJECT | 0 | 0 |  | true_miss | 0.61 |
| q10 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q11 | temporal | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q12 | temporal | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q13 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q14 | temporal | INJECT | 1 | 1 | 0 | hit | 0.55 |
| q15 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.73 |
| q16 | multi-hop | INJECT | 0 | 1 | 5 | deep_rank_lt10 | 0.54 |
| q17 | temporal | INJECT | 0 | 0 |  | true_miss | 0.5 |
| q18 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.59 |
| q19 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q20 | open-domain | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.6 |
| q21 | open-domain | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.49 |
| q22 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q23 | temporal | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.41 |
| q24 | multi-hop | FALLBACK_STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.52 |
| q25 | multi-hop | FALLBACK_STOCK | 0 | 1 | 6 | deep_rank_lt10 | 0.49 |
| q26 | temporal | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q27 | multi-hop | FALLBACK_STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.48 |
| q28 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.55 |
| q29 | multi-hop | FALLBACK_STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.42 |
| q30 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.59 |
| q31 | temporal | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.67 |
| q32 | multi-hop | INJECT | 0 | 0 |  | true_miss | 0.73 |
| q33 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q34 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.61 |
| q35 | temporal | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q36 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q37 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.6 |
| q38 | temporal | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.48 |
| q39 | temporal | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q40 | multi-hop | INJECT | 0 | 1 | 6 | deep_rank_lt10 | 0.54 |
| q41 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.69 |
| q42 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.55 |
| q43 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.61 |
| q44 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.5 |
| q45 | open-domain | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q46 | temporal | FALLBACK_STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.41 |
| q47 | temporal | FALLBACK_STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.41 |
| q48 | temporal | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q49 | multi-hop | INJECT | 0 | 1 | 9 | deep_rank_lt10 | 0.56 |
| q50 | multi-hop | INJECT | 0 | 0 |  | true_miss | 0.56 |
| q51 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q52 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.55 |
| q53 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q54 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.6 |
| q55 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.54 |
| q56 | temporal | INJECT | 0 | 0 |  | true_miss | 0.64 |
| q57 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.67 |
| q58 | temporal | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.56 |
| q59 | temporal | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q60 | temporal | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.5 |
| q61 | multi-hop | FALLBACK_STOCK | 1 | 1 | 1 | deep_rank_lt5 | 0.47 |
| q62 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q63 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q64 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q65 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.73 |
| q66 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.56 |
| q67 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.59 |
| q68 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q69 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q70 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q71 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.57 |
| q72 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q73 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q74 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.55 |
| q75 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.68 |
| q76 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q77 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q78 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q79 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q80 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q81 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.86 |
| q82 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q83 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q84 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q85 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.68 |
| q86 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q87 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.5 |
| q88 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q89 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q90 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q91 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q92 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.86 |
| q93 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.6 |
| q94 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.53 |
| q95 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.5 |
| q96 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q97 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.93 |
| q98 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.61 |
| q99 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q100 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.51 |
| q101 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q102 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.62 |
| q103 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q104 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q105 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q106 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q107 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.65 |
| q108 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.5 |
| q109 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.5 |
| q110 | single-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.5 |
| q111 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q112 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q113 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q114 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q115 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q116 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.53 |
| q117 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.46 |
| q118 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.6 |
| q119 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q120 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q121 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q122 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q123 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q124 | adversarial | INJECT | 0 | 0 |  | abstention | 0.68 |
| q125 | adversarial | INJECT | 0 | 0 |  | abstention | 0.75 |
| q126 | adversarial | INJECT | 0 | 0 |  | abstention | 0.58 |
| q127 | adversarial | INJECT | 0 | 0 |  | abstention | 0.61 |
| q128 | adversarial | INJECT | 0 | 0 |  | abstention | 0.8 |
| q129 | adversarial | INJECT | 0 | 0 |  | abstention | 0.57 |
| q130 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q131 | adversarial | INJECT | 0 | 0 |  | abstention | 0.55 |
| q132 | adversarial | INJECT | 0 | 0 |  | abstention | 0.81 |
| q133 | adversarial | INJECT | 0 | 0 |  | abstention | 0.67 |
| q134 | adversarial | INJECT | 0 | 0 |  | abstention | 0.66 |
| q135 | adversarial | INJECT | 0 | 0 |  | abstention | 0.59 |
| q136 | adversarial | INJECT | 0 | 0 |  | abstention | 0.78 |
| q137 | adversarial | INJECT | 0 | 0 |  | abstention | 0.68 |
| q138 | adversarial | INJECT | 0 | 0 |  | abstention | 0.55 |
| q139 | adversarial | INJECT | 0 | 0 |  | abstention | 0.64 |
| q140 | adversarial | INJECT | 0 | 0 |  | abstention | 0.63 |
| q141 | adversarial | INJECT | 0 | 0 |  | abstention | 0.85 |
| q142 | adversarial | INJECT | 0 | 0 |  | abstention | 0.6 |
| q143 | adversarial | INJECT | 0 | 0 |  | abstention | 0.54 |
| q144 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.5 |
| q145 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q146 | adversarial | INJECT | 0 | 0 |  | abstention | 0.86 |
| q147 | adversarial | INJECT | 0 | 0 |  | abstention | 0.62 |
| q148 | adversarial | INJECT | 0 | 0 |  | abstention | 0.62 |
| q149 | adversarial | INJECT | 0 | 0 |  | abstention | 0.78 |
| q150 | adversarial | INJECT | 0 | 0 |  | abstention | 0.72 |
| q151 | adversarial | INJECT | 0 | 0 |  | abstention | 0.66 |
| q152 | adversarial | INJECT | 0 | 0 |  | abstention | 0.5 |
| q153 | adversarial | INJECT | 0 | 0 |  | abstention | 0.5 |
| q154 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.5 |
| q155 | adversarial | INJECT | 0 | 0 |  | abstention | 0.66 |
| q156 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q157 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q158 | adversarial | INJECT | 0 | 0 |  | abstention | 0.68 |
