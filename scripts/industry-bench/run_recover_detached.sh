#!/usr/bin/env bash
# Detached recover tracks + watchdog. Survives agent shell exit.
# Each track is its own nohup process. Watchdog relaunches incomplete work.
#
#   export REKAL=./rekal
#   bash scripts/industry-bench/run_recover_detached.sh
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
REKAL="${REKAL:-$REPO_ROOT/rekal}"
export REKAL PATH="$REPO_ROOT:$PATH"
LOGDIR="${LOGDIR:-$HOME}"
STAMP=$(date +%Y%m%d-%H%M%S)
MASTER="$LOGDIR/imb-recover-detached.log"

log(){ echo "[$(date -Iseconds)] $*" | tee -a "$MASTER"; }

shard_terminal() {
  local logf=$1
  [[ -f "$logf" ]] || return 1
  grep -qE '^(done:)' "$logf" 2>/dev/null && return 0
  grep -qi 'VERIFICATION FAILED' "$logf" 2>/dev/null && return 0
  return 1
}

# --- LME-S: finish last 4 shards then full test ---
launch_lme_s() {
  local lock="$HOME/imb-lme-s/.recover.lock"
  mkdir -p "$HOME/imb-lme-s"
  if [[ -f "$lock" ]] && kill -0 "$(cat "$lock")" 2>/dev/null; then
    log "LME-S recover already running pid=$(cat "$lock")"
    return 0
  fi
  nohup bash -c '
    set -euo pipefail
    REPO_ROOT="'"$REPO_ROOT"'"; REKAL="'"$REKAL"'"
    JSONL="$REPO_ROOT/scripts/industry-bench/datasets/data/longmemeval-s-conversations.jsonl"
    echo "[$(date -Iseconds)] LME-S track start" 
    for off in 400 425 450 475; do
      out=$(printf "$HOME/imb-lme-s/shard-%04d" "$off")
      ndb=$(find "$out" -name data.db 2>/dev/null | wc -l | tr -d " ")
      if [[ -f "$out/ingest.log" ]] && grep -qE "^(done:)|VERIFICATION FAILED" "$out/ingest.log" 2>/dev/null && [[ "${ndb:-0}" -ge 20 ]]; then
        echo "[$(date -Iseconds)] skip shard $off dbs=$ndb"
        continue
      fi
      echo "[$(date -Iseconds)] ingest shard $off"
      rm -rf "$out"; mkdir -p "$out"
      PYTHONUNBUFFERED=1 python3 "$REPO_ROOT/scripts/industry-bench/sh_gen/gen.py" \
        --input "$JSONL" --offset "$off" --limit 25 --out "$out" --rekal "$REKAL" --verify \
        > "$out/ingest.log" 2>&1 || true
      echo "[$(date -Iseconds)] shard $off exit dbs=$(find "$out" -name data.db | wc -l | tr -d " ")"
    done
    echo "[$(date -Iseconds)] launching full TEST"
    DATE_TAG=fulltest-20260717 bash "$REPO_ROOT/scripts/industry-bench/run_full_test.sh" \
      >> "$HOME/imb-lme-s-full-test.log" 2>&1 || true
    echo "[$(date -Iseconds)] LME-S track done"
    rm -f "'"$lock"'"
  ' >> "$HOME/imb-lme-s-recover-track.log" 2>&1 &
  echo $! > "$lock"
  log "LME-S track pid=$! log=$HOME/imb-lme-s-recover-track.log"
}

