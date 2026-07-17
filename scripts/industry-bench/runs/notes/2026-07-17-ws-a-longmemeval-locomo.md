# WS-A: LongMemEval-S + LoCoMo real data flowing

Date: 2026-07-17 · Machine: linux/amd64 cloud agent ·
Branch: `cursor/industry-bench-longmemeval-real-ff4f`

## What ran

1. `datasets/get_locomo.sh` → `locomo10.json` (2.8 MB)
2. `datasets/normalize_locomo.py` → `locomo-conversations.jsonl`
3. Downloaded `longmemeval_s_cleaned.json` from
   `xiaowu0162/longmemeval-cleaned` (277 MB; SHA recorded at download time
   via `get_longmemeval.sh` on subsequent runs)
4. `datasets/normalize_longmemeval.py --variant s`
5. `datasets/verify_dataset.py {longmemeval-s,locomo,toy}` — all green
   after pinning toy question count to 4

## Verified counts

| Dataset | conversations | sessions | turns | questions | abstention |
|---|---:|---:|---:|---:|---:|
| LongMemEval-S (cleaned) | 500 | 23867 | 246750 | 500 | 30 |
| LoCoMo (locomo10) | 10 | 272 | 5882 | 1986 | 0 (adversarial=446) |
| toy | 1 | 3 | 12 | 4 | 1 |

LongMemEval-S categories (after `_abs` → abstention override):
single-session-user 64, single-session-assistant 56,
single-session-preference 30, multi-session 121, temporal-reasoning 127,
knowledge-update 72, abstention 30.

## Still open (WS-A DoD)

- LongMemEval-M getter path exists (`get_longmemeval.sh m`) — not downloaded
  yet (multi-GB).
- MSC normalizer not started.
- Penfield LoCoMo known-bad label list (`locomo-known-bad.jsonl`) not yet
  fetched (04-procedures §2 step 4).

## Next

Build `rekal`, run `sh_gen` on LongMemEval-S `--limit 1 --verify` then
scale toward full ingest (WS-B DoD #3).
