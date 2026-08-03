# rekal init

**Role:** Bootstrap Rekal in a git repository. The only command a developer must run once per repo.

**Invocation:** `rekal init`.

---

## Preconditions

- Must be run inside a git repository. Otherwise exit with "not a git repository".

---

## What init does

1. **Resolve git root** — Exit if not in a git repo.
2. **Check if already initialized** — If `.rekal/` exists, do **not** rebuild the store. Instead refresh the version-managed assets (re-install the skill tree and hooks, refresh the CLAUDE.md marker line and the detected-agent instruction lines, ensure the `.claude` gitignore entry), print "already initialized. Refreshed skills and hooks.", and exit. Data — the store, orphan branch, and index — is left untouched. A full reinitialize still requires `rekal clean` first.

   **Skill upgrades without re-init:** recall runs `maybeRefreshStaleSkill`, which compares `.claude/skills/<name>/.rekal-version` to the running binary `Version` and re-installs the skill tree on mismatch. That path touches only the gitignored skill files — not hooks or instruction lines. Explicit `rekal init` remains the upgrade path for hooks and agent rules. Dev builds that keep `Version="dev"` often will not auto-refresh (marker still matches); re-run `rekal init` in that case.
3. **Create `.rekal/`** — Directory for local databases.
4. **Create data DB** — Open `.rekal/data.db`, run data DDL (sessions, turns, tool_calls, checkpoints, files_touched, checkpoint_sessions, checkpoint_state).
5. **Create index DB** — Open `.rekal/index.db`, run index DDL (turns_ft, tool_calls_index, files_index, session_facets, file_cooccurrence, session_embeddings, index_state). Knowledge tables (`knowledge_chunks`, `knowledge_embeddings`) are created on demand by `EnsureKnowledgeSchema` at index/recall — see [db/README.md](../../db/README.md) and [knowledge-layer design](../../design/knowledge-layer.md).
6. **Update `.gitignore`** — Append `.rekal/` if not already present.
7. **Install hooks:**
   - `post-commit` — runs `rekal checkpoint`
   - `pre-push` — runs `rekal push`
   - Hooks contain the marker `# managed by rekal`. Existing non-Rekal hooks are not overwritten.
8. **Create orphan branch** — `rekal/<email>` with empty `rekal.body` and `dict.bin`. If the branch exists on the remote, fetch it. If it exists locally, leave it.
9. **Import existing data** — If the orphan branch has data (body > 9 bytes), import sessions and checkpoints into data DB.
10. **Install Claude Code skill** — Write `.claude/skills/rekal/` (one tip + `scripts/` + `references/` — progressive disclosure). Always overwritten; legacy companion dirs (`rekal-provenance`, `rekal-reflect`, `rekal-distill`, `rekal-census`, `rekal-wiki`) are purged so upgrades leave no residue.
10b. **Inject one CLAUDE.md sentence** — Append (or refresh in place, keyed by the `<!-- managed by rekal -->` marker) a single sentence pointing agents at the `rekal` skill and its routing rule. Creates CLAUDE.md when missing; never touches the user's own content. This is the whole dev-experience surface: init, done.

10c. **Install non-Claude agent instructions** — Detect which other AI agents are installed on this machine (home-dir probe: `~/.codex`, `~/.local/share/opencode`, `~/.cursor`, `~/.gemini`, `~/.copilot`) and inject one marker-tagged instruction line — naming the `rekal` commands directly, since those agents have no skill — into the file each reads: `AGENTS.md` (Codex/OpenCode/Cursor, written once), `GEMINI.md` (Gemini), `.github/copilot-instructions.md` (Copilot), `.kiro/steering/rekal.md` (Kiro). Same create-if-missing / refresh-in-place / preserve-user-content contract as the CLAUDE.md line, keyed by the same marker; clean removes the line. A file Rekal **newly creates** is gitignored (detection is per machine, so a machine-specific instructions file stays local, like the `.claude/skills/` tree); a file the user already tracked keeps its tracked status and only gains the marker line. Detection is per machine; a teammate on a different agent re-runs `rekal init` to add theirs. Best-effort — a failure for one agent never aborts init.
11. **Gitignore `.claude`** — If `.claude/` already existed (user has settings, CLAUDE.md, etc.), only ignore `.claude/skills/`. Otherwise ignore the entire `.claude/` directory.
12. **Initial checkpoint** — Capture any existing sessions.
12b. **Build the index** — Run the same structural rebuild as `rekal index`/`rekal sync` (FTS, LSA, facets, knowledge chunks) and start the background `rekal embed` for deep-semantic vectors. Non-fatal — a failure here only means the first recall pays the rebuild cost inline instead, same as before this step existed. This closes the gap where `init` reported success while the index was still empty, and the first real recall silently absorbed a full rebuild with no forewarning.
13. **Print** — `Rekal initialized.`

---

## No flags

No user-facing flags. Non-interactive.
