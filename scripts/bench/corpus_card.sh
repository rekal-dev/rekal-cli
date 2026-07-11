#!/usr/bin/env bash
# corpus_card.sh — aggregate corpus stats (no content) as one JSON object.
# Run from an initialized rekal repo.
set -euo pipefail

q() { rekal query "$1"; }

jq -n \
  --argjson sessions        "$(q 'SELECT count(*) AS n FROM sessions' | jq '.[0].n')" \
  --argjson turns           "$(q 'SELECT count(*) AS n FROM turns' | jq '.[0].n')" \
  --argjson tool_calls      "$(q 'SELECT count(*) AS n FROM tool_calls' | jq '.[0].n')" \
  --argjson checkpoints     "$(q 'SELECT count(*) AS n FROM checkpoints' | jq '.[0].n')" \
  --argjson linked          "$(q 'SELECT count(*) AS n FROM checkpoint_sessions' | jq '.[0].n')" \
  --argjson steering_turns  "$(q "SELECT count(*) AS n FROM turns WHERE role = 'human_steering'" | jq '.[0].n')" \
  --argjson roles           "$(q 'SELECT role, count(*) AS n FROM turns GROUP BY role ORDER BY n DESC')" \
  --argjson branches        "$(q 'SELECT count(DISTINCT git_branch) AS n FROM checkpoints' | jq '.[0].n')" \
  --arg     first           "$(q 'SELECT min(captured_at) AS t FROM sessions' | jq -r '.[0].t')" \
  --arg     last            "$(q 'SELECT max(captured_at) AS t FROM sessions' | jq -r '.[0].t')" \
  --arg     rekal_version   "$(rekal version 2>/dev/null | head -1)" \
  --arg     generated       "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  '{generated: $generated, rekal_version: $rekal_version,
    sessions: $sessions, turns: $turns, tool_calls: $tool_calls,
    checkpoints: $checkpoints, checkpoint_session_links: $linked,
    steering_turns: $steering_turns, distinct_branches: $branches,
    date_range: {first: $first, last: $last}, roles: $roles}'
