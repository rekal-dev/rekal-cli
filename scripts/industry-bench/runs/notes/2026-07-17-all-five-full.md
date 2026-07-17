# Full-tier status — all 5 tracks (2026-07-17 20:47)

**Goal:** all five selected benchmarks at full corpus (not smoke-only).

| # | Benchmark | Full bar | Status now | Workdir / output |
|---|---|---|---|---|
| 1 | **LongMemEval-S** | 500 conv / 500 Q | Last 4 shards re-ingesting → then index + **400 TEST** | `~/imb-lme-s` → `runs/full/longmemeval-s/` |
| 2 | **LongMemEval-M** | 500 conv / **237k sessions** | Normalized ✓; shard-0000 ingesting (10 conv, `--fast 10`) | `~/imb-lme-m` — ETA **days** for all 500 |
| 3 | **LoCoMo** | 10 / 1986 Q | **Retrieval full done** (~91%@5); official LLM judge still open | `~/imb-locomo` + `runs/full/locomo/` |
| 4 | **MSC** | eval = test+valid (1501 conv) | First 4 shards (200 conv) ingesting; rest queued | `~/imb-msc` |
| 5 | **BEAM** | 128k tier first (then 500k/1m/10m) | HF download via `datasets` running | `~/imb-beam` |

## Honest capacity note

- **#1 + #3** fit a laptop overnight.
- **#4 MSC eval** (~5.5k sessions) overnight with `--fast`.
- **#2 LME-M** (~238k sessions) is the long pole — keep shards running with `--fast`; do not expect same-day completion.
- **#5 BEAM-10M** needs `--fast` and lots of disk; 128k is the entry full-tier.

## Watch

```bash
tail -f ~/imb-lme-s-wait-full.log    # #1 → full TEST
tail -f ~/imb-lme-m/shard-0000/ingest.log
tail -f ~/imb-msc/shard-0000/ingest.log
tail -f ~/imb-beam-download.log
```
