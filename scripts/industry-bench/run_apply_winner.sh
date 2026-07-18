#!/usr/bin/env bash
# Apply the winning calibration (bm25-push, dev-validated) to a benchmark and
# produce a final stock-vs-skill(winner) aggregate. Repos must already be
# ingested + indexed. Works for both flat (workdir/<conv>/repo) and sharded
# (workdir/shard-*/<conv>/repo) layouts.
#
#   REKAL=./rekal BENCH=locomo IMB_ROOT=~/imb-locomo \
#     INPUT=scripts/industry-bench/datasets/data/locomo-conversations.jsonl \
#     CAL=scripts/industry-bench/calibration/locomo-bm25-push.json \
#     WORKERS=2 FLATTEN=0 bash scripts/industry-bench/run_apply_winner.sh
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
REKAL="${REKAL:-$REPO_ROOT/rekal}"
BENCH="${BENCH:?set BENCH}"
IMB_ROOT="${IMB_ROOT:?set IMB_ROOT}"
INPUT="${INPUT:?set INPUT}"
CAL="${CAL:?set CAL}"
WORKERS="${WORKERS:-2}"
FLATTEN="${FLATTEN:-0}"
# Which skill route to run as the non-stock arm: skill (single-shot) or
# skill-multi (reformulate+RRF multi-lookup). The output dirs still use the
# literal "skill" tag so aggregation is route-agnostic.
SKILL_ROUTE="${SKILL_ROUTE:-skill}"
DATE_TAG="${DATE_TAG:-winner-$(date +%Y%m%d)}"
SHIM="$REPO_ROOT/scripts/industry-bench/shim/shim.py"
RUNS="${RUNS:-$REPO_ROOT/scripts/industry-bench/runs/full/${BENCH}-${DATE_TAG}}"
LOG="${LOG:-$HOME/imb-${BENCH}-winner.log}"

log(){ echo "[$(date -Iseconds)] $*" | tee -a "$LOG"; }
mkdir -p "$RUNS"

WORKDIR="$IMB_ROOT"
if [[ "$FLATTEN" == "1" ]]; then
  FLAT="$IMB_ROOT/flat"
  mkdir -p "$FLAT"
  for shard in "$IMB_ROOT"/shard-*/; do
    [[ -d "$shard" ]] || continue
    for conv in "$shard"*/; do
      [[ -d "${conv}repo/.rekal" ]] || continue
      name="$(basename "$conv")"
      [[ -e "$FLAT/$name" ]] || ln -sfn "$(cd "$conv" && pwd)" "$FLAT/$name"
    done
  done
  WORKDIR="$FLAT"
  log "flattened → $FLAT ($(find "$FLAT" -maxdepth 1 -type l | wc -l | tr -d ' ') convs)"
fi

# conv ids from input
CONV_FILE="$(mktemp)"
python3 -c "
import json
for l in open(r'''$INPUT'''):
    l=l.strip()
    if l: print(json.loads(l)['conversation_id'])
" > "$CONV_FILE"
N=$(wc -l < "$CONV_FILE" | tr -d ' ')
log "=== apply-winner $BENCH START convs=$N cal=$(basename "$CAL") workers=$WORKERS tag=$DATE_TAG ==="

run_one(){
  local route=$1 conv=$2
  local out="$RUNS/${conv}-${route}-${DATE_TAG}"
  [[ -f "$out/manifest.json" ]] && { echo "skip $conv $route"; return 0; }
  [[ -d "$WORKDIR/$conv/repo/.rekal" ]] || { echo "missing $conv" >&2; return 0; }
  if [[ "$route" == skill ]]; then
    python3 "$SHIM" smoke --workdir "$WORKDIR" --conversation-id "$conv" --input "$INPUT" \
      --out "$out" --rekal "$REKAL" --route "$SKILL_ROUTE" --calibration "$CAL" >> "$LOG" 2>&1 || echo "fail $conv skill"
  else
    python3 "$SHIM" smoke --workdir "$WORKDIR" --conversation-id "$conv" --input "$INPUT" \
      --out "$out" --rekal "$REKAL" --route stock >> "$LOG" 2>&1 || echo "fail $conv stock"
  fi
  echo "done $conv $route"
}
export -f run_one
export RUNS DATE_TAG WORKDIR INPUT REKAL SHIM CAL LOG SKILL_ROUTE

JOBFILE="$(mktemp)"
while IFS= read -r conv; do
  [[ -n "$conv" ]] || continue
  echo "stock $conv" >> "$JOBFILE"
  echo "skill $conv" >> "$JOBFILE"
done < "$CONV_FILE"
rm -f "$CONV_FILE"

xargs -P "$WORKERS" -n 2 bash -c 'run_one "$1" "$2"' _ < "$JOBFILE"
rm -f "$JOBFILE"

python3 "$REPO_ROOT/scripts/industry-bench/scoring/aggregate_smokes.py" \
  --smoke-dir "$RUNS" --out "$RUNS/aggregate-${DATE_TAG}.json" 2>&1 | tee -a "$LOG"

# Winner-vs-stock summary (weighted by n_questions)
python3 - "$RUNS" "$DATE_TAG" <<'PY'
import json,sys
from pathlib import Path
root=Path(sys.argv[1]); tag=sys.argv[2]
agg={"stock":{"n":0,"h5":0.0,"h10":0.0,"ct":0.0,"r":0},"skill":{"n":0,"h5":0.0,"h10":0.0,"ct":0.0,"r":0}}
for m in root.glob("*/manifest.json"):
    d=json.loads(m.read_text()); name=m.parent.name
    route="skill" if "-skill-" in name else "stock"
    n=int(d.get("n_questions") or 0); r5=d.get("evidence_at_5_rate")
    if n<=0 or r5 is None: continue
    a=agg[route]; a["n"]+=n; a["h5"]+=float(r5)*n
    a["h10"]+=float(d.get("evidence_at_10_rate") or 0)*n
    a["ct"]+=float(d.get("retrieved_context_tokens_mean") or 0)*n; a["r"]+=1
lines=[f"# {root.name} — apply-winner final","",
       "| variant | runs | q | ev@5 | ev@10 | ctx_tok |","|---|---:|---:|---:|---:|---:|"]
for route in ("stock","skill"):
    a=agg[route]; n=a["n"] or 1
    lbl="stock" if route=="stock" else "skill WINNER (bm25-push)"
    lines.append(f"| {lbl} | {a['r']} | {a['n']} | {a['h5']/n:.4f} | {a['h10']/n:.4f} | {a['ct']/n:.1f} |")
(root/f"aggregate-{tag}.md").write_text("\n".join(lines)+"\n")
print("\n".join(lines))
PY

log "=== apply-winner $BENCH DONE → $RUNS ==="
