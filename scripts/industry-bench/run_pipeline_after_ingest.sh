#!/usr/bin/env bash
# Wait for LME-S shard ingest, then index + dev split + dev smokes.
# Run in background while away:
#   cd rekal-cli && export REKAL=$PWD/rekal && \
#   nohup bash scripts/industry-bench/run_pipeline_after_ingest.sh \
#     > ~/imb-lme-s-pipeline.log 2>&1 &
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
IMB_ROOT="${IMB_ROOT:-$HOME/imb-lme-s}"
REKAL="${REKAL:-$REPO_ROOT/rekal}"
WORKERS="${WORKERS:-8}"
SHARD_SIZE="${SHARD_SIZE:-25}"
TOTAL_CONV=500
EXPECTED_SHARDS=$(( (TOTAL_CONV + SHARD_SIZE - 1) / SHARD_SIZE ))
LOG="$IMB_ROOT/pipeline.log"

log() { echo "[$(date -Iseconds)] $*" | tee -a "$LOG"; }

log "pipeline start imb=$IMB_ROOT workers=$WORKERS expected_shards=$EXPECTED_SHARDS"

# 1. Wait for ingest shards (or finish launching if parent died)
wait_shards() {
  while true; do
    local done=0
    local failed=0
    local i=0
    while [[ $i -lt $TOTAL_CONV ]]; do
      local shard
      shard="$(printf '%s/shard-%04d' "$IMB_ROOT" "$i")"
      # Soft verify failures still ingested the repos — count as finished.
      if [[ -f "$shard/ingest.log" ]] && grep -qE '^(done:|done offset=)' "$shard/ingest.log" 2>/dev/null; then
        done=$((done + 1))
      elif [[ -f "$shard/ingest.log" ]] && grep -qi 'VERIFICATION FAILED' "$shard/ingest.log"; then
        done=$((done + 1))
        failed=$((failed + 1))
      fi
      i=$((i + SHARD_SIZE))
    done
    local running
    running="$(pgrep -f "sh_gen/gen.py.*imb-lme-s" 2>/dev/null | wc -l | tr -d ' ')"
    log "shards_finished=$done/$EXPECTED_SHARDS (verify_soft_fail=$failed) ingest_procs=$running"
    if [[ "$done" -ge "$EXPECTED_SHARDS" ]]; then
      if [[ "$failed" -gt 0 ]]; then
        log "WARN: $failed shards had soft verify fails (turn±1 / date off-by-one); continuing"
      fi
      return 0
    fi
    # Relaunch only unfinished shards if no workers left
    if [[ "$running" == "0" && "$done" -lt "$EXPECTED_SHARDS" ]]; then
      log "relaunching missing shards via run_ingest_shards"
      bash "$REPO_ROOT/scripts/industry-bench/run_ingest_shards.sh" "$IMB_ROOT" "$WORKERS" "$SHARD_SIZE" \
        >> "$IMB_ROOT/ingest-relaunch.log" 2>&1 &
    fi
    sleep 120
  done
}

wait_shards
log "ingest complete"

# 2. Flatten shard dirs into single workdir for smokes (symlink conversations)
FLAT="${IMB_ROOT}/flat"
mkdir -p "$FLAT"
for shard in "$IMB_ROOT"/shard-*/; do
  [[ -d "$shard" ]] || continue
  for conv in "$shard"*/; do
    [[ -d "${conv}repo/.rekal" ]] || continue
    name="$(basename "$conv")"
    if [[ ! -e "$FLAT/$name" ]]; then
      ln -sfn "$(cd "$conv" && pwd)" "$FLAT/$name"
    fi
  done
done
log "flattened to $FLAT ($(ls -1 "$FLAT" | wc -l | tr -d ' ') conversations)"

# 3. Index all repos
REKAL="$REKAL" WORKERS="$WORKERS" bash "$REPO_ROOT/scripts/industry-bench/run_index_all.sh" "$FLAT" \
  2>&1 | tee -a "$LOG"

# 4. Dev/test split
python3 "$REPO_ROOT/scripts/industry-bench/calibration/split_dataset.py" \
  --input "$REPO_ROOT/scripts/industry-bench/datasets/data/longmemeval-s-conversations.jsonl" \
  --name longmemeval-s --dev-ratio 0.2 --seed 42 \
  --out-dir "$REPO_ROOT/scripts/industry-bench/calibration" \
  2>&1 | tee -a "$LOG"

# 5. Dev smokes (skill + stock) — parallel by conversation
DEV_IDS="$REPO_ROOT/scripts/industry-bench/calibration/split-longmemeval-s-dev-ids.txt"
SHIM="$REPO_ROOT/scripts/industry-bench/shim/shim.py"
CAL="$REPO_ROOT/scripts/industry-bench/calibration/chat-provisional.json"
DEV_IN="$REPO_ROOT/scripts/industry-bench/calibration/split-longmemeval-s-dev-conversations.jsonl"
RUNS="$REPO_ROOT/scripts/industry-bench/runs/dev"
mkdir -p "$RUNS"

run_dev() {
  local route=$1 conv=$2
  local out="$RUNS/lme-s-${conv}-${route}-dev-20260717"
  [[ -f "$out/manifest.json" ]] && return 0
  if [[ "$route" == skill ]]; then
    python3 "$SHIM" smoke --workdir "$FLAT" --conversation-id "$conv" \
      --input "$DEV_IN" --out "$out" --rekal "$REKAL" --route skill --calibration "$CAL"
  else
    python3 "$SHIM" smoke --workdir "$FLAT" --conversation-id "$conv" \
      --input "$DEV_IN" --out "$out" --rekal "$REKAL" --route stock
  fi
}
export -f run_dev
export FLAT DEV_IN RUNS REKAL SHIM CAL

while IFS= read -r conv; do
  [[ -n "$conv" ]] || continue
  echo "stock $conv"
  echo "skill $conv"
done < "$DEV_IDS" | xargs -P "$WORKERS" -n 2 bash -c 'run_dev "$1" "$2"' _ \
  2>&1 | tee -a "$LOG"

python3 "$REPO_ROOT/scripts/industry-bench/scoring/aggregate_smokes.py" \
  --smoke-dir "$RUNS" \
  --out "$RUNS/aggregate-dev-20260717.json" 2>&1 | tee -a "$LOG"

log "pipeline finished — see $RUNS/aggregate-dev-20260717.md"
