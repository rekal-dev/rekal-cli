#!/usr/bin/env bash
# Re-ingest LoCoMo (10 conv) and sweep skill weight profiles for evidence@k.
# Runs detached-friendly; keeps workers low so LME-S recover keeps CPU.
#
#   REKAL=./rekal bash scripts/industry-bench/run_locomo_weight_sweep.sh
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
REKAL="${REKAL:-$REPO_ROOT/rekal}"
IMB="${IMB_LOCOMO:-$HOME/imb-locomo}"
IN="$REPO_ROOT/scripts/industry-bench/datasets/data/locomo-conversations.jsonl"
SHIM="$REPO_ROOT/scripts/industry-bench/shim/shim.py"
CAL_DIR="$REPO_ROOT/scripts/industry-bench/calibration"
RUNS="$REPO_ROOT/scripts/industry-bench/runs/sweep/locomo-weights-$(date +%Y%m%d-%H%M)"
LOG="$HOME/imb-locomo-weight-sweep.log"
WORKERS="${WORKERS:-2}"

log(){ echo "[$(date -Iseconds)] $*" | tee -a "$LOG"; }

mkdir -p "$IMB" "$RUNS" "$CAL_DIR"

# --- profiles to try (gate+weights; shim passes --weights + REKAL_HUNT_*) ---
write_profiles() {
  python3 - "$CAL_DIR" <<'PY'
import json, sys
from pathlib import Path
d = Path(sys.argv[1])
chat_gate = {"CONF_MIN": 0.55, "CONF_SOFT": 0.50, "GAP_MIN": 0.03, "MASS_MIN": 0, "KNOWLEDGE_MIN": 0.90}
profiles = {
  "locomo-stock-gates-only": {
    "gate": chat_gate,
    "weights": None,
    "notes": "chat gates, default weights",
  },
  "locomo-chat": {
    "gate": chat_gate,
    "weights": {
      "bm25": 0.30, "lsa": 0.20, "semantic": 0.50,
      "steering_boost": 1.0, "summary_boost": 1.0,
      "subagent_downweight": 1.0, "facet_boost": 0.1,
    },
  },
  "locomo-multi-hop": {
    "gate": {**chat_gate, "CONF_MIN": 0.52},
    "weights": {
      "bm25": 0.25, "lsa": 0.25, "semantic": 0.50,
      "steering_boost": 1.2, "summary_boost": 1.1,
      "subagent_downweight": 0.8, "facet_boost": 0.2,
    },
  },
  "locomo-bm25-push": {
    "gate": chat_gate,
    "weights": {
      "bm25": 0.55, "lsa": 0.10, "semantic": 0.35,
      "steering_boost": 1.0, "summary_boost": 1.0,
      "subagent_downweight": 1.0, "facet_boost": 0.05,
    },
  },
  "locomo-semantic-push": {
    "gate": chat_gate,
    "weights": {
      "bm25": 0.20, "lsa": 0.15, "semantic": 0.65,
      "steering_boost": 1.0, "summary_boost": 1.0,
      "subagent_downweight": 1.0, "facet_boost": 0.05,
    },
  },
  "locomo-lsa-push": {
    "gate": chat_gate,
    "weights": {
      "bm25": 0.25, "lsa": 0.35, "semantic": 0.40,
      "steering_boost": 1.0, "summary_boost": 1.0,
      "subagent_downweight": 1.0, "facet_boost": 0.1,
    },
  },
}
for name, body in profiles.items():
  p = d / f"{name}.json"
  p.write_text(json.dumps(body, indent=2) + "\n")
  print(p)
PY
}

log "=== LoCoMo weight sweep START workers=$WORKERS ==="
write_profiles

# Ingest if missing
n_repos=$(find "$IMB" -path '*/repo/.rekal/data.db' 2>/dev/null | wc -l | tr -d ' ')
if [[ "${n_repos:-0}" -lt 10 ]]; then
  log "ingest LoCoMo into $IMB (have $n_repos/10)"
  rm -rf "$IMB"
  mkdir -p "$IMB"
  PYTHONUNBUFFERED=1 python3 "$REPO_ROOT/scripts/industry-bench/sh_gen/gen.py" \
    --input "$IN" --out "$IMB" --rekal "$REKAL" --verify --index \
    > "$IMB/ingest.log" 2>&1 || log "ingest finished with soft errors (see $IMB/ingest.log)"
  n_repos=$(find "$IMB" -path '*/repo/.rekal/data.db' 2>/dev/null | wc -l | tr -d ' ')
  log "ingest done repos=$n_repos"
