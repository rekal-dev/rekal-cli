#!/usr/bin/env bash
# Write or refresh line 1 of .rekal/map.md with the current branch + HEAD.
# Creates a stub body when the file is missing. Does not regenerate content.
# Exit: 0 ok, 2 not a git repo / no HEAD
set -euo pipefail

map="${1:-.rekal/map.md}"
if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "rekal: not a git repository" >&2
  exit 2
fi
head="$(git rev-parse HEAD 2>/dev/null || true)"
branch="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)"
if [[ -z "${head}" || "${head//0/}" == "" ]]; then
  echo "rekal: no HEAD commit" >&2
  exit 2
fi
mkdir -p "$(dirname "${map}")"
mark="<!-- rekal-map ${branch} ${head} -->"
if [[ -f "${map}" ]]; then
  body="$(tail -n +2 "${map}" 2>/dev/null || true)"
  {
    echo "${mark}"
    printf '%s' "${body}"
    # Ensure trailing newline when body was non-empty without one.
    [[ -z "${body}" || "${body}" == *$'\n' ]] || echo
  } >"${map}.tmp"
  mv "${map}.tmp" "${map}"
else
  cat >"${map}" <<EOF
${mark}
# Map — $(basename "$(git rev-parse --show-toplevel)")

## System in one paragraph

(stub — fill via map workflow)

## Subsystems

## Flows
EOF
fi
echo "WROTE ${mark}"
exit 0
