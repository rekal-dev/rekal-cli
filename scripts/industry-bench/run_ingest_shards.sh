#!/usr/bin/env bash
# Parallel LongMemEval-S ingest via sh_gen shards.
# Usage:
#   export REKAL=/path/to/rekal
#   ./scripts/industry-bench/run_ingest_shards.sh /path/to/workdir [workers] [shard_size]
#
# Example (8 workers, 25 conv/shard → 20 shards for 500 total):
#   ./scripts/industry-bench/run_ingest_shards.sh ~/imb-lme-s 8 25
set -euo pipefail

ROOT="${1:?workdir required}"
WORKERS="${2:-4}"
SHARD_SIZE="${3:-25}"
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
FULL="$REPO_ROOT/scripts/industry-bench/datasets/data/longmemeval-s-conversations.jsonl"
REKAL_BIN="${REKAL:-rekal}"

if [ ! -f "$FULL" ]; then
  echo "missing $FULL — run get_longmemeval.sh + normalize first" >&2
  exit 1
fi
if ! command -v "$REKAL_BIN" >/dev/null; then
  echo "rekal not found (set REKAL= or PATH)" >&2
  exit 1
fi

mkdir -p "$ROOT"
TOTAL=$(wc -l < "$FULL")
echo "conversations=$TOTAL shard_size=$SHARD_SIZE workers=$WORKERS out=$ROOT"

run_shard() {
  local offset=$1
  local out="$ROOT/shard-$(printf '%04d' "$offset")"
  mkdir -p "$out"
  PYTHONUNBUFFERED=1 python3 "$REPO_ROOT/scripts/industry-bench/sh_gen/gen.py" \
    --input "$FULL" \
    --offset "$offset" \
    --limit "$SHARD_SIZE" \
    --out "$out" \
    --rekal "$REKAL_BIN" \
    --verify \
    > "$out/ingest.log" 2>&1
  echo "done offset=$offset"
}
export -f run_shard
export FULL ROOT SHARD_SIZE REKAL_BIN REPO_ROOT

if command -v parallel >/dev/null; then
  seq 0 "$SHARD_SIZE" $((TOTAL - 1)) | parallel -j "$WORKERS" run_shard {}
else
  echo "GNU parallel not found; running shards sequentially" >&2
  offset=0
  while [ "$offset" -lt "$TOTAL" ]; do
    run_shard "$offset"
    offset=$((offset + SHARD_SIZE))
  done
fi

echo "all shards finished under $ROOT"
