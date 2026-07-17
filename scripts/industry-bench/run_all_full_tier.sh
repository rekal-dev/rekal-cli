#!/usr/bin/env bash
# Master driver: push all five full-tier tracks.
# Usage: export REKAL=./rekal; nohup bash scripts/industry-bench/run_all_full_tier.sh >> ~/imb-all-full.log 2>&1 &
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
REKAL="${REKAL:-$REPO_ROOT/rekal}"
export REKAL
WORKERS="${WORKERS:-6}"
LOG="${LOG:-$HOME/imb-all-full.log}"
log(){ echo "[$(date -Iseconds)] $*" | tee -a "$LOG"; }

log "=== ALL FULL TIER START workers=$WORKERS ==="

# --- 1. LME-S finish last shards + full test ---
finish_lme_s() {
  log "[1] LME-S finishing shards 400-475"
  for off in 400 425 450 475; do
    out=$(printf "$HOME/imb-lme-s/shard-%04d" "$off")
    if [[ -f "$out/ingest.log" ]] && grep -qE '^(done:)|VERIFICATION FAILED' "$out/ingest.log"; then
      log "[1] shard $off already done"
      continue
    fi
    rm -rf "$out"
    mkdir -p "$out"
    (
      PYTHONUNBUFFERED=1 python3 "$REPO_ROOT/scripts/industry-bench/sh_gen/gen.py" \
        --input "$REPO_ROOT/scripts/industry-bench/datasets/data/longmemeval-s-conversations.jsonl" \
        --offset "$off" --limit 25 --out "$out" --rekal "$REKAL" --verify \
        > "$out/ingest.log" 2>&1 || true
      log "[1] shard $off finished"
    ) &
  done
  wait
  log "[1] launching full TEST"
  DATE_TAG=fulltest-20260717 bash "$REPO_ROOT/scripts/industry-bench/run_full_test.sh" \
    >> "$HOME/imb-lme-s-full-test.log" 2>&1 || log "[1] full test exited non-zero"
  log "[1] LME-S DONE"
}

# --- 2. LME-M normalize + first shard ingest (full continues overnight) ---
start_lme_m() {
  log "[2] LME-M normalize"
  local raw="$REPO_ROOT/scripts/industry-bench/datasets/data/longmemeval_m_cleaned.json"
  local jsonl="$REPO_ROOT/scripts/industry-bench/datasets/data/longmemeval-m-conversations.jsonl"
  if [[ ! -f "$jsonl" ]]; then
    python3 "$REPO_ROOT/scripts/industry-bench/datasets/normalize_longmemeval.py" --variant m \
      >> "$HOME/imb-lme-m-normalize.log" 2>&1
  fi
  mkdir -p "$HOME/imb-lme-m"
  log "[2] LME-M ingest shard-0000 (25 conv) — more shards via run_ingest after"
  # Reuse ingest shards pattern but on M file
  PYTHONUNBUFFERED=1 python3 "$REPO_ROOT/scripts/industry-bench/sh_gen/gen.py" \
    --input "$jsonl" --offset 0 --limit 25 \
    --out "$HOME/imb-lme-m/shard-0000" --rekal "$REKAL" --verify --fast 5 \
    > "$HOME/imb-lme-m/shard-0000/ingest.log" 2>&1 || true
  log "[2] LME-M shard-0000 done — kick remaining in parallel"
  # Launch remaining offsets 25..475 in background (don't wait all here)
  for off in $(seq 25 25 475); do
    out=$(printf "$HOME/imb-lme-m/shard-%04d" "$off")
    mkdir -p "$out"
    (
      PYTHONUNBUFFERED=1 python3 "$REPO_ROOT/scripts/industry-bench/sh_gen/gen.py" \
        --input "$jsonl" --offset "$off" --limit 25 \
        --out "$out" --rekal "$REKAL" --verify --fast 5 \
        > "$out/ingest.log" 2>&1 || true
      log "[2] LME-M shard $off done"
    ) &
    # throttle to WORKERS concurrent
    while [[ "$(jobs -rp | wc -l | tr -d ' ')" -ge "$WORKERS" ]]; do sleep 30; done
  done
  wait
  log "[2] LME-M ALL SHARDS DONE"
}

