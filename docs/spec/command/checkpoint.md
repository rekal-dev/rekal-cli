# rekal checkpoint

**Role:** Stateful local operation. Capture the current session after a commit. Invoked by the post-commit hook; can also be run manually. Does not update the index — index is a separate command.

**Invocation:** Subcommand only — `rekal checkpoint`.

---

## Preconditions

See [preconditions.md](../preconditions.md): must be in a git repository and init must have been run. Otherwise exit with the shared message (run init) or warning (not a git repo).

---

## If nothing changed, do not create a new checkpoint

Like git: if there is no new session data to capture since the last checkpoint, do nothing. Do not create a new checkpoint. The implementation must detect “no change” (e.g. same session content, or no new commits since last checkpoint) and skip writing. This keeps checkpoint idempotent when run repeatedly with no new work.

---

## What checkpoint does

1. **Run shared preconditions** — Resolve git root, ensure init is done; exit with shared message/warning if not.
2. **Determine current session** — From session files (e.g. under `~/.claude/projects/` or equivalent; exact source TBD).
3. **Check for change** — If nothing has changed since the last checkpoint (e.g. same content hash or no new commit), exit without writing. Do not create a new checkpoint.
4. **Extract session data** — Turns, tool calls; no tool results / system noise (per storage design).
5. **Write to local state only** — Append to data DB (sessions, checkpoints, files_touched, checkpoint_sessions).
6. **Update index incrementally** — Tell the index layer to index only the new checkpoint and its sessions. The index knows its state and applies only the delta. No full rebuild.
7. **Exit** — Silently or with a single-line status (e.g. "Checkpoint recorded." or nothing if skipped).

---

## Index update on commit

Checkpoint triggers an **incremental** index update after each new checkpoint. The index layer maintains state (what is already indexed) and adds only the new data. Full rebuild (`rekal index`) is for recovery or after sync; see [index.md](index.md).

---

## No flags

No user-facing flags. Same behaviour when invoked by the hook or manually.

---

## Checkpoint ID (orphan branch commit)

The **checkpoint ID** is the commit SHA on the **rekal orphan branch** (e.g. `rekal/<user_email>`), not the main repo's commit. It is created when you run **`rekal push`**: each push commits the Rekal dump to that branch, and that commit's SHA is the checkpoint ID. Commands like `rekal --checkpoint <ref>` and `rekal log --checkpoint <ref>` resolve `<ref>` against this orphan branch (e.g. `HEAD~3`, `@{date}`). The `rekal checkpoint` subcommand only records local state and links it to the **main** repo's current commit; it does not create the checkpoint ID — push does.
