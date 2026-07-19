#!/usr/bin/env bash
# Map watermark tool. Two subcommands, deterministic data for the agent:
#
#   map.sh fresh [map.md]      Check .rekal/map.md watermark against HEAD.
#                              Prints: FRESH <head> | STALE <old> <head> | MISSING
#                              On STALE, prints name-only diff (old..HEAD) so the
#                              agent rewrites only touched subsystems.
#                              Exit: 0 fresh, 1 stale/missing, 2 usage/error
#
#   map.sh watermark [map.md]  Write/refresh line 1 with current branch + HEAD.
#                              Creates a stub body when missing; never regenerates
#                              content. Prints: WROTE <marker>
#                              Exit: 0 ok, 2 not a git repo / no HEAD
set -euo pipefail

cmd="${1:-fresh}"
map="${2:-.rekal/map.md}"

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "rekal: not a git repository" >&2
  exit 2
fi

case "${cmd}" in
fresh)
  head="$(git rev-parse HEAD 2>/dev/null || true)"
  if [[ -z "${head}" || "${head//0/}" == "" ]]; then
    echo "MISSING"
    exit 1
  fi
  if [[ ! -f "${map}" ]]; then
    echo "MISSING"
    exit 1
  fi
  line="$(head -n 1 "${map}" || true)"
  if [[ ! "${line}" =~ rekal-map[[:space:]]+[^[:space:]]+[[:space:]]+([0-9a-fA-F]+) ]]; then
    echo "STALE unknown ${head}"
    exit 1
  fi
  watermark="${BASH_REMATCH[1]}"
  if [[ "${watermark}" == "${head}" ]]; then
    echo "FRESH ${head}"
    exit 0
  fi
  echo "STALE ${watermark} ${head}"
  git diff --name-only "${watermark}" HEAD 2>/dev/null | head -n 80 || true
  exit 1
  ;;
watermark)
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
  ;;
*)
  echo "usage: map.sh {fresh|watermark} [map.md]" >&2
  exit 2
  ;;
esac
