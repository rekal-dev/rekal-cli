# Full-tier target (all selected benchmarks)

**Decision (2026-07-17):** aim **full corpus** on every selected benchmark —
not smoke/limit subsets as the end state. Smokes remain the green light
before each full run.

| # | Benchmark | Full-tier definition | Workdir | ETA order |
|---|---|---|---|---|
| 1 | **LongMemEval-S** | 500 conversations / ~23.8k sessions / 500 Q | `~/imb-lme-s` | finishing last shards → full TEST |
| 2 | **LongMemEval-M** | 500 conv / ~237k sessions | `~/imb-lme-m` | normalized; shard ingest with `--fast` (multi-day) |
| 3 | **LoCoMo** | all 10 conversations / 1986 Q | `~/imb-locomo` | **retrieval full done** (~91%@5); official judge open |
| 4 | **MSC** | test+valid first (1501), then train | `~/imb-msc` | eval shards ingesting |
| 5 | **BEAM** | 128k → 500k → 1m → 10m | `~/imb-beam` | getter+normalize wired; download running |

## Discipline (unchanged)

1. Smoke green on exact code that will run full.
2. Dev-split tune → freeze `calibration/<dataset>.json` → test-split once.
3. Official judge = headline; tokens-per-question mandatory.
4. LoCoMo always ships with known-bad caveat + with/without columns.
5. No cross-benchmark averaging.

## Parallel streams

Each benchmark gets its own workdir (see `streams/README.md`). Agents may
run LME-M / MSC / LoCoMo eval in parallel with LME-S ingest as long as they
do not share a synthetic repo.
