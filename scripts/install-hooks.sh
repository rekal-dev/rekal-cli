#!/usr/bin/env bash
# Install git hooks so test and lint run before push.
# Run once from repo root: ./scripts/install-hooks.sh

set -e

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"
HOOKS_DIR="$(git rev-parse --git-path hooks)"
PRE_PUSH_SRC="$ROOT/scripts/pre-push"
PRE_PUSH_DST="$HOOKS_DIR/pre-push"

if [ ! -f "$PRE_PUSH_SRC" ]; then
  echo "Error: $PRE_PUSH_SRC not found. Run from repo root." >&2
  exit 1
fi

cp "$PRE_PUSH_SRC" "$PRE_PUSH_DST"
chmod +x "$PRE_PUSH_DST"
echo "Installed pre-push hook (test:ci + lint). Push will run checks first."
