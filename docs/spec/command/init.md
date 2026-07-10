# rekal init

**Role:** Bootstrap Rekal in a git repository. The only command a developer must run once per repo.

**Invocation:** `rekal init`.

---

## Preconditions

- Must be run inside a git repository. Otherwise exit with "not a git repository".

---

## What init does

1. **Resolve git root** — Exit if not in a git repo.
2. **Check if already initialized** — If `.rekal/` exists, do **not** rebuild the store. Instead refresh the version-managed assets (re-install the skill suite and hooks, ensure the `.claude` gitignore entry), print "already initialized. Refreshed skills and hooks.", and exit. This is the upgrade path: after updating the binary, `rekal init` pulls new or changed skills into an existing repo without a clean+reimport. Data — the store, orphan branch, and index — is left untouched. A full reinitialize still requires `rekal clean` first.
3. **Create `.rekal/`** — Directory for local databases.
4. **Create data DB** — Open `.rekal/data.db`, run data DDL (sessions, turns, tool_calls, checkpoints, files_touched, checkpoint_sessions, checkpoint_state).
5. **Create index DB** — Open `.rekal/index.db`, run index DDL (turns_ft, tool_calls_index, files_index, session_facets, file_cooccurrence, session_embeddings, index_state).
6. **Update `.gitignore`** — Append `.rekal/` if not already present.
7. **Install hooks:**
   - `post-commit` — runs `rekal checkpoint`
   - `pre-push` — runs `rekal push`
   - Hooks contain the marker `# managed by rekal`. Existing non-Rekal hooks are not overwritten.
8. **Create orphan branch** — `rekal/<email>` with empty `rekal.body` and `dict.bin`. If the branch exists on the remote, fetch it. If it exists locally, leave it.
9. **Import existing data** — If the orphan branch has data (body > 9 bytes), import sessions and checkpoints into data DB.
10. **Install Claude Code skills** — Write the skill suite to `.claude/skills/<name>/SKILL.md` (rekal, rekal-provenance, rekal-reflect, rekal-distill, rekal-census) for agent integration. Always overwritten — skills are managed by rekal and track the binary version.
11. **Gitignore `.claude`** — If `.claude/` already existed (user has settings, CLAUDE.md, etc.), only ignore `.claude/skills/`. Otherwise ignore the entire `.claude/` directory.
12. **Initial checkpoint** — Capture any existing sessions.
13. **Print** — `Rekal initialized.`

---

## No flags

No user-facing flags. Non-interactive.
