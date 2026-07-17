# Skill improvements vs harness gap (fixed 2026-07-17)

## What was wrong

Skill work landed in three places, but the LoCoMo smokes did not benefit:

1. **`hunt-gate.py` marker fix** — in source tree, but shim preferred **stale**
   `.claude/skills/rekal/scripts/` frozen at `rekal init` ingest time.
2. **`chat-provisional.json`** — documented in run notes, **never wired** into
   shim; all committed manifests said `"calibration": "stock"`.
3. **Confidence ordering** — only applied on `INJECT`; chat hits SILENCE on
   coding bars (`MASS_MIN=3.5`, `CONF_MIN=0.70`) then `FALLBACK_STOCK` kept
   **score** rank #0.
4. **Evidence matching** — substring `s1 in s13` inflated `evidence_in_top`.

## Fixes

| Layer | Change |
|---|---|
| `hunt-gate.py` | `REKAL_HUNT_*` env overrides; `MASS_MIN=0` disables mass floor |
| `shim.py` | Prefer source-tree skill scripts; `--calibration` → env; confidence sort on `FALLBACK_STOCK`; exact `-sN.md` evidence; evidence@5/@10 + `miss_reason` |
| `chat-provisional.json` | `MASS_MIN: 0` (chat mass ~0.1–0.2) |

## LoCoMo conv-26 q1–q10 after fix

| Route | evidence@5 | evidence@10 | q5 identity | q8 relationship |
|---|---:|---:|---|---|
| stock v2 | 0.70 | 1.00 | deep_rank_lt10 (was false hit) | deep_rank_lt10 |
| skill + provisional v2 | **0.80** | 1.00 | **hit rank 0 INJECT** | deep_rank_lt10 (true gap) |

q8 remains a retrieval/product gap (gold in s2/s3 at rank ~6–8); skill cannot
rescue when confidence also prefers the wrong session (s1). Needs WS-C WHY route
or better retrieval — not a gate tweak.
