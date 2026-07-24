# rekal clean

**Role:** Undo everything init did. Local only — does not touch remote git.

**Invocation:** `rekal clean`.

---

## Preconditions

- Must be run inside a git repository. Otherwise exit with "not a git repository".

---

## What clean does

1. **Resolve git root** — Exit if not in a git repo.
2. **Remove `.rekal/`** — Delete the directory and all contents (data DB, index DB).
3. **Remove Rekal hooks** — If `post-commit` and `pre-push` hooks contain the `# managed by rekal` marker, remove them. Leave other hooks unchanged.
4. **Remove the agent skill** — Delete `.claude/skills/rekal/` (tip + scripts + references) and any legacy companion dirs (`rekal-provenance`, `rekal-reflect`, `rekal-distill`, `rekal-census`, `rekal-wiki`), then prune `.claude/skills/` and `.claude/` only if empty. A user's own `.claude` content is never touched.
5. **Remove the managed instruction lines** — Delete only the `<!-- managed by rekal -->` marker line from every file init may have written it to: `CLAUDE.md` and the per-agent files `AGENTS.md`, `GEMINI.md`, `.github/copilot-instructions.md`, `.kiro/steering/rekal.md`. For each, if nothing but whitespace remains (the file was ours) remove the file; a file with the user's own content keeps everything else. Emptied agent directories (`.github/`, `.kiro/steering/`, `.kiro/`) are pruned only when nothing else remains in them.
5. **Do not modify `.gitignore`** — Leave as-is.
6. **Print** — `Rekal cleaned. Run 'rekal init' to reinitialize.`

---

## Idempotent

Running clean when `.rekal/` doesn't exist still prints the success message.

---

## No flags

No user-facing flags.
