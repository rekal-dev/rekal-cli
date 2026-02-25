# rekal init (bootstrap)

**Role:** The only command a developer must run once per repo. No options, no choices. Sets up everything the system needs.

---

## Preconditions

- Must be run inside a git repository. Otherwise exit with a clear error (e.g. "not a git repository").

---

## Re-run = clean + reinit

If `.rekal/` already exists, treat re-run as **clean then init**: remove `.rekal/`, remove Rekal hooks, then perform a fresh init. So the developer always gets a clean slate when re-running init.

---

## What init does

1. **Resolve git root** — Exit if not in a git repo.
2. **Clean slate (if already initialized)** — If `.rekal/` exists, run `rekal clean`, then continue as fresh init.
3. **Create `.rekal/`** — Directory and whatever the system needs (config, placeholder for data DB path).
5. **Remote branch** — Check if remote has branch `rekal/<user_email>` (or chosen naming). If not, create it with scaffold (e.g. initial commit so the branch exists).
6. **Hooks** — Install two hooks (idempotent; overwrite if present, same Rekal marker):
   - **post-commit** — runs `rekal checkpoint`.
   - **pre-push** — runs `rekal push` when the user runs `git push`; must print status like git while doing it (e.g. "Pushing rekal to rekal/<user>…", progress, "done").
7. **`.gitignore`** — Ensure `.rekal/` is in `.gitignore`; append if missing.
8. **Exit** — Print success message only (e.g. "Rekal initialized.").

---

## No flags

No user-facing flags. Non-interactive only (env-only identity).

---

## Dependency for other commands

All other commands use the **shared preconditions**: they check init is done and that the run is in a git repo. See [preconditions.md](../preconditions.md).
