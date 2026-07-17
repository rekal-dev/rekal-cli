#!/usr/bin/env bash
# Full LongMemEval-S TEST-split eval (400 conversations × stock + skill).
# Prerequisites: ingest under ~/imb-lme-s, flat symlinks, indexed repos,
# frozen calibration/longmemeval-s.json.
#
# Usage:
#   export REKAL=./rekal WORKERS=8
#   ./scripts/industry-bench/run_full_test.sh
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
IMB_ROOT="${IMB_ROOT:-$HOME/imb-lme-s}"
FLAT="${FLAT:-$IMB_ROOT/flat}"
REKAL="${REKAL:-$REPO_ROOT/rekal}"
WORKERS="${WORKERS:-8}"
DATE_TAG="${DATE_TAG:-fulltest-$(date +%Y%m%d)}"
CAL="${CAL:-$REPO_ROOT/scripts/industry-bench/calibration/longmemeval-s.json}"
TEST_IDS="$REPO_ROOT/scripts/industry-bench/calibration/split-longmemeval-s-test-ids.txt"
TEST_IN="$REPO_ROOT/scripts/industry-bench/calibration/split-longmemeval-s-test-conversations.jsonl"
SHIM="$REPO_ROOT/scripts/industry-bench/shim/shim.py"
RUNS="$REPO_ROOT/scripts/industry-bench/runs/full/longmemeval-s"
LOG="$IMB_ROOT/full-test.log"

log() { echo "[$(date -Iseconds)] $*" | tee -a "$LOG"; }

if [[ ! -x "$REKAL" ]]; then
  echo "missing rekal at $REKAL" >&2
  exit 1
fi
if [[ ! -f "$CAL" ]]; then
  echo "missing frozen calibration $CAL — copy from chat-provisional or tune first" >&2
  exit 1
fi
if [[ ! -f "$TEST_IDS" || ! -f "$TEST_IN" ]]; then
  echo "missing test split — run calibration/split_dataset.py" >&2
  exit 1
fi

# Flatten shards → flat/
mkdir -p "$FLAT"
for shard in "$IMB_ROOT"/shard-*/; do
  [[ -d "$shard" ]] || continue
  for conv in "$shard"*/; do
    [[ -d "${conv}repo/.rekal" ]] || continue
    name="$(basename "$conv")"
    [[ -e "$FLAT/$name" ]] || ln -sfn "$(cd "$conv" && pwd)" "$FLAT/$name"
  done
done
n_flat="$(find "$FLAT" -maxdepth 1 -type l | wc -l | tr -d ' ')"
log "flat conversations=$n_flat"

# Index missing
log "indexing (workers=$WORKERS)"
REKAL="$REKAL" WORKERS="$WORKERS" bash "$REPO_ROOT/scripts/industry-bench/run_index_all.sh" "$FLAT" \
  2>&1 | tee -a "$LOG"

mkdir -p "$RUNS"
run_one() {
  local route=$1 conv=$2
  local out="$RUNS/${conv}-${route}-${DATE_TAG}"
  if [[ -f "$out/manifest.json" ]]; then
    echo "skip $conv $route"
    return 0
  fi
  if [[ ! -d "$FLAT/$conv/repo/.rekal" ]]; then
    echo "missing repo $conv" >&2
    return 0
  fi
  echo "start $conv $route"
  if [[ "$route" == skill ]]; then
    python3 "$SHIM" smoke \
      --workdir "$FLAT" --conversation-id "$conv" --input "$TEST_IN" \
      --out "$out" --rekal "$REKAL" --route skill --calibration "$CAL"
  else
    python3 "$SHIM" smoke \
      --workdir "$FLAT" --conversation-id "$conv" --input "$TEST_IN" \
      --out "$out" --rekal "$REKAL" --route stock
  fi
  echo "done $conv $route"
}
export -f run_one
export FLAT TEST_IN RUNS DATE_TAG REKAL SHIM CAL

JOBS=()
while IFS= read -r conv; do
  [[ -n "$conv" ]] || continue
  JOBS+=("stock $conv")
  JOBS+=("skill $conv")
done < "$TEST_IDS"

log "full test jobs=${#JOBS[@]} workers=$WORKERS cal=$(basename "$CAL") tag=$DATE_TAG"
JOBFILE="$(mktemp)"
printf '%s\n' "${JOBS[@]}" > "$JOBFILE"
xargs -P "$WORKERS" -n 2 bash -c 'run_one "$1" "$2"' _ < "$JOBFILE"
rm -f "$JOBFILE"

python3 "$REPO_ROOT/scripts/industry-bench/scoring/aggregate_smokes.py" \
  --smoke-dir "$RUNS" \
  --out "$RUNS/aggregate-${DATE_TAG}.json" 2>&1 | tee -a "$LOG"

log "FULL TEST COMPLETE → $RUNS/aggregate-${DATE_TAG}.md"
