# Failed-case autopsy: stock evidence@rank0 vs skill route

Date: 2026-07-17 · stock smoke misses re-run through
`rekal | recall-route.py` (shipped skill gate, `CONF_MIN=0.70`).

## Cases

| Case | Stock failure mode | Skill `recall-route` | Verdict |
|---|---|---|---|
| **LoCoMo conv-26 q8** — "Caroline's relationship status?" gold `Single`, evidence s2/s3 | evidence **absent from top-5**; top is unrelated painting chat (score 0.85 / conf 0.57 / mass 2.18) | **KNOWLEDGE** (HEAD marker files) — wrong substrate | **True miss.** Retrieval never surfaces s2/s3; skill cannot rescue. |
| **LME-S `118b2229`** — commute length, gold `45 minutes each way` | evidence at **rank 2 by score** (score 0.64); rank 0 is coral-reef filler (0.85 / conf 0.55) | **INJECT top=0.75** (= evidence session; highest *confidence*) | **Metric miss, gate hit.** Skill confidence ordering finds the gold session; stock `@rank0` metric does not. |
| **LME-S `6f9b354f`** — bedroom paint color | evidence at **rank 1 by score**; rank 0 high score/conf but wrong topic; gold string is mid-turn in evidence session | **INJECT top=0.88** (= evidence; conf beats score-#1's 0.83) | **Same pattern.** Snippet truncation hid gold; session is correct under skill. |

## What this means for mixing skill + test

1. **Do not score only `evidence_in_top` on score-ranked `#0`.** Report also: evidence@k, confidence-top session id, and gate verdict (`INJECT`/`KNOWLEDGE`/`SILENCE`).
2. **Skill gate already helps 2/3 of these "failures"** by preferring absolute confidence over max-norm-ish score rank — exactly what `hunt-gate.py` was built for.
3. **LoCoMo q8 is the real product gap:** relationship-status multi-hop with weak lexical overlap; needs better retrieval or why-mode assembly, not a lower SILENCE bar.
4. Next shim flag: `--route skill` recording `gate`, `gate_top_conf`, `confidence_top_is_evidence` beside stock columns.
