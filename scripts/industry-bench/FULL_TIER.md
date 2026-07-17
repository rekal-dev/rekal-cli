# Full-tier target (all selected benchmarks)

**Decision (2026-07-17):** aim **full corpus** on every selected benchmark —
not smoke/limit subsets as the end state. Smokes remain the green light
before each full run.

| # | Benchmark | Full-tier definition | Workdir | ETA order |
|---|---|---|---|---|
| 1 | **LongMemEval-S** | 500 conversations / ~23.8k sessions / 500 Q | `~/imb-lme-s` | **in progress** (ingest ~90%) |
| 2 | **LongMemEval-M** | cleaned full LongMemEval (multi-session haystacks) | `~/imb-lme-m` | re-download → normalize → shard ingest |
| 3 | **LoCoMo** | all 10 conversations / 1986 Q (official judge later) | `~/imb-locomo` | **ingest+retrieve smoke done**; official judge/eval next |
| 4 | **MSC** | full Multi-Session Chat (ParlAI / HF mirror) | `~/imb-msc` | getter+normalize+ingest (WS-A open) |
| 5 | **BEAM / AMB** | published BEAM tiers incl. 10M (stretch) | `~/imb-beam` | after F baselines + `--fast` ingest |

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
