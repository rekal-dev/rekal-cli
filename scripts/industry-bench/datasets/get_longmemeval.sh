#!/usr/bin/env bash
# WS-A: fetch LongMemEval cleaned releases from Hugging Face
# (xiaowu0162/longmemeval-cleaned). Data stays out of git (see .gitignore);
# only this script is committed.
#
# Canonical source (official README, 2025-09 cleaned release):
#   https://huggingface.co/datasets/xiaowu0162/longmemeval-cleaned
#   https://github.com/xiaowu0162/LongMemEval
#
# License: MIT on the HF dataset card. We download at run time and do not
# redistribute raw conversation text in git (datasets/data/ is gitignored).
#
# Variants:
#   s       — LongMemEval-S (~40 sessions / instance, ~115k tokens)
#   m       — LongMemEval-M (~500 sessions / instance)
#   oracle  — evidence sessions only
#   all     — s + m + oracle (default: s)
set -euo pipefail

VARIANT="${1:-s}"
DIR="$(cd "$(dirname "$0")" && pwd)/data"
mkdir -p "$DIR"
BASE="https://huggingface.co/datasets/xiaowu0162/longmemeval-cleaned/resolve/main"

fetch_one() {
  local name="$1"
  local url="$BASE/$name"
  local out="$DIR/$name"
  echo "fetching $name ..."
  curl -fsSL "$url" -o "$out"
  local bytes
  bytes=$(wc -c < "$out")
  echo "downloaded $out ($bytes bytes)"
  sha256sum "$out"
  python3 - "$out" <<'EOF'
import json, sys
path = sys.argv[1]
data = json.load(open(path))
assert isinstance(data, list), f"{path}: expected JSON array"
n = len(data)
assert n == 500, f"{path}: expected 500 instances, got {n}"
n_sess = sum(len(x["haystack_sessions"]) for x in data)
n_turn = sum(len(s) for x in data for s in x["haystack_sessions"])
n_abs = sum(1 for x in data if x["question_id"].endswith("_abs"))
print(f"instances={n} sessions={n_sess} turns={n_turn} abstention={n_abs}")
EOF
}

case "$VARIANT" in
  s)
    fetch_one longmemeval_s_cleaned.json
    ;;
  m)
    fetch_one longmemeval_m_cleaned.json
    ;;
  oracle)
    fetch_one longmemeval_oracle.json
    ;;
  all)
    fetch_one longmemeval_s_cleaned.json
    fetch_one longmemeval_m_cleaned.json
    fetch_one longmemeval_oracle.json
    ;;
  *)
    echo "usage: $0 [s|m|oracle|all]" >&2
    exit 2
    ;;
esac

# License note is committed next to this script (not under gitignored data/).
NOTE_PATH="$(cd "$(dirname "$0")" && pwd)/LICENSE-NOTE-longmemeval.md"
if [ ! -f "$NOTE_PATH" ]; then
  cat > "$NOTE_PATH" <<'NOTE'
# LongMemEval — local-use license note

Source: https://huggingface.co/datasets/xiaowu0162/longmemeval-cleaned
Upstream: https://github.com/xiaowu0162/LongMemEval (ICLR 2025)
License (HF card): MIT

We download at run time via `get_longmemeval.sh` and do **not** commit raw
JSON or normalized conversation text. Aggregates / manifests under
`scripts/industry-bench/runs/` may cite counts and question ids only.
NOTE
fi
echo "license note: $NOTE_PATH"
