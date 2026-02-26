# rekal push

**Role:** Push local Rekal data to the remote branch. Exports unexported checkpoints from DuckDB to wire format, commits to the orphan branch, and pushes to origin.

**Invocation:** `rekal push` or `rekal push --force`.

---

## Preconditions

See [preconditions.md](../preconditions.md): must be in a git repository and init must have been run.

---

## What push does

1. **Run shared preconditions** — Git root, init done.
2. **Check local branch** — Verify the orphan branch (`rekal/<email>`) exists. If not, print "no data to push" and exit.
3. **Check remote** — Verify `origin` is configured. If not, print "no remote configured" and exit.
4. **Export wire format** — Query `data.db` for unexported checkpoints. For each:
   - Encode linked sessions as `SessionFrame` (turns + tool calls, zstd compressed).
   - Encode checkpoint as `CheckpointFrame` (git SHA, files touched, session refs).
   - Append a `MetaFrame` with summary counts.
   - Update string dictionary (`dict.bin`) with session IDs, emails, branches, paths.
   - Mark checkpoints as `exported = TRUE`.
5. **Commit to orphan branch** — Write `rekal.body` and `dict.bin` via `git hash-object` + `git mktree` + `git commit-tree`. Uses the HEAD commit message from the main branch.
6. **Compare with remote** — Skip push if local and remote SHAs match.
7. **Push** — `git push --no-verify origin rekal/<email>`. Handle non-fast-forward with a warning suggesting `--force`.

---

## Flags

| Flag | Description |
|------|-------------|
| `--force`, `-f` | Force push, overwriting the remote branch with local data |

When a normal push is rejected (non-fast-forward), push prints a warning and suggests `rekal push --force`. Force push is safe because each user owns their branch and the local DuckDB is the source of truth.

---

## Hooked to git push

`rekal init` installs a pre-push hook that runs `rekal push` on `git push`. When invoked by the hook, `--force` is not passed — conflicts are reported and resolved on the next manual push.
