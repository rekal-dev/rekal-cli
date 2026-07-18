#!/usr/bin/env bash
# WS-A: fetch LoCoMo (snap-research/locomo, locomo10.json) from GitHub raw.
# Data stays out of git (see .gitignore); only this script is committed.
#
# License note: the LoCoMo repo (github.com/snap-research/locomo) publishes
# the dataset for research use; we download at run time and do not
# redistribute. Verify the repo's LICENSE before any public artifact
# includes raw conversation text.
#
# Known flaws (docs/research/industry-bench/00-landscape.md §6): ~6.4% of
# the answer key is wrong per the Penfield audit; scores must be reported
# with the caveat block from 04-procedures §9.
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)/data"
mkdir -p "$DIR"
URL="https://raw.githubusercontent.com/snap-research/locomo/main/data/locomo10.json"
OUT="$DIR/locomo10.json"

curl -fsSL "$URL" -o "$OUT"
BYTES=$(wc -c < "$OUT")
echo "downloaded $OUT ($BYTES bytes)"
sha256sum "$OUT"

python3 - "$OUT" <<'EOF'
import json, sys
d = json.load(open(sys.argv[1]))
n_conv = len(d)
n_sess = sum(1 for c in d for k, v in c["conversation"].items()
             if k.startswith("session_") and isinstance(v, list))
n_turn = sum(len(v) for c in d for k, v in c["conversation"].items()
             if k.startswith("session_") and isinstance(v, list))
n_q = sum(len(c["qa"]) for c in d)
print(f"conversations={n_conv} sessions={n_sess} turns={n_turn} questions={n_q}")
assert n_conv == 10, "locomo10.json should hold 10 conversations"
EOF
