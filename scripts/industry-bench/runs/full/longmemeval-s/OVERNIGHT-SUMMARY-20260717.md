# Overnight autonomous run — summary (2026-07-17T23:47:43+10:00)

## HEADLINE: self-improvement validated on held-out test
LME-S FULL TEST (400-conv test split), evidence@k:
| variant | ev@5 | ev@10 | ctx_tok |
|---|---:|---:|---:|
| stock | 0.9400 | 0.9575 | 385 |
| skill interim (bm25 .30) | 0.9450 | 0.9550 | 380 |
| skill DEV-TUNED (bm25 .55) | 0.9600 | 0.9675 | 382 |
Dev-tuned = +2.0pt@5 / +1.0pt@10 vs stock; +1.5 / +1.25 vs interim skill.
Method (rigorous): swept 5 weight profiles on the 100-conv DEV split
(runs/sweep/lme-s-dev-20260717-2328, bm25-push won 0.9697 vs interim 0.9394),
froze calibration/longmemeval-s-tuned.json, validated on the 400 TEST convs.
Reports:
- runs/full/longmemeval-s/aggregate-tuned-vs-interim-20260717.md
- runs/full/longmemeval-s/aggregate-fulltest-20260717.md (interim full test)

## Other completed
- LoCoMo weight sweep: bm25-push won ev@5 0.8927 vs stock 0.8742
  (runs/sweep/locomo-weights-20260717-2208/CONCLUSION.md). Same direction as LME-S
  → BM25-weighted retrieval generalizes across chat-QA benchmarks.
- BEAM: 20/20 convs ingested (root causes fixed). No retrieval@k metric — BEAM
  ships no evidence_session_ids; it needs the (deferred) LLM answer-quality judge.

## Root causes fixed tonight (deep, not band-aids)
1. normalize_beam.py: nested chat sessions + ast-parsed probing_questions +
   ISO date parse for time_anchor.
2. sh_gen/gen.py verify: --fast batch-aware commit-backdating check.
3. shim run_rekal: retry the transient "index not built/rebuilding" state
   (was hard-fail; dropped 114 skill runs under load).
4. daemon_action_loop: duplicate full-test launch guard (refill lock + aggregate
   file), and BEAM backoff removed after fixing normalizer.

## Still running (action loop, 5-min ticks)
- LME-M ingest: 79/500 dbs; hours to go; disk guard pauses < 8Gi.
- Disk: 54Gi free.

## Recommendation for you
- Adopt longmemeval-s-tuned.json as the new frozen LME-S calibration (bm25 .55).
  Interim kept for pre-registration audit trail; not overwritten retroactively.
- Consider same bm25-push tune for other chat-QA benchmarks (LoCoMo already
  confirms). Repo's own recall weights NOT changed (cross-domain overfit risk).

## Monitors: ~/imb-action-loop.log  ~/imb-lme-s-dev-sweep.log  ~/imb-lme-s-tuned-test.log
