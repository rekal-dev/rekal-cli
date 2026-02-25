# rekal clean

**Role:** Undo everything init did. Local only — does not touch remote git (no delete branch, no force-push, no changing remote refs).

**Invocation:** Subcommand only — `rekal clean`. No short form at root; root is reserved for search/recall.

---

## Preconditions

- Must be run inside a git repository. Otherwise exit with a clear error.

---

## What clean does

1. **Resolve git root** — Exit if not in a git repo.
2. **Remove `.rekal/`** — Delete the directory and all contents (config, local DBs, etc.).
3. **Remove Rekal hooks** — If post-commit and pre-push hooks are the Rekal hooks (marker), remove them. Leave other hooks unchanged.
4. **`.gitignore`** — Do not modify `.gitignore`. Leave as-is (next init will recreate `.rekal/`).
5. **Exit** — Print e.g. "Rekal cleaned. Run `rekal init` to reinitialize."

---

## Idempotent

Running clean again after the first time is a no-op (nothing to remove).

---

## No flags

No user-facing flags.
