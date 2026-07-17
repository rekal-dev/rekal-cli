# Parallel streams — one workdir per dataset / workstream

The adapter already isolates corpora: **one synthetic git repo per
conversation**, grouped under a **workdir per ingest campaign**. Different
workstreams can run on different datasets in parallel without touching each
other.

```
~/imb-toy/              # toy (WS-B round-trip)
~/imb-lme-s/            # LongMemEval-S full or sharded (WS-D/E primary)
~/imb-lme-oracle/       # evidence-only sessions (cheap WS-C/D smoke)
~/imb-locomo/           # LoCoMo 10 conversations (comparability appendix)
~/imb-lme-m/            # LongMemEval-M (scale column) — when raw download is valid
```

Nothing in Rekal core ties these together. The only shared inputs are the
normalized `conversations.jsonl` files under `datasets/data/` (gitignored raw).

## What each stream needs

| Stream | WS | Input jsonl | Workdir | Notes |
|---|---|---|---|---|
| toy | B smoke | `datasets/toy/conversations.jsonl` | `/tmp/imb-toy-local` | committed corpus |
| LME-S | D, E, F | `data/longmemeval-s-conversations.jsonl` | `~/imb-lme-s` | 500 × ~48 sess; shard with `run_ingest_shards.sh` |
| LME oracle | C, D smoke | `data/longmemeval-oracle-conversations.jsonl` | `~/imb-lme-oracle` | 500 × ~2 sess; fast ingest for gate/mode tuning |
| LoCoMo | E appendix | `data/locomo-conversations.jsonl` | `~/imb-locomo` | 10 × ~27 sess |
| LME-M | E scale | `data/longmemeval-m-conversations.jsonl` | `~/imb-lme-m` | **blocked** until raw re-download (local file corrupt) |
| MSC | E smoke | *not wired* | `~/imb-msc` | WS-A getter still open |

## Ingest one stream

```bash
export REKAL="$PWD/rekal"
STREAM=oracle   # toy | lme-s | oracle | locomo
IMB_ROOT="$HOME/imb-${STREAM//-/_}"   # imb_oracle, imb_lme_s, …

case "$STREAM" in
  toy)    IN=scripts/industry-bench/datasets/toy/conversations.jsonl; LIMIT=1 ;;
  lme-s)  IN=scripts/industry-bench/datasets/data/longmemeval-s-conversations.jsonl; LIMIT=3 ;;
  oracle) IN=scripts/industry-bench/datasets/data/longmemeval-oracle-conversations.jsonl; LIMIT=3 ;;
  locomo) IN=scripts/industry-bench/datasets/data/locomo-conversations.jsonl; LIMIT=0 ;;
esac

python3 scripts/industry-bench/sh_gen/gen.py \
  --input "$IN" \
  ${LIMIT:+--limit "$LIMIT"} \
  --out "$IMB_ROOT" \
  --rekal "$REKAL" \
  --verify --index
```

Use `--offset` / `--limit` for shards (LME-S full run: `run_ingest_shards.sh`).

## Smoke one stream

```bash
export REKAL="$PWD/rekal"
CAL=scripts/industry-bench/calibration/chat-provisional.json

# Pick conversation id from the jsonl (first line: jq -r .conversation_id)
python3 scripts/industry-bench/shim/shim.py smoke \
  --workdir "$IMB_ROOT" \
  --conversation-id <id> \
  --input "$IN" \
  --out scripts/industry-bench/runs/smoke/<stream>-<id>-skill-20260717 \
  --rekal "$REKAL" --route skill --calibration "$CAL"
```

Or batch with parallel smokes (uses all cores; safe across conversations):

```bash
export REKAL=./rekal WORKERS=8
./scripts/industry-bench/run_smokes_parallel.sh locomo   # 10 conv × stock+skill
./scripts/industry-bench/run_smokes_parallel.sh oracle   # evidence-only stream
./scripts/industry-bench/run_smokes_parallel.sh lme-s    # 3-conv smoke set
```

Full LongMemEval-S ingest (8 workers, ~4–5 h):

```bash
export REKAL=./rekal
./scripts/industry-bench/run_ingest_shards.sh ~/imb-lme-s 8 25
```

## Isolation guarantees

- **Git repos**: one per conversation under `<workdir>/<conversation-id>/repo/`
- **Claude discovery**: `CLAUDE_CONFIG_DIR=<workdir>/<id>/claude-config` — no bleed to `~/.claude`
- **Rekal store**: `.rekal/` inside each synthetic repo only
- **Calibration**: per-stream JSON in `calibration/<dataset>.json` (WS-D); shim passes via `--calibration` + `REKAL_HUNT_*`

## Running workstreams in parallel

| Agent / terminal | Branch focus | Workdir | Safe to run alongside |
|---|---|---|---|
| WS-C modes | classify + persona on dev split | `~/imb-lme-oracle` (small) | yes |
| WS-D calibration | grid on dev split | `~/imb-lme-s` shard-0000 | yes |
| WS-E smokes | stock/skill eval | `~/imb-locomo` | yes (DuckDB lock per repo) |
| WS-F baselines | mem0/zep on same jsonl | separate `~/imb-*-mem0` workdir | yes (different system) |

DuckDB locks are **per repo** — parallel smokes on *different* conversations are fine; same conversation needs serialization (shim retries lock).

## Status on this machine (2026-07-17)

| Stream | Normalized | Ingested locally | Smoked |
|---|---|---|---|
| toy | yes | `/tmp/imb-toy-local` | yes |
| LME-S (3) | yes | `/tmp/imb-lme-s-smoke` | yes |
| LME-S (500) | yes | `~/imb-lme-s` (~90% ingest) | pending index+dev |
| LME oracle | yes | `~/imb-lme-oracle` | yes |
| LoCoMo (10) | yes | `~/imb-locomo` | yes (full retrieve) |
| LME-M | re-download | — | — |
| MSC | getter wired | — | — |
| BEAM | path only | — | — |

**Policy:** full tier on all five selected benchmarks — see `FULL_TIER.md`.