# --- MSC eval ---
launch_msc() {
  local lock="$HOME/imb-msc/.recover.lock"
  mkdir -p "$HOME/imb-msc"
  if [[ -f "$lock" ]] && kill -0 "$(cat "$lock")" 2>/dev/null; then
    log "MSC recover already running pid=$(cat "$lock")"
    return 0
  fi
  nohup bash -c '
    set -euo pipefail
    REPO_ROOT="'"$REPO_ROOT"'"; REKAL="'"$REKAL"'"
    SUBSET="$REPO_ROOT/scripts/industry-bench/datasets/data/msc-eval-conversations.jsonl"
    WORKERS=3
    echo "[$(date -Iseconds)] MSC track start"
    TOTAL=$(wc -l < "$SUBSET" | tr -d " ")
    off=0
    while [[ $off -lt $TOTAL ]]; do
      out=$(printf "$HOME/imb-msc/shard-%04d" "$off")
      if [[ -f "$out/ingest.log" ]] && grep -qE "^(done:)|VERIFICATION FAILED" "$out/ingest.log" 2>/dev/null; then
        echo "[$(date -Iseconds)] skip MSC $off"
      else
        echo "[$(date -Iseconds)] MSC ingest $off"
        rm -rf "$out"; mkdir -p "$out"
        (
          PYTHONUNBUFFERED=1 python3 "$REPO_ROOT/scripts/industry-bench/sh_gen/gen.py" \
            --input "$SUBSET" --offset "$off" --limit 50 --out "$out" --rekal "$REKAL" --verify --fast 3 \
            > "$out/ingest.log" 2>&1 || true
          echo "[$(date -Iseconds)] MSC $off done"
        ) &
        while [[ "$(jobs -rp | wc -l | tr -d " ")" -ge "$WORKERS" ]]; do sleep 20; done
      fi
      off=$((off + 50))
    done
    wait
    echo "[$(date -Iseconds)] MSC track done"
    rm -f "'"$lock"'"
  ' >> "$HOME/imb-msc-recover-track.log" 2>&1 &
  echo $! > "$lock"
  log "MSC track pid=$! log=$HOME/imb-msc-recover-track.log"
}

# --- LME-M (throttled) ---
launch_lme_m() {
  local lock="$HOME/imb-lme-m/.recover.lock"
  mkdir -p "$HOME/imb-lme-m"
  if [[ -f "$lock" ]] && kill -0 "$(cat "$lock")" 2>/dev/null; then
    log "LME-M recover already running pid=$(cat "$lock")"
    return 0
  fi
  nohup bash -c '
    set -euo pipefail
    REPO_ROOT="'"$REPO_ROOT"'"; REKAL="'"$REKAL"'"
    JSONL="$REPO_ROOT/scripts/industry-bench/datasets/data/longmemeval-m-conversations.jsonl"
    WORKERS=2
    echo "[$(date -Iseconds)] LME-M track start"
    for off in $(seq 0 25 475); do
      out=$(printf "$HOME/imb-lme-m/shard-%04d" "$off")
      ndb=$(find "$out" -name data.db 2>/dev/null | wc -l | tr -d " ")
      if [[ -f "$out/ingest.log" ]] && grep -qE "^(done:)|VERIFICATION FAILED" "$out/ingest.log" 2>/dev/null && [[ "${ndb:-0}" -ge 20 ]]; then
        echo "[$(date -Iseconds)] skip M $off"
        continue
      fi
      echo "[$(date -Iseconds)] M ingest $off"
      rm -rf "$out"; mkdir -p "$out"
      (
        PYTHONUNBUFFERED=1 python3 "$REPO_ROOT/scripts/industry-bench/sh_gen/gen.py" \
          --input "$JSONL" --offset "$off" --limit 25 --out "$out" --rekal "$REKAL" --verify --fast 5 \
          > "$out/ingest.log" 2>&1 || true
        echo "[$(date -Iseconds)] M $off done"
      ) &
      while [[ "$(jobs -rp | wc -l | tr -d " ")" -ge "$WORKERS" ]]; do sleep 45; done
    done
    wait
    echo "[$(date -Iseconds)] LME-M track done"
    rm -f "'"$lock"'"
  ' >> "$HOME/imb-lme-m-recover-track.log" 2>&1 &
  echo $! > "$lock"
  log "LME-M track pid=$! log=$HOME/imb-lme-m-recover-track.log"
}

# --- BEAM ---
launch_beam() {
  local lock="$HOME/imb-beam/.recover.lock"
  mkdir -p "$HOME/imb-beam"
  if [[ -f "$lock" ]] && kill -0 "$(cat "$lock")" 2>/dev/null; then
    log "BEAM recover already running pid=$(cat "$lock")"
    return 0
  fi
  nohup bash -c '
    set -euo pipefail
    REPO_ROOT="'"$REPO_ROOT"'"; REKAL="'"$REKAL"'"
    echo "[$(date -Iseconds)] BEAM track start"
    bash "$REPO_ROOT/scripts/industry-bench/datasets/get_beam.sh" 128k >> "$HOME/imb-beam-download.log" 2>&1 || true
    python3 "$REPO_ROOT/scripts/industry-bench/datasets/normalize_beam.py" --tier 128k >> "$HOME/imb-beam-download.log" 2>&1 || true
    JSONL="$REPO_ROOT/scripts/industry-bench/datasets/data/beam-128k-conversations.jsonl"
    if [[ -f "$JSONL" ]]; then
      PYTHONUNBUFFERED=1 python3 "$REPO_ROOT/scripts/industry-bench/sh_gen/gen.py" \
        --input "$JSONL" --limit 5 --out "$HOME/imb-beam" --rekal "$REKAL" --verify --index --fast 10 \
        > "$HOME/imb-beam/ingest-limit5.log" 2>&1 || true
    fi
    echo "[$(date -Iseconds)] BEAM track done"
    rm -f "'"$lock"'"
  ' >> "$HOME/imb-beam-recover-track.log" 2>&1 &
  echo $! > "$lock"
  log "BEAM track pid=$! log=$HOME/imb-beam-recover-track.log"
}

