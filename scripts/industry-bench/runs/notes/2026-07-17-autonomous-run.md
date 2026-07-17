# Autonomous run — 2026-07-17 (user away)

## Done before pipeline

| Item | Result |
|---|---|
| LoCoMo full (10 conv × stock+skill) | **41 smoke manifests** in `runs/smoke/*-20260717/` |
| LME-S smoke (3 conv) | evidence@5 **1.0** stock+skill |
| Oracle stream (3 conv) | ingested + smoked |
| Dev/test split | **100 dev / 400 test** — `calibration/split-longmemeval-s-*` seed=42 |

### LoCoMo headline (retrieval evidence@5, skill+provisional)

| Conv | Questions | ev@5 | ev@10 |
|---|---:|---:|---:|
| conv-26 | 199 | 0.90 | 0.95 |
| conv-30 | 105 | 0.96 | 0.98 |
| conv-41 | 193 | 0.91 | 0.93 |
| conv-42 | 260 | 0.88 | 0.91 |
| conv-43 | 242 | 0.94 | 0.96 |
| conv-44 | 158 | 0.89 | 0.92 |
| conv-47 | 190 | 0.95 | 0.96 |
| conv-48 | 239 | 0.92 | 0.93 |
| conv-49 | 196 | 0.90 | 0.91 |
| conv-50 | 204 | 0.86 | 0.89 |

Aggregate: `runs/smoke/aggregate-20260717.md`

## In progress (background)

1. **LME-S ingest** — `~/imb-lme-s/shard-*/` (8 workers, ~20 shards total)
   - Monitor: `tail -f ~/imb-lme-s-ingest.log`
   - Per-shard: `tail -f ~/imb-lme-s/shard-0000/ingest.log`

2. **Pipeline watcher** — `run_pipeline_after_ingest.sh` (pid in `~/imb-lme-s-pipeline.log`)
   - Waits for all shards
   - Flattens to `~/imb-lme-s/flat/` (symlinks)
   - Parallel index (`run_index_all.sh`)
   - Dev smokes → `runs/dev/lme-s-*-dev-20260717/`

## Not started yet

- Gate grid tuning (`calibration/tune_gates.py`) — after dev smokes land
- Full test-split eval — after calibration frozen
- Official LongMemEval LLM judge — WS-E open

## Commands when back

```bash
# Status
grep shards_done ~/imb-lme-s/pipeline.log | tail -3
ls ~/imb-lme-s/shard-*/ingest.log | wc -l
grep -l 'done:' ~/imb-lme-s/shard-*/ingest.log | wc -l

# If pipeline died, restart:
export REKAL=$PWD/rekal
nohup bash scripts/industry-bench/run_pipeline_after_ingest.sh >> ~/imb-lme-s-pipeline.log 2>&1 &
```
