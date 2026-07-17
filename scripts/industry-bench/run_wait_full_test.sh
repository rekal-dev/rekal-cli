#!/usr/bin/env bash
# Wait for LME-S ingest to finish, then launch full TEST eval.
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
IMB_ROOT="${IMB_ROOT:-$HOME/imb-lme-s}"
REKAL="${REKAL:-$REPO_ROOT/rekal}"
WORKERS="${WORKERS:-8}"
SHARD_SIZE=25
TOTAL=500
EXPECTED=$(( (TOTAL + SHARD_SIZE - 1) / SHARD_SIZE ))
LOG="$IMB_ROOT/wait-full-test.log"

log() { echo "[$(date -Iseconds)] $*" | tee -a "$LOG"; }

finished_shards() {
  local done=0 i=0
  while [[ $i -lt $TOTAL ]]; do
    local shard
    shard="$(printf '%s/shard-%04d' "$IMB_ROOT" "$i")"
    if [[ -f "$shard/ingest.log" ]] && grep -qE '^(done:)' "$shard/ingest.log" 2>/dev/null; then
      done=$((done + 1))
    elif [[ -f "$shard/ingest.log" ]] && grep -qi 'VERIFICATION FAILED' "$shard/ingest.log"; then
      done=$((done + 1))
    fi
    i=$((i + SHARD_SIZE))
  done
  echo "$done"
}

log "waiting for ingest ($EXPECTED shards)"
while true; do
  done="$(finished_shards)"
  running="$(pgrep -f 'sh_gen/gen.py.*imb-lme-s' 2>/dev/null | wc -l | tr -d ' ' || true)"
  [[ -z "$running" ]] && running=0
  log "shards=$done/$EXPECTED procs=$running"
  if [[ "$done" -ge "$EXPECTED" ]]; then
    break
  fi
  sleep 90
done

log "ingest complete — launching full test"
export REKAL WORKERS IMB_ROOT
DATE_TAG="fulltest-20260717" bash "$REPO_ROOT/scripts/industry-bench/run_full_test.sh" \
  2>&1 | tee -a "$LOG"
log "done"
