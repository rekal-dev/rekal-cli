#!/usr/bin/env bash
# corpus_card.sh — aggregate corpus stats (no content) as one JSON object.
# Run from an initialized rekal repo.
set -euo pipefail

q() { rekal query "$1"; }

jq -n \
  --argjson sessions        "$(q 'SELECT count(*) AS n FROM sessions' | jq '.n')" \
  --argjson turns           "$(q 'SELECT count(*) AS n FROM turns' | jq '.n')" \
  --argjson tool_calls      "$(q 'SELECT count(*) AS n FROM tool_calls' | jq '.n')" \
  --argjson checkpoints     "$(q 'SELECT count(*) AS n FROM checkpoints' | jq '.n')" \
  --argjson linked          "$(q 'SELECT count(*) AS n FROM checkpoint_sessions' | jq '.n')" \
  --argjson steering_turns  "$(q "SELECT count(*) AS n FROM turns WHERE role = 'human_steering'" | jq '.n')" \
  --argjson roles           "$(q 'SELECT role, count(*) AS n FROM turns GROUP BY role ORDER BY n DESC' | jq -sc .)" \
  --argjson branches        "$(q 'SELECT count(DISTINCT git_branch) AS n FROM checkpoints' | jq '.n')" \
  --arg     first           "$(q 'SELECT min(captured_at) AS t FROM sessions' | jq -r '.t')" \
  --arg     last            "$(q 'SELECT max(captured_at) AS t FROM sessions' | jq -r '.t')" \
  --arg     rekal_version   "$(rekal version 2>/dev/null | head -1)" \
  --arg     generated       "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  '{generated: $generated, rekal_version: $rekal_version,
    sessions: $sessions, turns: $turns, tool_calls: $tool_calls,
    checkpoints: $checkpoints, checkpoint_session_links: $linked,
    steering_turns: $steering_turns, distinct_branches: $branches,
    date_range: {first: $first, last: $last}, roles: $roles}'
