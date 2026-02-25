# rekal push

**Role:** Send local Rekal data to the remote branch. Invoked by the pre-push hook when the user runs `git push`; can also be run manually. Must print status like git while doing it.

**Invocation:** Subcommand only — `rekal push`.

---

## Hooked to git push

`rekal push` is hooked to `git push`: init installs a pre-push hook that runs `rekal push` when the user runs `git push`. So Rekal data is pushed to the remote in the same flow as code. The user can also run `rekal push` manually.

---

## Preconditions

See [preconditions.md](../preconditions.md): must be in a git repository and init must have been run.

---

## What push does

1. **Run shared preconditions** — Resolve git root, ensure init is done; exit with shared message/warning if not.
2. **Prepare local data** — Whatever the system exports for the remote (e.g. SQL dump, rekal.body, per storage design).
3. **Push to remote branch** — Update the Rekal branch on the remote (e.g. `rekal/<user_email>` or configured name). No user prompts; use existing config.
4. **Print status like git** — While pushing, output progress in a git-like style. Prefix lines with `rekal: ` so it's clear they come from Rekal when mixed with git output (e.g. `rekal: Pushing to rekal/<user>…`, `rekal: done`).
5. **Exit** — Success or clear error.

---

## Checkpoint ID = orphan branch commit SHA

Each `rekal push` creates a new commit on the Rekal orphan branch (e.g. `rekal/<user_email>`). That commit's SHA **is** the checkpoint ID. It is what `--checkpoint <ref>` refers to in recall and log (e.g. `rekal "JWT" --checkpoint HEAD~5`). No separate ID is generated — git's ref resolution (SHA, `HEAD~n`, `@{date}`) on the rekal branch is the checkpoint ID.

---

## No flags

No user-facing flags. Same behaviour when invoked by the hook or manually.