else
  log "reuse existing ingest repos=$n_repos"
fi

# Discover conversation ids (macOS /bin/bash 3.2 portable — no mapfile / procsubst)
CONV_FILE="$(mktemp)"
python3 -c "
import json
from pathlib import Path
src = Path(r'''$IN''')
dst = Path(r'''$CONV_FILE''')
with open(dst, 'w') as o:
    for line in src.read_text().splitlines():
        if line.strip():
            o.write(json.loads(line)['conversation_id'] + '\n')
"
CONVS=()
while IFS= read -r line; do
  [ -n "$line" ] && CONVS[${#CONVS[@]}]="$line"
done < "$CONV_FILE"
rm -f "$CONV_FILE"
log "conversations=${#CONVS[@]}"

run_one() {
  local route=$1 cal=$2 conv=$3
  local tag
  if [[ -n "$cal" ]]; then
    tag="$(basename "$cal" .json)"
  else
    tag="no-cal"
  fi
  local out="$RUNS/${conv}-${route}-${tag}"
  if [[ -f "$out/manifest.json" ]]; then
    echo "skip $conv $route $tag"
    return 0
  fi
  local args=(smoke --workdir "$IMB" --conversation-id "$conv" --input "$IN" --out "$out" --rekal "$REKAL" --route "$route")
  if [[ -n "$cal" ]]; then
    args+=(--calibration "$cal")
  fi
  echo "start $conv $route $tag"
  python3 "$SHIM" "${args[@]}" >> "$LOG" 2>&1 || echo "fail $conv $route $tag"
  echo "done $conv $route $tag"
}
export -f run_one
export IMB IN SHIM REKAL RUNS LOG

# Baseline stock (no cal) + skill with each profile
JOBS=()
for conv in "${CONVS[@]}"; do
  JOBS+=("stock||$conv")
  JOBS+=("skill||$conv")  # skill, default weights, no cal gates
  for cal in "$CAL_DIR"/locomo-*.json; do
    JOBS+=("skill|$cal|$conv")
  done
done

log "jobs=${#JOBS[@]} → $RUNS"
printf '%s\n' "${JOBS[@]}" | xargs -P "$WORKERS" -I{} bash -c '
  IFS="|" read -r route cal conv <<<"{}"
  run_one "$route" "$cal" "$conv"
'

# Aggregate
python3 - "$RUNS" <<'PY'
import json, sys
from pathlib import Path
from collections import defaultdict
root = Path(sys.argv[1])
buckets = defaultdict(lambda: {"n": 0, "h5": 0.0, "h10": 0.0, "runs": 0})
for m in root.glob("*/manifest.json"):
    d = json.loads(m.read_text())
    name = m.parent.name
    # ...-stock-no-cal or ...-skill-<profile>
    parts = name.split("-")
    # conversation ids like conv-26
    # find route
    if "-stock-" in name:
        key = "stock/no-cal"
    else:
        # skill-<tag> at end
        tag = name.split("-skill-", 1)[-1] if "-skill-" in name else "skill"
        key = f"skill/{tag}"
    n = int(d.get("n_questions") or 0)
    r5 = d.get("evidence_at_5_rate")
    r10 = d.get("evidence_at_10_rate")
    if n <= 0 or r5 is None:
        continue
    b = buckets[key]
    b["n"] += n
    b["h5"] += float(r5) * n
    b["h10"] += float(r10 or 0) * n
    b["runs"] += 1

lines = ["# LoCoMo weight sweep", "", f"runs_dir: `{root}`", "",
         "| profile | convs | q | evidence@5 | evidence@10 |",
         "|---|---:|---:|---:|---:|"]
rows = []
for k, b in buckets.items():
    if b["n"] == 0:
        continue
    rows.append((b["h5"] / b["n"], k, b))
rows.sort(reverse=True)
for score, k, b in rows:
    lines.append(
        f"| {k} | {b['runs']} | {b['n']} | {b['h5']/b['n']:.4f} | {b['h10']/b['n']:.4f} |"
    )
out = root / "aggregate.md"
out.write_text("\n".join(lines) + "\n")
(root / "aggregate.json").write_text(json.dumps({
    k: {"runs": b["runs"], "n_questions": b["n"],
        "evidence_at_5": b["h5"]/b["n"] if b["n"] else None,
        "evidence_at_10": b["h10"]/b["n"] if b["n"] else None}
    for k, b in buckets.items() if b["n"]
}, indent=2) + "\n")
print(out.read_text())
PY

log "=== LoCoMo weight sweep DONE → $RUNS ==="
