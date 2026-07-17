#!/usr/bin/env bash
# Parallel rekal index over all ingested repos under a workdir tree.
# Usage: REKAL=./rekal WORKERS=8 ./run_index_all.sh ~/imb-lme-s
set -euo pipefail

ROOT="${1:?workdir required}"
REKAL="${REKAL:-rekal}"
WORKERS="${WORKERS:-8}"

if [[ ! -x "$REKAL" ]] && ! command -v "$REKAL" >/dev/null; then
  echo "rekal not found: $REKAL" >&2
  exit 1
fi

REPOS="$(find "$ROOT" -path '*/repo/.rekal' -print | sed 's|/.rekal||' | sort)"
N="$(echo "$REPOS" | grep -c . || true)"
echo "indexing $N repos under $ROOT (workers=$WORKERS)"

index_one() {
  local repo="$1"
  if [[ -f "$repo/.rekal/index.db" ]]; then
    echo "skip $(basename "$(dirname "$repo")") (indexed)"
    return 0
  fi
  echo "index $(basename "$(dirname "$repo")")"
  (cd "$repo" && "$REKAL" index) > "$(dirname "$repo")/index.log" 2>&1
}
export -f index_one
export REKAL

if command -v parallel >/dev/null 2>&1; then
  echo "$REPOS" | parallel -j "$WORKERS" index_one {}
else
  echo "$REPOS" | xargs -P "$WORKERS" -n 1 bash -c 'index_one "$1"' _ 
fi

echo "done: indexed $N repos"
