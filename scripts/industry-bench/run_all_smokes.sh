#!/usr/bin/env bash
# Rerun industry-bench smokes locally (stock + skill+provisional).
# Requires pre-ingested workdirs; see HANDOVER.md.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
REKAL="${REKAL:-$ROOT/rekal}"
SHIM="$ROOT/scripts/industry-bench/shim/shim.py"
CAL="$ROOT/scripts/industry-bench/calibration/chat-provisional.json"
RUNS="$ROOT/scripts/industry-bench/runs/smoke"
DATE_TAG="${DATE_TAG:-20260717}"

SKIP_LOCOMO="${SKIP_LOCOMO:-0}"

if [[ ! -x "$REKAL" ]]; then
  echo "rekal binary not found at $REKAL (build with: go build -o ./rekal ./cmd/rekal)" >&2
  exit 1
fi

run_smoke() {
  local route=$1 workdir=$2 conv=$3 input=$4 out=$5
  shift 5
  echo "==> smoke $conv ($route) -> $out"
  if [[ "$route" == skill ]]; then
    python3 "$SHIM" smoke \
      --workdir "$workdir" \
      --conversation-id "$conv" \
      --input "$input" \
      --out "$out" \
      --rekal "$REKAL" \
      --route skill \
      --calibration "$CAL" \
      "$@"
  else
    python3 "$SHIM" smoke \
      --workdir "$workdir" \
      --conversation-id "$conv" \
      --input "$input" \
      --out "$out" \
      --rekal "$REKAL" \
      --route stock \
      "$@"
  fi
}

mkdir -p "$RUNS"

# Toy
TOY_WD="${TOY_WORKDIR:-/tmp/imb-toy-local}"
TOY_IN="$ROOT/scripts/industry-bench/datasets/toy/conversations.jsonl"
run_smoke stock "$TOY_WD" toy-001 "$TOY_IN" "$RUNS/toy-stock-$DATE_TAG"
run_smoke skill "$TOY_WD" toy-001 "$TOY_IN" "$RUNS/toy-skill-$DATE_TAG"

# LongMemEval-S (3-conversation smoke set)
LME_WD="${LME_WORKDIR:-/tmp/imb-lme-s-smoke}"
LME_IN="${LME_INPUT:-/tmp/imb-input-longmemeval-s.jsonl}"
if [[ ! -f "$LME_IN" ]]; then
  LME_IN="$ROOT/scripts/industry-bench/datasets/data/longmemeval-s-conversations.jsonl"
fi
for conv in e47becba 118b2229 51a45a95; do
  run_smoke stock "$LME_WD" "$conv" "$LME_IN" "$RUNS/lme-s-${conv}-stock-$DATE_TAG"
  run_smoke skill "$LME_WD" "$conv" "$LME_IN" "$RUNS/lme-s-${conv}-skill-$DATE_TAG"
done

# LoCoMo (all ingested conversations under workdir)
if [[ "$SKIP_LOCOMO" != "1" ]]; then
  LOCO_WD="${LOCO_WORKDIR:-$HOME/imb-locomo}"
  LOCO_IN="$ROOT/scripts/industry-bench/datasets/data/locomo-conversations.jsonl"
  if [[ ! -d "$LOCO_WD" ]]; then
    echo "LoCoMo workdir missing: $LOCO_WD (run sh_gen ingest first)" >&2
    exit 1
  fi
  for conv_dir in "$LOCO_WD"/*/; do
    conv="$(basename "$conv_dir")"
    [[ -d "$conv_dir/repo/.rekal" ]] || continue
    run_smoke stock "$LOCO_WD" "$conv" "$LOCO_IN" "$RUNS/locomo-${conv}-stock-$DATE_TAG"
    run_smoke skill "$LOCO_WD" "$conv" "$LOCO_IN" "$RUNS/locomo-${conv}-skill-$DATE_TAG"
  done
fi

echo "done: manifests under $RUNS"
