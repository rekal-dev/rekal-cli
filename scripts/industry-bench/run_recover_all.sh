#!/usr/bin/env bash
# Resume/recover stalled full-tier ingest after disk-full.
# Priority: LME-S finish → full TEST; MSC eval; LME-M shards; BEAM download.
#
#   export REKAL=./rekal WORKERS=4
#   nohup bash scripts/industry-bench/run_recover_all.sh >> ~/imb-recover.log 2>&1 &
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
REKAL="${REKAL:-$REPO_ROOT/rekal}"
export REKAL
WORKERS="${WORKERS:-4}"
LOG="${LOG:-$HOME/imb-recover.log}"
log(){ echo "[$(date -Iseconds)] $*" | tee -a "$LOG"; }

shard_terminal() {
  local logf=$1
  [[ -f "$logf" ]] || return 1
  grep -qE '^(done:)' "$logf" 2>/dev/null && return 0
  grep -qi 'VERIFICATION FAILED' "$logf" 2>/dev/null && return 0
  return 1
}

# --- 1. LME-S: wipe incomplete 400-475, re-ingest, then full test ---
recover_lme_s() {
  log "[1] LME-S recover shards 400-475"
  local jsonl="$REPO_ROOT/scripts/industry-bench/datasets/data/longmemeval-s-conversations.jsonl"
  for off in 400 425 450 475; do
    out=$(printf "$HOME/imb-lme-s/shard-%04d" "$off")
    ndb=$(find "$out" -name data.db 2>/dev/null | wc -l | tr -d ' ')
    if shard_terminal "$out/ingest.log" && [[ "${ndb:-0}" -ge 20 ]]; then
      log "[1] shard $off OK (dbs=$ndb) — skip"
      continue
    fi
    log "[1] re-ingest shard $off (was dbs=$ndb)"
    rm -rf "$out"
    mkdir -p "$out"
    (
      PYTHONUNBUFFERED=1 python3 "$REPO_ROOT/scripts/industry-bench/sh_gen/gen.py" \
        --input "$jsonl" --offset "$off" --limit 25 \
        --out "$out" --rekal "$REKAL" --verify \
        > "$out/ingest.log" 2>&1 || true
      log "[1] shard $off finished (exit)"
    ) &
  done
  wait
  log "[1] LME-S shards done — flatten + index + full TEST"
  DATE_TAG=fulltest-20260717 bash "$REPO_ROOT/scripts/industry-bench/run_full_test.sh" \
    >> "$HOME/imb-lme-s-full-test.log" 2>&1 || log "[1] full test exited non-zero"
  log "[1] LME-S DONE"
}

# --- 2. MSC eval subset: redo any shard without terminal marker ---
recover_msc() {
  log "[2] MSC eval recover"
  local subset="$REPO_ROOT/scripts/industry-bench/datasets/data/msc-eval-conversations.jsonl"
  local full="$REPO_ROOT/scripts/industry-bench/datasets/data/msc-conversations.jsonl"
  if [[ ! -f "$subset" ]]; then
    log "[2] building msc-eval subset"
    python3 - "$full" "$subset" <<'PY'
import json, sys
from pathlib import Path
src, out = Path(sys.argv[1]), Path(sys.argv[2])
n=0
with open(src) as f, open(out,"w") as o:
    for line in f:
        c=json.loads(line)
        cid=c.get("conversation_id","")
        if c.get("extra",{}).get("split") in ("test","valid") or cid.startswith(("msc-test-","msc-valid-")):
            o.write(line); n+=1
print(f"msc eval subset conversations={n}")
PY
  fi
  mkdir -p "$HOME/imb-msc"
  TOTAL=$(wc -l < "$subset" | tr -d ' ')
  off=0
  while [[ $off -lt $TOTAL ]]; do
    out=$(printf "$HOME/imb-msc/shard-%04d" "$off")
    if shard_terminal "$out/ingest.log"; then
      log "[2] MSC shard $off already terminal — skip"
    else
      log "[2] MSC re-ingest shard $off"
      rm -rf "$out"
      mkdir -p "$out"
      (
        PYTHONUNBUFFERED=1 python3 "$REPO_ROOT/scripts/industry-bench/sh_gen/gen.py" \
          --input "$subset" --offset "$off" --limit 50 \
          --out "$out" --rekal "$REKAL" --verify --fast 3 \
          > "$out/ingest.log" 2>&1 || true
        log "[2] MSC shard $off finished"
      ) &
      while [[ "$(jobs -rp | wc -l | tr -d ' ')" -ge "$WORKERS" ]]; do sleep 15; done
    fi
    off=$((off + 50))
  done
  wait
  log "[2] MSC eval DONE"
}

