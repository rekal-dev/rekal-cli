# Industry-bench run notes — 2026-07-17

Machine: darwin/arm64 · Rekal: **v0.2.27-15-g90ff222** (local `./rekal` build
from `research/industry-bench` — includes marker-only hunt-gate fix; release
0.2.27 on PATH is insufficient for skill-route smokes) · Branch:
`research/industry-bench` · Scoring lineage on.

## Datasets (WS-A)

| Dataset | Status | Counts |
|---|---|---|
| toy | verified | 1 / 3 / 12 / 4 |
| LoCoMo | verified | 10 / 272 / 5882 / 1986 |
| LongMemEval-S cleaned | verified | 500 / 23867 / 246750 / 500 |
| LongMemEval-M | deferred | HF resolve returned HTML stub from this net path; escalate if needed |
| MSC | deferred | ParlAI/HF path not yet wired; smoke-only per 01-selection |

Scripts: `get_longmemeval_s.sh`, `normalize_longmemeval.py`,
`verify_dataset.py` (toy/locomo/longmemeval-s).

## Smoke results

### Toy (stock + provisional)

- Ingest verify green on rekal 0.2.27.
- **Stock gates**: evidence_recall **1.0**, abstain SILENCE **1.0**, but
  all answerable routes were **SILENCE** (tok=0 injected). Ranking still
  put gold sessions on top.
- **Provisional chat calibration** (`calibration/chat-provisional.json`):
  INJECT on q1–q3, SILENCE on abstention, evidence_recall **1.0**.

### LongMemEval-S (limit 3, provisional)

- Ingest: 3 haystacks, ~53 sessions each, verify green.
- Eval: evidence_recall **1.0** (3/3), routes INJECT, tok ≈ 2.4k–3.3k/q.

### LoCoMo (1 conversation / 20 questions)

- Ingest verify green (19 sessions in conv-26).
- Provisional: evidence_recall **0.85** (17/20).
- Stock: same **0.85** ranking; more SILENCE routes (`below_mass`).
- Misses (q5, q8, q20): multi-hop evidence not in top-5 — **WS-C WHY
  route**, not a core bug. q8 also `below_mass` at conf 0.71 → keep
  lowering chat `MASS_MIN` in WS-D, or optional mainline PR to expose
  mass floor in config (do not hardcode chat bars into the skill).

## Negatives → triage (skill / config / mainline)

### N1. Knowledge layer hijacks chat QA under stock `recall-route`

- **Symptom**: marker files + `CLAUDE.md` score ≥ `KNOWLEDGE_MIN` and win
  `KNOWLEDGE` before episodes inject.
- **Fix (mainline, on branch)**: `hunt-gate.py` ignores marker-only knowledge
  (`_is_marker_only_knowledge`). Requires **local build** from this branch —
  `~/.local/bin/rekal` 0.2.27 alone does not ship this until PR #49 merges.
- **Triage**: **mainline PR #49** + local `./rekal` for harness runs.

### N2. Stock hunt bars silence true chat hits (`below_mass` / `below_gate`)

- **Symptom**: correct dog turn at conf≈0.67, mass below `MASS_MIN=3.5` →
  SILENCE. Sister (unanswerable) conf≈0.49 — separation exists.
- **Triage**: **config/calibration (WS-D)** — `calibration/chat-provisional.json`
  lowers CONF/MASS for smoke. **Must re-tune on LongMemEval-S dev and freeze
  before any test-split run** (04 §prereg). Hardcoding new bars into
  `hunt-gate.py` would ship chat-tuned gates to coding users — **do not**.
- **Mainline option (optional, separate PR)**: make `CONF_MIN`/`MASS_MIN`
  readable from `.rekal/config.json` / global config so WS-D writes one file
  the skill consumes. Not required for paper #2 if the shim keeps overrides.

### N3. macOS `/tmp` → `/private/tmp` path mismatch (ingest empty)

- **Symptom**: sh-gen wrote sessions under `-tmp-...` but `git rev-parse
  --show-toplevel` is `/private/tmp/...` → checkpoint found nothing.
- **Fix (local, adapter)**: `sh_gen/gen.py` now `Path.resolve()`s the repo
  before sanitizing / writing JSONL cwd.
- **Mainline?** No — discovery is correct; the generator was wrong.

### N4. `REKAL_BENCH=1` kill-switch (prior note)

- Already encoded: strip in sh-gen child env. Skill/core unchanged.

## What "whole benchmark" still needs

1. WS-D: freeze `calibration/longmemeval-s.json` on a seeded dev split.
2. WS-E: official LongMemEval / LoCoMo judge scripts + pinned answer model
   (tokens already logged).
3. Full LongMemEval-S ingest (500 × ~48 sessions) — hours on a laptop;
   smoke is the green light.
4. Full LoCoMo (10 conversations) + caveat block on any published number.
5. WS-F baselines once the shim+scorer path is stable.
6. MSC getter if we want the cheap regression smoke.

No core change required for the negatives above; fixes live in
`scripts/industry-bench/` (+ optional future config surface for gate bars).
