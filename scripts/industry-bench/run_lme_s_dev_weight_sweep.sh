#!/usr/bin/env bash
# Weight sweep on the LongMemEval-S 100-conv DEV split (NEVER test).
# Rigorous calibration: tune on dev, freeze, then re-run test separately.
# Reuses the frozen gate from calibration/longmemeval-s.json; varies weights.
#
#   REKAL=./rekal WORKERS=4 bash scripts/industry-bench/run_lme_s_dev_weight_sweep.sh
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
REKAL="${REKAL:-$REPO_ROOT/rekal}"
FLAT="${FLAT:-$HOME/imb-lme-s/flat}"
SHIM="$REPO_ROOT/scripts/industry-bench/shim/shim.py"
DEV_IDS="$REPO_ROOT/scripts/industry-bench/calibration/split-longmemeval-s-dev-ids.txt"
DEV_IN="$REPO_ROOT/scripts/industry-bench/calibration/split-longmemeval-s-dev-conversations.jsonl"
CAL_DIR="$REPO_ROOT/scripts/industry-bench/calibration/dev-profiles"
RUNS="$REPO_ROOT/scripts/industry-bench/runs/sweep/lme-s-dev-$(date +%Y%m%d-%H%M)"
LOG="$HOME/imb-lme-s-dev-sweep.log"
WORKERS="${WORKERS:-4}"

log(){ echo "[$(date -Iseconds)] $*" | tee -a "$LOG"; }
mkdir -p "$CAL_DIR" "$RUNS"

# Profiles: reuse frozen gate, vary the ranking-layer mix.
python3 - "$CAL_DIR" "$REPO_ROOT/scripts/industry-bench/calibration/longmemeval-s.json" <<'PY'
import json, sys
from pathlib import Path
cal_dir = Path(sys.argv[1])
gate = json.loads(Path(sys.argv[2]).read_text())["gate"]
base = {"steering_boost": 1.0, "summary_boost": 1.0, "subagent_downweight": 1.0}
profiles = {
    "dev-interim":       {"bm25": 0.30, "lsa": 0.20, "semantic": 0.50, "facet_boost": 0.10},
    "dev-bm25-push":     {"bm25": 0.55, "lsa": 0.10, "semantic": 0.35, "facet_boost": 0.05},
    "dev-bm25-mid":      {"bm25": 0.45, "lsa": 0.15, "semantic": 0.40, "facet_boost": 0.10},
    "dev-semantic-push": {"bm25": 0.20, "lsa": 0.15, "semantic": 0.65, "facet_boost": 0.05},
    "dev-balanced":      {"bm25": 0.40, "lsa": 0.20, "semantic": 0.40, "facet_boost": 0.10},
}
for name, w in profiles.items():
    body = {"gate": gate, "weights": {**base, **w}, "notes": f"LME-S dev sweep {name}"}
    (cal_dir / f"{name}.json").write_text(json.dumps(body, indent=2) + "\n")
    print(cal_dir / f"{name}.json")
PY

log "=== LME-S DEV weight sweep START workers=$WORKERS → $RUNS ==="

run_one() {
  local prof=$1 conv=$2
  local out="$RUNS/${conv}-${prof}"
  [[ -f "$out/manifest.json" ]] && { echo "skip $conv $prof"; return 0; }
  [[ -d "$FLAT/$conv/repo/.rekal" ]] || { echo "missing repo $conv"; return 0; }
  python3 "$SHIM" smoke --workdir "$FLAT" --conversation-id "$conv" --input "$DEV_IN" \
    --out "$out" --rekal "$REKAL" --route skill \
    --calibration "$CAL_DIR/${prof}.json" >> "$LOG" 2>&1 || echo "fail $conv $prof"
  echo "done $conv $prof"
}
export -f run_one
export RUNS FLAT SHIM DEV_IN REKAL CAL_DIR LOG

JOBS=()
while IFS= read -r conv; do
  [[ -n "$conv" ]] || continue
  for prof in dev-interim dev-bm25-push dev-bm25-mid dev-semantic-push dev-balanced; do
    JOBS+=("$prof|$conv")
  done
done < "$DEV_IDS"

log "jobs=${#JOBS[@]}"
printf '%s\n' "${JOBS[@]}" | xargs -P "$WORKERS" -I{} bash -c '
  IFS="|" read -r prof conv <<<"{}"
  run_one "$prof" "$conv"
'

# Aggregate per profile.
python3 - "$RUNS" <<'PY'
import json, sys
from pathlib import Path
from collections import defaultdict
root = Path(sys.argv[1])
agg = defaultdict(lambda: {"runs": 0, "q": 0, "h5": 0.0, "h10": 0.0})
for m in root.glob("*/manifest.json"):
    prof = m.parent.name.split("-", 1)[1] if "-" in m.parent.name else m.parent.name
    # name is <conv>-dev-<profile>; recover profile by matching known prefixes
    name = m.parent.name
    for p in ("dev-interim", "dev-bm25-push", "dev-bm25-mid", "dev-semantic-push", "dev-balanced"):
        if name.endswith(p):
            prof = p
            break
    d = json.loads(m.read_text())
    n = int(d.get("n_questions") or 0)
    a = agg[prof]; a["runs"] += 1; a["q"] += n
    if d.get("evidence_at_5_rate") is not None: a["h5"] += d["evidence_at_5_rate"] * n
    if d.get("evidence_at_10_rate") is not None: a["h10"] += d["evidence_at_10_rate"] * n
rows = []
for prof, a in agg.items():
    q = a["q"] or 1
    rows.append((a["h5"]/q, a["h10"]/q, prof, a["runs"], a["q"]))
rows.sort(reverse=True)
out = ["# LME-S DEV weight sweep", "", "| profile | runs | q | ev@5 | ev@10 |", "|---|---:|---:|---:|---:|"]
for h5, h10, prof, runs, q in rows:
    out.append(f"| {prof} | {runs} | {q} | {h5:.4f} | {h10:.4f} |")
(root / "aggregate.md").write_text("\n".join(out) + "\n")
print("\n".join(out))
PY
log "=== LME-S DEV weight sweep DONE → $RUNS/aggregate.md ==="
