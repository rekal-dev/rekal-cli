#!/usr/bin/env bash
# WS-A: fetch the dial481/locomo-audit consolidated errors list and emit
# locomo-known-bad.jsonl (score-corrupting errors only; WRONG_CITATION
# excluded). Required by 04-procedures §2 step 4 for LoCoMo honesty columns.
#
# Source: https://github.com/dial481/locomo-audit (Penfield Labs audit;
# also summarized at DEV/Substack "We audited LoCoMo").
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
DATA="$DIR/data"
mkdir -p "$DATA"
URL="https://raw.githubusercontent.com/dial481/locomo-audit/main/errors.json"
RAW="$DATA/locomo-errors.json"
OUT="$DIR/locomo-known-bad.jsonl"

curl -fsSL "$URL" -o "$RAW"
sha256sum "$RAW"

python3 - "$RAW" "$DIR/data/locomo10.json" "$OUT" <<'EOF'
import json, re, sys
from collections import Counter
errors_path, locomo_path, out_path = sys.argv[1:4]
errors = json.load(open(errors_path))
sample_ids = [c["sample_id"] for c in json.load(open(locomo_path))]
n = 0
types = Counter()
with open(out_path, "w") as f:
    for e in errors:
        if e["error_type"] == "WRONG_CITATION":
            continue
        m = re.fullmatch(r"locomo_(\d+)_qa(\d+)", e["question_id"])
        if not m:
            raise SystemExit(f"unparseable audit id {e['question_id']!r}")
        conv_i, qa_i = int(m.group(1)), int(m.group(2))
        row = {
            "audit_question_id": e["question_id"],
            "conversation_id": sample_ids[conv_i],
            "question_id": f"q{qa_i}",
            "error_type": e["error_type"],
            "category": e["category"],
            "golden_answer": e.get("golden_answer"),
            "correct_answer": e.get("correct_answer"),
            "reasoning": e.get("reasoning"),
        }
        types[row["error_type"]] += 1
        f.write(json.dumps(row, ensure_ascii=False) + "\n")
        n += 1
print(f"wrote {out_path}: score_corrupting={n} types={dict(types)}")
assert n == 99, f"expected 99 score-corrupting errors, got {n}"
EOF

# Keep a copy under data/ too (gitignored) for local scoring runs.
cp -f "$OUT" "$DATA/locomo-known-bad.jsonl"
echo "committed path: $OUT"