# --- 3. LoCoMo: write final retrieval aggregate as full-tier retrieve result ---
finalize_locomo() {
  log "[3] LoCoMo aggregate retrieval (already ingested)"
  python3 "$REPO_ROOT/scripts/industry-bench/scoring/aggregate_smokes.py" \
    --smoke-dir "$REPO_ROOT/scripts/industry-bench/runs/smoke" \
    --out "$REPO_ROOT/scripts/industry-bench/runs/full/locomo/aggregate-retrieve-20260717.json"
  log "[3] LoCoMo retrieval DONE (official judge still separate)"
}

# --- 4. MSC: ingest test+valid first (full train follows) ---
start_msc() {
  log "[4] MSC prepare test+valid jsonl"
  local full="$REPO_ROOT/scripts/industry-bench/datasets/data/msc-conversations.jsonl"
  local subset="$REPO_ROOT/scripts/industry-bench/datasets/data/msc-eval-conversations.jsonl"
  python3 - <<PY
import json
from pathlib import Path
src=Path("$full")
out=Path("$subset")
n=0
with open(src) as f, open(out,"w") as o:
    for line in f:
        c=json.loads(line)
        if c.get("extra",{}).get("split") in ("test","valid") or c["conversation_id"].startswith(("msc-test-","msc-valid-")):
            o.write(line); n+=1
print(f"msc eval subset conversations={n}")
PY
  mkdir -p "$HOME/imb-msc"
  log "[4] MSC ingest eval subset (parallel offset shards of 50)"
  TOTAL=$(wc -l < "$subset")
  off=0
  while [[ $off -lt $TOTAL ]]; do
    out=$(printf "$HOME/imb-msc/shard-%04d" "$off")
    mkdir -p "$out"
    (
      PYTHONUNBUFFERED=1 python3 "$REPO_ROOT/scripts/industry-bench/sh_gen/gen.py" \
        --input "$subset" --offset "$off" --limit 50 \
        --out "$out" --rekal "$REKAL" --verify --fast 3 \
        > "$out/ingest.log" 2>&1 || true
      log "[4] MSC shard $off done"
    ) &
    while [[ "$(jobs -rp | wc -l | tr -d ' ')" -ge "$WORKERS" ]]; do sleep 20; done
    off=$((off + 50))
  done
  wait
  log "[4] MSC eval subset DONE"
}

# --- 5. BEAM 128K tier download + normalize + ingest ---
start_beam() {
  log "[5] BEAM getter + normalize + smoke ingest"
  bash "$REPO_ROOT/scripts/industry-bench/datasets/get_beam.sh" 128k \
    >> "$HOME/imb-beam-download.log" 2>&1 || log "[5] get_beam failed"
  python3 "$REPO_ROOT/scripts/industry-bench/datasets/normalize_beam.py" --tier 128k \
    >> "$HOME/imb-beam-download.log" 2>&1 || log "[5] normalize_beam failed"
  mkdir -p "$HOME/imb-beam"
  local jsonl="$REPO_ROOT/scripts/industry-bench/datasets/data/beam-128k-conversations.jsonl"
  if [[ -f "$jsonl" ]]; then
    PYTHONUNBUFFERED=1 python3 "$REPO_ROOT/scripts/industry-bench/sh_gen/gen.py" \
      --input "$jsonl" --limit 5 --out "$HOME/imb-beam" --rekal "$REKAL" --verify --index --fast 10 \
      > "$HOME/imb-beam/ingest-limit5.log" 2>&1 || true
    log "[5] BEAM 128k smoke ingest done — full tier continues separately"
  fi
}

# Run tracks: LoCoMo finalize sync; LME-S + MSC + BEAM + LME-M in background-friendly order
finalize_locomo
# LME-S is highest priority full test — run to completion in this process's first parallel group
finish_lme_s &
PID1=$!
start_beam &
PID5=$!
start_msc &
PID4=$!
start_lme_m &
PID2=$!

wait $PID1 || true
wait $PID5 || true
wait $PID4 || true
wait $PID2 || true
log "=== ALL FULL TIER DRIVER EXITED ==="
