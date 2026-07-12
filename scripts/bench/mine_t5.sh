#!/usr/bin/env bash
# mine_t5.sh <outdir> — T5 decision-drift CANDIDATES (opportunistic). Emits
# labels-t5-candidates.jsonl for MANUAL confirmation — this is not auto-gold.
#
# Heuristic (03-benchmark.md T5): two sessions that touched the same file set,
# >= MIN_GAP sessions apart in time, with a human_steering turn in each — a
# later decision may reverse an earlier one. Whether it is an actual reversal
# needs a human read; confirm and relabel task:"t5" before scoring. Expect few
# or none on repos without a real reversal history (06-eval-strategy.md).
set -euo pipefail
OUT="${1:?usage: mine_t5.sh <outdir>}"
mkdir -p "$OUT"
MIN_SHARED="${T5_MIN_SHARED:-2}"
LIMIT="${T5_LIMIT:-100}"

# Pairs sharing files where both sessions carry steering, later session second.
rekal query --index "
  WITH steered AS (
    SELECT DISTINCT session_id FROM turns_ft WHERE role = 'human_steering'
  ),
  fa AS (
    SELECT fi.session_id, fi.file_path, sf.captured_at
    FROM files_index fi JOIN session_facets sf ON sf.session_id = fi.session_id
    WHERE fi.session_id IN (SELECT session_id FROM steered)
  )
  SELECT a.session_id AS earlier, b.session_id AS later,
         COUNT(DISTINCT a.file_path) AS shared,
         string_agg(DISTINCT a.file_path, ',') AS files
  FROM fa a JOIN fa b
    ON a.file_path = b.file_path AND a.captured_at < b.captured_at
   AND a.session_id <> b.session_id
  GROUP BY a.session_id, b.session_id
  HAVING COUNT(DISTINCT a.file_path) >= $MIN_SHARED
  ORDER BY shared DESC
  LIMIT $LIMIT" \
| while read -r row; do
  earlier=$(jq -r '.earlier' <<<"$row"); later=$(jq -r '.later' <<<"$row")
  e_intent=$(rekal query "SELECT string_agg(content, ' | ') AS i FROM turns
    WHERE session_id='$earlier' AND role='human_steering'
    AND length(content) BETWEEN 40 AND 500" | jq -r '.i // empty' | head -c 800)
  l_intent=$(rekal query "SELECT string_agg(content, ' | ') AS i FROM turns
    WHERE session_id='$later' AND role='human_steering'
    AND length(content) BETWEEN 40 AND 500" | jq -r '.i // empty' | head -c 800)
  if [ -z "$e_intent" ] || [ -z "$l_intent" ]; then continue; fi
  files=$(jq -r '.files' <<<"$row" | tr ',' '\n' | head -8 | jq -R . | jq -sc .)
  jq -nc --arg earlier "$earlier" --arg later "$later" \
    --arg ei "$e_intent" --arg li "$l_intent" --argjson files "$files" \
    --argjson shared "$(jq '.shared' <<<"$row")" \
    '{task:"t5-candidate", gold:[$later], earlier:$earlier, shared:$shared,
      source:{earlier_intent:$ei, later_intent:$li, files:$files},
      confirm:"manual: is later a reversal/supersession of earlier?"}'
done > "$OUT/labels-t5-candidates.jsonl"
echo "labels-t5-candidates.jsonl: $(wc -l < "$OUT/labels-t5-candidates.jsonl") candidates (confirm manually)" >&2