# --- Watchdog: every 3 min, relaunch dead tracks if work remains ---
launch_watchdog() {
  local lock="$HOME/imb-watchdog.lock"
  if [[ -f "$lock" ]] && kill -0 "$(cat "$lock")" 2>/dev/null; then
    log "watchdog already running pid=$(cat "$lock")"
    return 0
  fi
  nohup bash -c '
    REPO_ROOT="'"$REPO_ROOT"'"
    while true; do
      sleep 180
      echo "[$(date -Iseconds)] watchdog tick disk=$(df -h /Users/frank | awk "NR==2{print \$4}")" >> "'"$MASTER"'"
      # LME-S: if any of 400-475 incomplete and no lock alive → relaunch
      need=0
      for off in 400 425 450 475; do
        out=$(printf "$HOME/imb-lme-s/shard-%04d" "$off")
        ndb=$(find "$out" -name data.db 2>/dev/null | wc -l | tr -d " ")
        if ! { [[ -f "$out/ingest.log" ]] && grep -qE "^(done:)|VERIFICATION FAILED" "$out/ingest.log" 2>/dev/null && [[ "${ndb:-0}" -ge 20 ]]; }; then
          need=1
        fi
      done
      lock="$HOME/imb-lme-s/.recover.lock"
      if [[ "$need" -eq 1 ]]; then
        if ! { [[ -f "$lock" ]] && kill -0 "$(cat "$lock")" 2>/dev/null; }; then
          echo "[$(date -Iseconds)] watchdog relaunch LME-S" >> "'"$MASTER"'"
          REKAL="'"$REKAL"'" bash "$REPO_ROOT/scripts/industry-bench/run_recover_detached.sh" --lme-s-only
        fi
      fi
      # MSC
      lock="$HOME/imb-msc/.recover.lock"
      incomplete_msc=0
      for s in "$HOME"/imb-msc/shard-*; do
        [[ -d "$s" ]] || continue
        if ! { [[ -f "$s/ingest.log" ]] && grep -qE "^(done:)|VERIFICATION FAILED" "$s/ingest.log" 2>/dev/null; }; then
          incomplete_msc=1; break
        fi
      done
      # also if fewer than 30 shards expected (1501/50)
      nshards=$(find "$HOME/imb-msc" -maxdepth 1 -type d -name "shard-*" 2>/dev/null | wc -l | tr -d " ")
      if [[ "${nshards:-0}" -lt 30 ]]; then incomplete_msc=1; fi
      if [[ "$incomplete_msc" -eq 1 ]]; then
        if ! { [[ -f "$lock" ]] && kill -0 "$(cat "$lock")" 2>/dev/null; }; then
          echo "[$(date -Iseconds)] watchdog relaunch MSC" >> "'"$MASTER"'"
          REKAL="'"$REKAL"'" bash "$REPO_ROOT/scripts/industry-bench/run_recover_detached.sh" --msc-only
        fi
      fi
    done
  ' >> "$HOME/imb-watchdog.log" 2>&1 &
  echo $! > "$lock"
  log "watchdog pid=$!"
}

ONLY="${1:-}"
case "$ONLY" in
  --lme-s-only) launch_lme_s; exit 0 ;;
  --msc-only) launch_msc; exit 0 ;;
  --lme-m-only) launch_lme_m; exit 0 ;;
  --beam-only) launch_beam; exit 0 ;;
esac

log "=== DETACHED RECOVER $STAMP ==="
log "LoCoMo torn (aggregate kept in runs/full/locomo/). Oracle torn."
launch_lme_s
launch_msc
launch_beam
# LME-M after short delay via background sleeper so LME-S gets CPU/disk first
nohup bash -c "sleep 90; REKAL='$REKAL' bash '$REPO_ROOT/scripts/industry-bench/run_recover_detached.sh' --lme-m-only" >> "$MASTER" 2>&1 &
launch_watchdog
log "all tracks launched — monitor: tail -f $MASTER $HOME/imb-*-recover-track.log"
