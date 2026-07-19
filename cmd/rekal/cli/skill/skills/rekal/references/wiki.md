# Wiki — materialize `docs/wiki/` via PR

Browseable topic pages from the ledger's correlations — not a file crawl.

## Gate (required)

```bash
ROOT="${CLAUDE_SKILL_DIR:-$(git rev-parse --show-toplevel)/.claude/skills/rekal}"
bash "$ROOT/scripts/wiki-gate.sh"
```

Exit 1 on the default branch → create a feature branch first. Merge is the
admission gate. Never write wiki pages straight to main/master.

## Rules

1. **Graph stays virtual** — clusters from `file_cooccurrence` / `files_index`
   at generation time; only markdown is persisted.
2. **Distrust dirty commit messages** — cite SHA; never quote "update" as
   evidence; derive why from the session behind the commit.

## Workflow

1. Discover topics: co-occurrence clusters of heavily touched paths.
2. Per topic: gather sessions that touch those files; summarise; list key
   decisions with session/commit pointers.
3. Write `docs/wiki/<topic>.md` + update `docs/wiki/index.md`.
4. Open a PR. Review admits the page.

Each page: summary, key decisions, provenance links (sessions, commits, files).
