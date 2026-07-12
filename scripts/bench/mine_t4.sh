#!/usr/bin/env bash
# mine_t4.sh <outdir> — T4 multi-hop gold: pairs of sessions that co-touched
# files, for questions answerable only from BOTH. Emits labels-t4.jsonl.
#
# Run inside a checkpoint-bearing repo: files_index only holds this repo's own
# checkpoint sessions (imported cross-repo sessions have no checkpoints), so
# the gold is automatically own-repo. Pool across repos by running this per
# labeled repo and concatenating the outputs. The scored haystack is still the
# full machine-wide index (06-eval-strategy.md).
set -euo pipefail
OUT="${1:?usage: mine_t4.sh <outdir>}"
mkdir -p "$OUT"
MIN_SHARED="${T4_MIN_SHARED:-2}"   # min shared files to call a pair linked
LIMIT="${T4_LIMIT:-200}"           # cap candidate pairs (strongest links first)

# Session pairs sharing >= MIN_SHARED files, strongest links first. session_id
# < session_id keeps each unordered pair once; distinct files guard fan-out.
rekal query --index "
  SELECT a.session_id AS s1, b.session_id AS s2,
         COUNT(DISTINCT a.file_path) AS shared,
         string_agg(DISTINCT a.file_path, ',') AS files
  FROM files_index a JOIN files_index b
    ON a.file_path = b.file_path AND a.session_id < b.session_id
  GROUP BY a.session_id, b.session_id
  HAVING COUNT(DISTINCT a.file_path) >= $MIN_SHARED
  ORDER BY shared DESC
  LIMIT $LIMIT" \
| while read -r row; do
  s1=$(jq -r '.s1' <<<"$row"); s2=$(jq -r '.s2' <<<"$row")
  # A distinctive human/steering turn from each session gives the query
  # generator material from both halves of the hop.
  snip1=$(rekal query "SELECT content FROM turns WHERE session_id='$s1'
    AND role IN ('human','human_steering') AND length(content) BETWEEN 60 AND 600
    ORDER BY length(content) DESC LIMIT 1" | jq -r '.content // empty' | head -c 600)
  snip2=$(rekal query "SELECT content FROM turns WHERE session_id='$s2'
    AND role IN ('human','human_steering') AND length(content) BETWEEN 60 AND 600
    ORDER BY length(content) DESC LIMIT 1" | jq -r '.content // empty' | head -c 600)
  if [ -z "$snip1" ] || [ -z "$snip2" ]; then continue; fi
  files=$(jq -r '.files' <<<"$row" | tr ',' '\n' | head -8 | jq -R . | jq -sc .)
  jq -nc --arg s1 "$s1" --arg s2 "$s2" --arg snip1 "$snip1" --arg snip2 "$snip2" \
    --argjson files "$files" --argjson shared "$(jq '.shared' <<<"$row")" \
    '{task:"t4", gold: [$s1,$s2], shared: $shared,
      source: {s1_snippet:$snip1, s2_snippet:$snip2, files:$files}}'
done > "$OUT/labels-t4.jsonl"
echo "labels-t4.jsonl: $(wc -l < "$OUT/labels-t4.jsonl") candidate pairs" >&2
