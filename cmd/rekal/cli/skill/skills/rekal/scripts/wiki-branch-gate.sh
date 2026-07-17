#!/usr/bin/env bash
# Refuse wiki writes on the default branch. PR is the admission gate.
# Exit: 0 ok (feature branch), 1 on default branch, 2 error
set -euo pipefail

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "rekal: not a git repository" >&2
  exit 2
fi
branch="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true)"
if [[ -z "${branch}" || "${branch}" == "HEAD" ]]; then
  echo "SILENCE reason=detached_head" >&2
  exit 1
fi
default="$(git symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null | sed 's#^origin/##' || true)"
if [[ -z "${default}" ]]; then
  for cand in main master; do
    if git show-ref --verify --quiet "refs/heads/${cand}" || git show-ref --verify --quiet "refs/remotes/origin/${cand}"; then
      default="${cand}"
      break
    fi
  done
fi
default="${default:-main}"
if [[ "${branch}" == "${default}" ]]; then
  echo "SILENCE branch=${branch} reason=default_branch_no_wiki_write"
  exit 1
fi
echo "PASS branch=${branch} default=${default}"
exit 0
