#!/usr/bin/env bash
# WS-A: fetch BEAM tier from Hugging Face (Mohammadta/BEAM).
# Tiers: 128k | 500k | 1m  (10M is Mohammadta/BEAM-10M — separate, huge)
# Requires: curl, python3. Prefer `datasets` if installed; else try parquet/json via HF resolve.
set -euo pipefail
TIER="${1:-128k}"
DIR="$(cd "$(dirname "$0")" && pwd)/data"
mkdir -p "$DIR"
OUT="$DIR/beam-${TIER}-raw"

case "$TIER" in
  128k|500k|1m|10m) ;;
  *) echo "usage: $0 [128k|500k|1m|10m]" >&2; exit 2 ;;
esac

echo "fetching BEAM tier=$TIER (needs huggingface datasets or HF API)"
python3 - "$TIER" "$OUT" <<'PY'
import json, sys
from pathlib import Path
tier, out = sys.argv[1], Path(sys.argv[2])
out.mkdir(parents=True, exist_ok=True)
try:
    from datasets import load_dataset
except ImportError:
    print("pip install datasets  # required for BEAM download", file=sys.stderr)
    sys.exit(2)

if tier == "10m":
    ds = load_dataset("Mohammadta/BEAM-10M", split="10M")
elif tier == "128k":
    # HF config labels this tier 100K (context budget ~128k tokens).
    ds = load_dataset("Mohammadta/BEAM", split="100K")
elif tier == "500k":
    ds = load_dataset("Mohammadta/BEAM", split="500K")
else:
    ds = load_dataset("Mohammadta/BEAM", split="1M")

path = out / "conversations.jsonl"
n = 0
with open(path, "w") as f:
    for row in ds:
        f.write(json.dumps(dict(row), ensure_ascii=False, default=str) + "\n")
        n += 1
print(f"wrote {path} rows={n}")
# peek keys
print("sample_keys", list(dict(ds[0]).keys())[:20])
PY

cat > "$(cd "$(dirname "$0")" && pwd)/LICENSE-NOTE-beam.md" <<'NOTE'
# BEAM — local-use license note

Source: https://huggingface.co/datasets/Mohammadta/BEAM
Paper: Beyond a Million Tokens (long-term memory benchmark).
Downloaded at run time via get_beam.sh; raw data not committed.
NOTE
echo "done BEAM $TIER"
