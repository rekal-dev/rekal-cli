# smoke conv-43

- questions: 242
- route: skill
- calibration: locomo-bm25-push.json
- evidence@5 (answerable): 0.93
- evidence@10 (answerable): 0.95
- retrieved_context_tokens mean: 174.1
- answer_path_tokens mean: 205.0

| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |
|---|---|---|---:|---:|---:|---|---:|
| q1 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q2 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q3 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q4 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.69 |
| q5 | multi-hop | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.55 |
| q6 | open-domain | INJECT | 0 | 0 |  | true_miss | 0.72 |
| q7 | temporal | INJECT | 1 | 1 | 0 | hit | 0.75 |
| q8 | multi-hop | FALLBACK_STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.52 |
| q9 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q10 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q11 | temporal | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q12 | multi-hop | FALLBACK_STOCK | 1 | 1 | 4 | deep_rank_lt5 | 0.48 |
| q13 | temporal | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q14 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.74 |
| q15 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q16 | open-domain | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.33 |
| q17 | temporal | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q18 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.58 |
| q19 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q20 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q21 | temporal | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.5 |
| q22 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q23 | temporal | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.52 |
| q24 | temporal | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q25 | temporal | FALLBACK_STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.44 |
| q26 | multi-hop | FALLBACK_STOCK | 0 | 1 | 7 | deep_rank_lt10 | 0.49 |
| q27 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q28 | open-domain | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.66 |
| q29 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q30 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.89 |
| q31 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q32 | temporal | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q33 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q34 | temporal | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q35 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q36 | multi-hop | FALLBACK_STOCK | 1 | 1 | 2 | deep_rank_lt5 | 0.49 |
| q37 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q38 | temporal | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q39 | multi-hop | INJECT | 0 | 1 | 5 | deep_rank_lt10 | 0.73 |
| q40 | temporal | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q41 | temporal | INJECT | 0 | 0 |  | true_miss | 0.59 |
| q42 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.78 |
| q43 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q44 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q45 | temporal | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q46 | temporal | INJECT | 0 | 0 |  | true_miss | 0.61 |
| q47 | temporal | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.5 |
| q48 | temporal | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.53 |
| q49 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.88 |
| q50 | multi-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q51 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q52 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q53 | multi-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.55 |
| q54 | open-domain | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.64 |
| q55 | temporal | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q56 | temporal | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q57 | temporal | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q58 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q59 | temporal | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q60 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.55 |
| q61 | temporal | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q62 | multi-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.54 |
| q63 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q64 | temporal | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q65 | temporal | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q66 | multi-hop | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q67 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q68 | open-domain | INJECT | 0 | 1 | 7 | deep_rank_lt10 | 0.55 |
| q69 | temporal | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q70 | temporal | INJECT | 1 | 1 | 4 | deep_rank_lt5 | 0.55 |
| q71 | open-domain | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q72 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q73 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q74 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.95 |
| q75 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.93 |
| q76 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q77 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.78 |
| q78 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q79 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q80 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q81 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.62 |
| q82 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q83 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.54 |
| q84 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.89 |
| q85 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q86 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q87 | single-hop | FALLBACK_STOCK | 0 | 0 |  | true_miss | 0.46 |
| q88 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.61 |
| q89 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.5 |
| q90 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q91 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q92 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q93 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q94 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q95 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.7 |
| q96 | single-hop | INJECT | 1 | 1 | 2 | deep_rank_lt5 | 0.58 |
| q97 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q98 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.94 |
| q99 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q100 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q101 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q102 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.65 |
| q103 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q104 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.6 |
| q105 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q106 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q107 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q108 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q109 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.62 |
| q110 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q111 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q112 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q113 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.57 |
| q114 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.54 |
| q115 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.59 |
| q116 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.66 |
| q117 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q118 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.55 |
| q119 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.66 |
| q120 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q121 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q122 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q123 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.58 |
| q124 | single-hop | INJECT | 1 | 1 | 3 | deep_rank_lt5 | 0.79 |
| q125 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q126 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.68 |
| q127 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q128 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.48 |
| q129 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q130 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.9 |
| q131 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q132 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.85 |
| q133 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.76 |
| q134 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q135 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q136 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.89 |
| q137 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.88 |
| q138 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.74 |
| q139 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q140 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.73 |
| q141 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q142 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.92 |
| q143 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.66 |
| q144 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q145 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q146 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.54 |
| q147 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.92 |
| q148 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.82 |
| q149 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q150 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q151 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.72 |
| q152 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.52 |
| q153 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q154 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.84 |
| q155 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q156 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q157 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q158 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q159 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.87 |
| q160 | single-hop | FALLBACK_STOCK | 1 | 1 | 3 | deep_rank_lt5 | 0.53 |
| q161 | single-hop | FALLBACK_STOCK | 1 | 1 | 0 | hit | 0.54 |
| q162 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.77 |
| q163 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.81 |
| q164 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.64 |
| q165 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q166 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.8 |
| q167 | single-hop | INJECT | 0 | 0 |  | true_miss | 0.58 |
| q168 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q169 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q170 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.79 |
| q171 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.67 |
| q172 | single-hop | INJECT | 1 | 1 | 1 | deep_rank_lt5 | 0.58 |
| q173 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.69 |
| q174 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.83 |
| q175 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.71 |
| q176 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.63 |
| q177 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.78 |
| q178 | single-hop | INJECT | 1 | 1 | 0 | hit | 0.59 |
| q179 | adversarial | INJECT | 0 | 0 |  | abstention | 0.68 |
| q180 | adversarial | INJECT | 0 | 0 |  | abstention | 0.68 |
| q181 | adversarial | INJECT | 0 | 0 |  | abstention | 0.95 |
| q182 | adversarial | INJECT | 0 | 0 |  | abstention | 0.57 |
| q183 | adversarial | INJECT | 0 | 0 |  | abstention | 0.79 |
| q184 | adversarial | INJECT | 0 | 0 |  | abstention | 0.63 |
| q185 | adversarial | INJECT | 0 | 0 |  | abstention | 0.83 |
| q186 | adversarial | INJECT | 0 | 0 |  | abstention | 0.55 |
| q187 | adversarial | INJECT | 0 | 0 |  | abstention | 0.89 |
| q188 | adversarial | INJECT | 0 | 0 |  | abstention | 0.64 |
| q189 | adversarial | INJECT | 0 | 0 |  | abstention | 0.64 |
| q190 | adversarial | INJECT | 0 | 0 |  | abstention | 0.53 |
| q191 | adversarial | INJECT | 0 | 0 |  | abstention | 0.68 |
| q192 | adversarial | INJECT | 0 | 0 |  | abstention | 0.6 |
| q193 | adversarial | INJECT | 0 | 0 |  | abstention | 0.76 |
| q194 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q195 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q196 | adversarial | INJECT | 0 | 0 |  | abstention | 0.58 |
| q197 | adversarial | INJECT | 0 | 0 |  | abstention | 0.7 |
| q198 | adversarial | INJECT | 0 | 0 |  | abstention | 0.77 |
| q199 | adversarial | INJECT | 0 | 0 |  | abstention | 0.59 |
| q200 | adversarial | INJECT | 0 | 0 |  | abstention | 0.83 |
| q201 | adversarial | INJECT | 0 | 0 |  | abstention | 0.67 |
| q202 | adversarial | INJECT | 0 | 0 |  | abstention | 0.84 |
| q203 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.47 |
| q204 | adversarial | INJECT | 0 | 0 |  | abstention | 0.74 |
| q205 | adversarial | INJECT | 0 | 0 |  | abstention | 0.65 |
| q206 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.53 |
| q207 | adversarial | INJECT | 0 | 0 |  | abstention | 0.56 |
| q208 | adversarial | INJECT | 0 | 0 |  | abstention | 0.79 |
| q209 | adversarial | INJECT | 0 | 0 |  | abstention | 0.8 |
| q210 | adversarial | INJECT | 0 | 0 |  | abstention | 0.79 |
| q211 | adversarial | INJECT | 0 | 0 |  | abstention | 0.56 |
| q212 | adversarial | INJECT | 0 | 0 |  | abstention | 0.79 |
| q213 | adversarial | INJECT | 0 | 0 |  | abstention | 0.64 |
| q214 | adversarial | INJECT | 0 | 0 |  | abstention | 0.82 |
| q215 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.49 |
| q216 | adversarial | INJECT | 0 | 0 |  | abstention | 0.91 |
| q217 | adversarial | INJECT | 0 | 0 |  | abstention | 0.9 |
| q218 | adversarial | INJECT | 0 | 0 |  | abstention | 0.65 |
| q219 | adversarial | INJECT | 0 | 0 |  | abstention | 0.64 |
| q220 | adversarial | INJECT | 0 | 0 |  | abstention | 0.63 |
| q221 | adversarial | INJECT | 0 | 0 |  | abstention | 0.71 |
| q222 | adversarial | INJECT | 0 | 0 |  | abstention | 0.62 |
| q223 | adversarial | INJECT | 0 | 0 |  | abstention | 0.76 |
| q224 | adversarial | INJECT | 0 | 0 |  | abstention | 0.91 |
| q225 | adversarial | INJECT | 0 | 0 |  | abstention | 0.83 |
| q226 | adversarial | INJECT | 0 | 0 |  | abstention | 0.79 |
| q227 | adversarial | INJECT | 0 | 0 |  | abstention | 0.63 |
| q228 | adversarial | INJECT | 0 | 0 |  | abstention | 0.77 |
| q229 | adversarial | INJECT | 0 | 0 |  | abstention | 0.83 |
| q230 | adversarial | INJECT | 0 | 0 |  | abstention | 0.69 |
| q231 | adversarial | INJECT | 0 | 0 |  | abstention | 0.88 |
| q232 | adversarial | FALLBACK_STOCK | 0 | 0 |  | abstention | 0.52 |
| q233 | adversarial | INJECT | 0 | 0 |  | abstention | 0.76 |
| q234 | adversarial | INJECT | 0 | 0 |  | abstention | 0.79 |
| q235 | adversarial | INJECT | 0 | 0 |  | abstention | 0.63 |
| q236 | adversarial | INJECT | 0 | 0 |  | abstention | 0.78 |
| q237 | adversarial | INJECT | 0 | 0 |  | abstention | 0.67 |
| q238 | adversarial | INJECT | 0 | 0 |  | abstention | 0.78 |
| q239 | adversarial | INJECT | 0 | 0 |  | abstention | 0.59 |
| q240 | adversarial | INJECT | 0 | 0 |  | abstention | 0.83 |
| q241 | adversarial | INJECT | 0 | 0 |  | abstention | 0.73 |
| q242 | adversarial | INJECT | 0 | 0 |  | abstention | 0.79 |
