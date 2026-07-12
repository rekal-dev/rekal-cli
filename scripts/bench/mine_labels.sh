#!/usr/bin/env bash
# mine_labels.sh <outdir> — extract gold labels for T1/T2/T3 from the repo's
# own rekal ledger + git. Emits labels-t1.jsonl / labels-t2.jsonl /
# labels-t3.jsonl. Label content stays local (04-data-plan.md §7).
set -euo pipefail
OUT="${1:?usage: mine_labels.sh <outdir>}"
mkdir -p "$OUT"

DEFAULT_BRANCH="$(git symbolic-ref --short refs/remotes/origin/HEAD 2>/dev/null | sed 's|origin/||' || echo main)"

# ---- T1: commit → producing session(s). One label per checkpoint; gold is
# the session set; query material (subject + files) comes from git, which is
# not indexed session text (the built-in leakage control).
rekal query "SELECT c.id AS cp, c.git_sha AS sha, c.git_branch AS branch,
  string_agg(cs.session_id, ',') AS sids
  FROM checkpoint_sessions cs JOIN checkpoints c ON c.id = cs.checkpoint_id
  GROUP BY c.id, c.git_sha, c.git_branch" \
| while read -r row; do
  sha=$(jq -r '.sha' <<<"$row")
  subject=$(git show -s --format='%s' "$sha" 2>/dev/null || true)
  [ -z "$subject" ] && continue
  files=$(git show --name-only --format= "$sha" 2>/dev/null | head -20 | jq -R . | jq -sc .)
  jq -nc --argjson r "$row" --arg subject "$subject" --argjson files "$files" \
    '{task:"t1", gold: ($r.sids | split(",")), commit: $r.sha, branch: $r.branch,
      source: {subject: $subject, files: $files}}'
done > "$OUT/labels-t1.jsonl"

# ---- T2: decision recall — substantial human_steering turns; gold is the
# session (turn-level hit checked against snippet_turn_index at scoring).
rekal query "SELECT session_id AS sid, turn_index AS ti, content
  FROM turns WHERE role = 'human_steering' AND length(content) BETWEEN 80 AND 2000" \
| jq -c '{task:"t2", gold: [.sid], turn_index: .ti, source: {content: .content}}' \
  > "$OUT/labels-t2.jsonl"

# ---- T3: dead-end awareness — sessions on branches that never reached the
# default branch. git decides abandonment; rekal supplies the sessions.
rekal query "SELECT DISTINCT c.git_branch AS branch FROM checkpoints c
  WHERE c.git_branch NOT IN ('main','master')" \
| jq -r '.branch' | while read -r branch; do
  tip=$(git rev-parse --verify --quiet "$branch" || git rev-parse --verify --quiet "origin/$branch" || true)
  [ -z "$tip" ] && continue
  if git merge-base --is-ancestor "$tip" "origin/$DEFAULT_BRANCH" 2>/dev/null; then
    continue # merged — not a dead end
  fi
  sids=$(rekal query "SELECT string_agg(DISTINCT cs.session_id, ',') AS s
    FROM checkpoint_sessions cs JOIN checkpoints c ON c.id = cs.checkpoint_id
    WHERE c.git_branch = '$(printf %s "$branch" | sed "s/'/''/g")'" | jq -r '.s // empty')
  [ -z "$sids" ] && continue
  intent=$(rekal query "SELECT string_agg(t.content, ' | ') AS i FROM turns t
    WHERE t.role IN ('human','human_steering') AND t.session_id IN
    ($(printf %s "$sids" | awk -F, '{for(i=1;i<=NF;i++) printf "%s'\''%s'\''", (i>1?",":""), $i}'))
    AND length(t.content) BETWEEN 40 AND 500" | jq -r '.i // empty' | head -c 2000)
  [ -z "$intent" ] && continue
  jq -nc --arg branch "$branch" --arg sids "$sids" --arg intent "$intent" \
    '{task:"t3", gold: ($sids | split(",")), branch: $branch, source: {intent: $intent}}'
done > "$OUT/labels-t3.jsonl"

for f in t1 t2 t3; do
  echo "labels-$f.jsonl: $(wc -l < "$OUT/labels-$f.jsonl") labels" >&2
done
