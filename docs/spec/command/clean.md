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
4. **Do not modify `.gitignore`** — Leave as-is.
5. **Print** — `Rekal cleaned. Run 'rekal init' to reinitialize.`

---

## Idempotent

Running clean when `.rekal/` doesn't exist still prints the success message.

---

## No flags

No user-facing flags.
