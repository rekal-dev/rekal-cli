# Shared preconditions (all commands)

All commands **except** `rekal init` and `rekal clean` use the same central checks before doing anything.

---

## 1. Git repository

If the current directory is **not** inside a git repository:

- Exit with a clear **warning** (e.g. `Not a git repository. Run from a git repo.`).
- Do not proceed.

So: every command that depends on git resolves the git root first; if that fails, we warn and exit.

---

## 2. Init has been run

If Rekal is **not** initialized (e.g. `.rekal/` does not exist or does not have the expected layout):

- Exit with a single message asking the user to run init (e.g. `Rekal not initialized. Run 'rekal init' in a git repository.`).
- Do not proceed.

So: one central way to “check init is done”; same message everywhere; no command runs its main logic until this passes.

---

## 3. Index DB (for recall, log, query)

Commands that read from the index (recall, query) need `.rekal/index.db` to exist and be usable. Init creates the index DB (and data DB) as part of the expected layout. If the index is missing or empty when one of these commands runs, run the index step (e.g. `rekal index` or equivalent) so the command can proceed. The user never sees "run rekal index first" — index existence is guaranteed by init or by this step.

---

## Commands that use these checks

- **checkpoint**, **push**, **sync**, **index**, **log**, **query**, and **root (recall)** — all require both: in a git repo, and init done.
- **init** — requires only: in a git repo (no “init done” check).
- **clean** — requires only: in a git repo (no “init done” check).

Implementation: one shared helper (e.g. “ensure git root” and “ensure rekal initialized”); each command calls it before doing work.
