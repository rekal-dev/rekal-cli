#!/usr/bin/env bash
# Check .rekal/map.md watermark against HEAD.
# Prints: FRESH | STALE <old-sha> <head> | MISSING
# Exit: 0 fresh, 1 stale/missing, 2 usage/error
set -euo pipefail

map="${1:-.rekal/map.md}"
if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "rekal: not a git repository" >&2
  exit 2
fi
head="$(git rev-parse HEAD 2>/dev/null || true)"
if [[ -z "${head}" || "${head//0/}" == "" ]]; then
  echo "MISSING"
  exit 1
fi
if [[ ! -f "${map}" ]]; then
  echo "MISSING"
  exit 1
fi
# Line 1: <!-- rekal-map <branch> <HEAD-sha> -->
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
# Name-only diff so the agent can rewrite only touched subsystems.
git diff --name-only "${watermark}" HEAD 2>/dev/null | head -n 80 || true
exit 1
