# LME-S parallel ingest progress

Date: 2026-07-17 · while limit-20 + shards 20–40 / 40–60 run

| Worker | out | status (snapshot) |
|---|---|---|
| limit-20 (0–19) | `/tmp/imb-lme-s-limit20` | running; ~5/20 at ~12 min |
| shard 20–40 | `/tmp/imb-lme-s-s20-40` | running |
| shard 40–60 | `/tmp/imb-lme-s-s40-60` | running |

`sh_gen/gen.py` gained `--offset` for cleaner sharding without pre-sliced
jsonl files. Full 500 ≈ 25 shards × 20 at ~40–50 min/shard wall with 3-way
parallelism ≈ **several hours** on this machine class (better than 12h
serial).

Do not commit workdir corpora (temp). Update this note when workers exit.
