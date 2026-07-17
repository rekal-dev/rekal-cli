#!/usr/bin/env bash
# Parallel industry-bench smokes — one job per (conversation × route).
# Safe across conversations: each has its own repo / DuckDB (no cross-repo lock).
#
# Usage:
#   export REKAL=./rekal WORKERS=8
#   ./run_smokes_parallel.sh locomo   # all convs under ~/imb-locomo
#   ./run_smokes_parallel.sh oracle   # 3-conv oracle stream
#   ./run_smokes_parallel.sh lme-s    # 3-conv smoke set under /tmp/imb-lme-s-smoke
#
# Env:
#   WORKERS       parallel jobs (default: ncpu, cap 8)
#   DATE_TAG      manifest suffix (default: 20260717)
#   ONLY_MISSING  1 = skip if manifest.json exists (default: 1)
set -euo pipefail

STREAM="${1:?stream required: toy | lme-s | oracle | locomo}"
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
REKAL="${REKAL:-$REPO_ROOT/rekal}"
SHIM="$REPO_ROOT/scripts/industry-bench/shim/shim.py"
CAL="$REPO_ROOT/scripts/industry-bench/calibration/chat-provisional.json"
RUNS="$REPO_ROOT/scripts/industry-bench/runs/smoke"
DATE_TAG="${DATE_TAG:-20260717}"
ONLY_MISSING="${ONLY_MISSING:-1}"

if [[ ! -x "$REKAL" ]]; then
  echo "rekal not found at $REKAL" >&2
  exit 1
fi

NCPU="$(sysctl -n hw.ncpu 2>/dev/null || nproc 2>/dev/null || echo 4)"
WORKERS="${WORKERS:-$NCPU}"
if [[ "$WORKERS" -gt 8 ]]; then
  WORKERS=8
fi

case "$STREAM" in
  toy)
    WORKDIR="${TOY_WORKDIR:-/tmp/imb-toy-local}"
    INPUT="$REPO_ROOT/scripts/industry-bench/datasets/toy/conversations.jsonl"
    CONVS=(toy-001)
    ;;
  lme-s)
    WORKDIR="${LME_WORKDIR:-/tmp/imb-lme-s-smoke}"
    INPUT="${LME_INPUT:-/tmp/imb-input-longmemeval-s.jsonl}"
    [[ -f "$INPUT" ]] || INPUT="$REPO_ROOT/scripts/industry-bench/datasets/data/longmemeval-s-conversations.jsonl"
    CONVS=(e47becba 118b2229 51a45a95)
    ;;
  oracle)
    WORKDIR="${ORACLE_WORKDIR:-$HOME/imb-lme-oracle}"
    INPUT="$REPO_ROOT/scripts/industry-bench/datasets/data/longmemeval-oracle-conversations.jsonl"
    CONVS=()
    while IFS= read -r line; do
      [[ -n "$line" ]] && CONVS+=("$line")
    done < <(python3 -c "
import json
from pathlib import Path
for line in Path('$INPUT').read_text().splitlines():
    if line.strip():
        print(json.loads(line)['conversation_id'])
")
    if [[ ${#CONVS[@]} -eq 0 ]]; then
      for d in "$WORKDIR"/*/; do
        [[ -d "${d}repo/.rekal" ]] && CONVS+=("$(basename "$d")")
      done
    fi
    ;;
  locomo)
    WORKDIR="${LOCO_WORKDIR:-$HOME/imb-locomo}"
    INPUT="$REPO_ROOT/scripts/industry-bench/datasets/data/locomo-conversations.jsonl"
    CONVS=()
    for d in "$WORKDIR"/*/; do
      [[ -d "${d}repo/.rekal" ]] && CONVS+=("$(basename "$d")")
    done
    ;;
  *)
    echo "unknown stream: $STREAM" >&2
    exit 1
    ;;
esac

if [[ ! -d "$WORKDIR" ]]; then
  echo "workdir missing: $WORKDIR" >&2
  exit 1
fi

run_one() {
  local route=$1 conv=$2
  local out="$RUNS/${STREAM}-${conv}-${route}-${DATE_TAG}"
  if [[ "$ONLY_MISSING" == "1" && -f "$out/manifest.json" ]]; then
    echo "skip $conv $route (exists)"
    return 0
  fi
  echo "start $conv $route"
  if [[ "$route" == skill ]]; then
    python3 "$SHIM" smoke \
      --workdir "$WORKDIR" \
      --conversation-id "$conv" \
      --input "$INPUT" \
      --out "$out" \
      --rekal "$REKAL" \
      --route skill \
      --calibration "$CAL"
  else
    python3 "$SHIM" smoke \
      --workdir "$WORKDIR" \
      --conversation-id "$conv" \
      --input "$INPUT" \
      --out "$out" \
      --rekal "$REKAL" \
      --route stock
  fi
  echo "done $conv $route"
}
export -f run_one
export WORKDIR INPUT RUNS DATE_TAG REKAL SHIM CAL ONLY_MISSING STREAM

JOBS=()
for conv in "${CONVS[@]}"; do
  [[ -d "$WORKDIR/$conv/repo/.rekal" ]] || continue
  JOBS+=("stock $conv")
  JOBS+=("skill $conv")
done

echo "stream=$STREAM workers=$WORKERS jobs=${#JOBS[@]} workdir=$WORKDIR"

if command -v parallel >/dev/null 2>&1; then
  printf '%s\n' "${JOBS[@]}" | parallel -j "$WORKERS" --colsep ' ' run_one {1} {2}
else
  JOBFILE="$(mktemp)"
  printf '%s\n' "${JOBS[@]}" > "$JOBFILE"
  xargs -P "$WORKERS" -n 2 bash -c 'run_one "$1" "$2"' _ < "$JOBFILE"
  rm -f "$JOBFILE"
fi

echo "finished stream=$STREAM"