# --- 3. LME-M: resume all missing shards (huge — throttled) ---
recover_lme_m() {
  log "[3] LME-M recover"
  local jsonl="$REPO_ROOT/scripts/industry-bench/datasets/data/longmemeval-m-conversations.jsonl"
  if [[ ! -f "$jsonl" ]]; then
    log "[3] missing M jsonl — normalize"
    python3 "$REPO_ROOT/scripts/industry-bench/datasets/normalize_longmemeval.py" --variant m \
      >> "$HOME/imb-lme-m-normalize.log" 2>&1 || { log "[3] normalize failed"; return 1; }
  fi
  mkdir -p "$HOME/imb-lme-m"
  local m_workers="$WORKERS"
  if [[ "$m_workers" -gt 3 ]]; then m_workers=3; fi
  for off in $(seq 0 25 475); do
    out=$(printf "$HOME/imb-lme-m/shard-%04d" "$off")
    ndb=$(find "$out" -name data.db 2>/dev/null | wc -l | tr -d ' ')
    if shard_terminal "$out/ingest.log" && [[ "${ndb:-0}" -ge 20 ]]; then
      log "[3] LME-M shard $off OK — skip"
      continue
    fi
    log "[3] LME-M re-ingest shard $off (was dbs=${ndb:-0})"
    rm -rf "$out"
    mkdir -p "$out"
    (
      PYTHONUNBUFFERED=1 python3 "$REPO_ROOT/scripts/industry-bench/sh_gen/gen.py" \
        --input "$jsonl" --offset "$off" --limit 25 \
        --out "$out" --rekal "$REKAL" --verify --fast 5 \
        > "$out/ingest.log" 2>&1 || true
      log "[3] LME-M shard $off finished"
    ) &
    while [[ "$(jobs -rp | wc -l | tr -d ' ')" -ge "$m_workers" ]]; do sleep 30; done
  done
  wait
  log "[3] LME-M ALL SHARDS DONE"
}

# --- 4. BEAM download + normalize + smoke ingest ---
recover_beam() {
  log "[4] BEAM download/normalize/smoke"
  bash "$REPO_ROOT/scripts/industry-bench/datasets/get_beam.sh" 128k \
    >> "$HOME/imb-beam-download.log" 2>&1 || log "[4] get_beam failed"
  python3 "$REPO_ROOT/scripts/industry-bench/datasets/normalize_beam.py" --tier 128k \
    >> "$HOME/imb-beam-download.log" 2>&1 || log "[4] normalize_beam failed"
  mkdir -p "$HOME/imb-beam"
  local jsonl="$REPO_ROOT/scripts/industry-bench/datasets/data/beam-128k-conversations.jsonl"
  if [[ -f "$jsonl" ]]; then
    PYTHONUNBUFFERED=1 python3 "$REPO_ROOT/scripts/industry-bench/sh_gen/gen.py" \
      --input "$jsonl" --limit 5 --out "$HOME/imb-beam" --rekal "$REKAL" --verify --index --fast 10 \
      > "$HOME/imb-beam/ingest-limit5.log" 2>&1 || true
    log "[4] BEAM smoke ingest done"
  else
    log "[4] BEAM jsonl missing — skip ingest"
  fi
}

log "=== RECOVER START workers=$WORKERS disk=$(df -h /Users/frank | awk 'NR==2{print $4}') free ==="

# LoCoMo already complete — leave alone
log "[0] LoCoMo: leave existing full retrieve aggregate"

# LME-S first (highest priority), others in parallel with lower disk pressure
recover_lme_s &
PID1=$!
recover_beam &
PID4=$!
# MSC after a short delay so LME-S gets disk headroom first
(
  sleep 60
  recover_msc
) &
PID2=$!
# LME-M last / parallel but throttled to 3
(
  sleep 120
  recover_lme_m
) &
PID3=$!

wait $PID1 || true
wait $PID4 || true
wait $PID2 || true
wait $PID3 || true
log "=== RECOVER DRIVER EXITED ==="
log "disk=$(df -h /Users/frank | awk 'NR==2{print $4}') free"
